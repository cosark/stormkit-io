package authwallhandlers_test

import (
	"context"
	"fmt"
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
)

type HandlerAuthDeleteSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAuthDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerAuthDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerAuthDeleteSuite) Test_Delete_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	aw := &authwall.AuthWall{
		LoginEmail:    "email@example.org",
		LoginPassword: "123pass",
		TeamID:        app.TeamID,
		AppIDs:        []types.ID{app.ID},
	}

	s.NoError(authwall.Store().CreateLogin(context.Background(), aw))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/auth-wall?id=%s", aw.LoginID.String()),
		map[string]any{
			"envId": env.ID.String(),
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	logins, err := authwall.Store().Logins(context.Background(), env.ID)
	s.NoError(err)
	s.Len(logins, 0)
}

func (s *HandlerAuthDeleteSuite) Test_Delete_Success_Multiple() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	aw := &authwall.AuthWall{
		LoginEmail:    "email@example.org",
		LoginPassword: "123pass",
		TeamID:        app.TeamID,
		AppIDs:        []types.ID{app.ID},
	}

	s.NoError(authwall.Store().CreateLogin(context.Background(), aw))

	aw2 := &authwall.AuthWall{
		LoginEmail:    "email-2@example.org",
		LoginPassword: "123pass5",
		TeamID:        app.TeamID,
		AppIDs:        []types.ID{app.ID},
	}

	s.NoError(authwall.Store().CreateLogin(context.Background(), aw2))

	loginIDs := strings.Join([]string{
		aw.LoginID.String(),
		aw2.LoginID.String(),
	}, ",")

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/auth-wall?id=%s", loginIDs),
		map[string]any{
			"envId": env.ID.String(),
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	logins, err := authwall.Store().Logins(context.Background(), env.ID)
	s.NoError(err)
	s.Len(logins, 0)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "DELETE:AUTHWALL",
		EnvName:     env.Name,
		EnvID:       env.ID,
		AppID:       app.ID,
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				AuthWallDeleteLoginIDs: loginIDs,
			},
		},
	}, audits[0])
}

// Test_Delete_KeepsOtherApps verifies that deleting a login from an app's wall
// only takes away its access to that app, and leaves logins of other apps
// alone.
func (s *HandlerAuthDeleteSuite) Test_Delete_KeepsOtherApps() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	otherEnv := s.MockEnv(s.MockApp(usr))
	otherAppID := otherEnv.AppID

	shared := &authwall.AuthWall{
		TeamID:        app.TeamID,
		LoginEmail:    "shared@example.org",
		LoginPassword: "123pass",
		AppIDs:        []types.ID{app.ID, otherAppID},
	}

	other := &authwall.AuthWall{
		TeamID:        app.TeamID,
		LoginEmail:    "other@example.org",
		LoginPassword: "123pass",
		AppIDs:        []types.ID{otherAppID},
	}

	s.NoError(authwall.Store().CreateLogin(context.Background(), shared))
	s.NoError(authwall.Store().CreateLogin(context.Background(), other))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/auth-wall?id=%s,%s", shared.LoginID.String(), other.LoginID.String()),
		map[string]any{
			"envId": env.ID.String(),
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	logins, err := authwall.Store().Logins(context.Background(), env.ID)
	s.NoError(err)
	s.Empty(logins)

	logins, err = authwall.Store().Logins(context.Background(), otherEnv.ID)
	s.NoError(err)
	s.Len(logins, 2)
}

func TestHandlerAuthDeleteSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthDeleteSuite{})
}
