package authwallhandlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type HandlerAuthCreateSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAuthCreateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerAuthCreateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

// create posts a new login to the wall of env.
func (s *HandlerAuthCreateSuite) create(usr *factory.MockUser, env *factory.MockEnv, body map[string]any) shttptest.Response {
	body["envId"] = env.ID.String()

	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall",
		body,
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)
}

func (s *HandlerAuthCreateSuite) Test_AuthCreate_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := s.create(usr, env, map[string]any{
		"email":    "my-email@example.org",
		"password": "my-paSword",
		"metadata": `{"tenant":"acme"}`,
	})

	s.Equal(http.StatusOK, response.Code)

	logins, err := authwall.Store().Logins(context.Background(), env.ID)
	s.NoError(err)
	s.Len(logins, 1)
	s.Equal("my-email@example.org", logins[0].LoginEmail)
	s.Equal(`{"tenant":"acme"}`, logins[0].Metadata)
	s.Equal(app.TeamID, logins[0].TeamID)
	s.False(logins[0].AllApps)

	var hash string
	s.Require().NoError(s.conn.QueryRow("SELECT login_password FROM auth_wall WHERE login_id = $1", logins[0].LoginID).Scan(&hash))
	s.NoError(bcrypt.CompareHashAndPassword([]byte(hash), []byte("my-paSword")))

	other, err := authwall.Store().Logins(context.Background(), s.MockEnv(s.MockApp(usr)).ID)
	s.NoError(err)
	s.Empty(other)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "CREATE:AUTHWALL",
		EnvName:     env.Name,
		EnvID:       env.ID,
		AppID:       app.ID,
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			New: audit.DiffFields{
				AuthWallCreateLoginEmail: "my-email@example.org",
				AuthWallCreateLoginID:    logins[0].LoginID.String(),
			},
		},
	}, audits[0])
}

// existingLogin creates a login for email that opens the wall of app.
func (s *HandlerAuthCreateSuite) existingLogin(app *factory.MockApp, email string) {
	s.Require().NoError(authwall.Store().CreateLogin(context.Background(), &authwall.AuthWall{
		TeamID:        app.TeamID,
		LoginEmail:    email,
		LoginPassword: "123pass",
		AppIDs:        []types.ID{app.ID},
	}))
}

// Test_AuthCreate_FailDuplicate verifies that an email is unique within the
// team, regardless of its case.
func (s *HandlerAuthCreateSuite) Test_AuthCreate_FailDuplicate() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	s.existingLogin(app, "my-email@example.org")

	response := s.create(usr, s.MockEnv(app), map[string]any{
		"email":    "MY-EMAIL@example.org",
		"password": "my-paSword",
	})

	s.Equal(http.StatusConflict, response.Code)
}

// Test_AuthCreate_FailDuplicate_OtherApp verifies that an email is unique
// within the team, not within the app.
func (s *HandlerAuthCreateSuite) Test_AuthCreate_FailDuplicate_OtherApp() {
	usr := s.MockUser()
	s.existingLogin(s.MockApp(usr), "my-email@example.org")

	response := s.create(usr, s.MockEnv(s.MockApp(usr)), map[string]any{
		"email":    "my-email@example.org",
		"password": "my-paSword",
	})

	s.Equal(http.StatusConflict, response.Code)
}

// Test_AuthCreate_Success_OtherTeam verifies that another team can use an
// email that is already taken.
func (s *HandlerAuthCreateSuite) Test_AuthCreate_Success_OtherTeam() {
	s.existingLogin(s.MockApp(s.MockUser()), "my-email@example.org")

	usr := s.MockUser()
	response := s.create(usr, s.MockEnv(s.MockApp(usr)), map[string]any{
		"email":    "my-email@example.org",
		"password": "my-paSword",
	})

	s.Equal(http.StatusOK, response.Code)
}

// Test_EnvTeamID_NoTeam verifies that an app without a team, whose owner has
// no personal team either, resolves to no team instead of failing.
func (s *HandlerAuthCreateSuite) Test_EnvTeamID_NoTeam() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	_, err := s.conn.Exec("UPDATE apps SET team_id = NULL WHERE app_id = $1", app.ID)
	s.Require().NoError(err)

	teamID, err := authwall.Store().EnvTeamID(context.Background(), env.ID)
	s.NoError(err)
	s.Equal(usr.DefaultTeamID, teamID, "falls back to the owner's personal team")

	_, err = s.conn.Exec("UPDATE teams SET is_default = FALSE WHERE user_id = $1", usr.ID)
	s.Require().NoError(err)

	teamID, err = authwall.Store().EnvTeamID(context.Background(), env.ID)
	s.NoError(err)
	s.Zero(teamID)
}

func (s *HandlerAuthCreateSuite) Test_AuthCreate_FailMetadata() {
	usr := s.MockUser()
	env := s.MockEnv(s.MockApp(usr))

	for _, metadata := range []string{"line\nbreak", strings.Repeat("a", authwall.MetadataMaxLength+1)} {
		response := s.create(usr, env, map[string]any{
			"email":    "my-email@example.org",
			"password": "my-paSword",
			"metadata": metadata,
		})

		s.Equal(http.StatusBadRequest, response.Code)
	}
}

func (s *HandlerAuthCreateSuite) Test_AuthCreate_FailPasswordTooLong() {
	usr := s.MockUser()

	response := s.create(usr, s.MockEnv(s.MockApp(usr)), map[string]any{
		"email":    "my-email@example.org",
		"password": strings.Repeat("a", 73),
	})

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerAuthCreateSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthCreateSuite{})
}
