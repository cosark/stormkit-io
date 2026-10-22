package teamhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team/teamhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamsMigrateAppSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamsMigrateAppSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamsMigrateAppSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamsMigrateAppSuite) Test_Success() {
	user := s.MockUser()
	myApp := s.MockApp(user)
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: user.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/migrate",
		map[string]string{
			"appId":  myApp.ID.String(),
			"teamId": newTeam.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(user.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	updatedApp, err := app.NewStore().AppByID(context.Background(), myApp.ID)
	s.NoError(err)
	s.Equal(newTeam.ID, updatedApp.TeamID)
}

// login creates an Auth Wall login in the team.
func (s *HandlerTeamsMigrateAppSuite) login(aw *authwall.AuthWall) *authwall.AuthWall {
	if aw.LoginPassword == "" {
		aw.LoginPassword = "123pass"
	}

	s.Require().NoError(authwall.Store().CreateLogin(context.Background(), aw))

	return aw
}

// openWall reports whether the credentials open the wall of the environment.
func (s *HandlerTeamsMigrateAppSuite) openWall(envID types.ID, email, password string) bool {
	aw, err := authwall.Store().Login(context.Background(), authwall.LoginParams{
		EnvID:    envID,
		Email:    email,
		Password: password,
	})

	s.Require().NoError(err)

	return aw != nil
}

// Test_Success_MovesAuthWallLogins verifies that the logins opening an app's
// Auth Wall move with the app, without taking the old team's other apps along.
func (s *HandlerTeamsMigrateAppSuite) Test_Success_MovesAuthWallLogins() {
	user := s.MockUser()
	myApp := s.MockApp(user)
	otherApp := s.MockApp(user)
	env := s.MockEnv(myApp)
	otherEnv := s.MockEnv(otherApp)
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: user.ID, Role: team.ROLE_OWNER, Status: true}

	s.Require().NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))

	only := s.login(&authwall.AuthWall{TeamID: myApp.TeamID, LoginEmail: "only@example.org", AppIDs: []types.ID{myApp.ID}})
	shared := s.login(&authwall.AuthWall{TeamID: myApp.TeamID, LoginEmail: "shared@example.org", AppIDs: []types.ID{myApp.ID, otherApp.ID}})
	all := s.login(&authwall.AuthWall{TeamID: myApp.TeamID, LoginEmail: "all@example.org", AllApps: true})
	s.login(&authwall.AuthWall{TeamID: myApp.TeamID, LoginEmail: "dup@example.org", LoginPassword: "old-team", AppIDs: []types.ID{myApp.ID}})
	s.login(&authwall.AuthWall{TeamID: newTeam.ID, LoginEmail: "dup@example.org", LoginPassword: "new-team"})
	unrelated := s.login(&authwall.AuthWall{TeamID: myApp.TeamID, LoginEmail: "unrelated@example.org", AppIDs: []types.ID{otherApp.ID}})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/migrate",
		map[string]string{
			"appId":  myApp.ID.String(),
			"teamId": newTeam.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(user.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	for _, aw := range []*authwall.AuthWall{only, shared, all} {
		s.True(s.openWall(env.ID, aw.LoginEmail, "123pass"), aw.LoginEmail)
	}

	// The login already in the new team keeps its password.
	s.True(s.openWall(env.ID, "dup@example.org", "new-team"))
	s.False(s.openWall(env.ID, "dup@example.org", "old-team"))
	s.False(s.openWall(env.ID, unrelated.LoginEmail, "123pass"))

	// The old team keeps access to its other apps; the login that only opened
	// the moved app is gone from it.
	s.True(s.openWall(otherEnv.ID, shared.LoginEmail, "123pass"))
	s.True(s.openWall(otherEnv.ID, all.LoginEmail, "123pass"))
	s.True(s.openWall(otherEnv.ID, unrelated.LoginEmail, "123pass"))

	var count int
	s.Require().NoError(s.conn.QueryRow("SELECT COUNT(*) FROM auth_wall WHERE login_id = $1", only.LoginID).Scan(&count))
	s.Zero(count)
}

func (s *HandlerTeamsMigrateAppSuite) Test_FailPermission() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	myApp := s.MockApp(usr2)
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: usr1.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))
	s.NoError(team.NewStore().AddMemberToTeam(context.Background(), &team.Member{
		TeamID: newTeam.ID,
		UserID: usr2.ID,
		Role:   team.ROLE_DEVELOPER,
		Status: true,
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/migrate",
		map[string]string{
			"appId":  myApp.ID.String(),
			"teamId": newTeam.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr2.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

// Test_FailNotMemberOfSourceTeam verifies that an app cannot be moved into the
// caller's team unless the caller has write access to the app's current team.
func (s *HandlerTeamsMigrateAppSuite) Test_FailNotMemberOfSourceTeam() {
	victim := s.MockUser()
	victimApp := s.MockApp(victim)

	attacker := s.MockUser()
	attackerTeam := team.Team{Name: "Attacker Team"}
	member := team.Member{UserID: attacker.ID, Role: team.ROLE_OWNER, Status: true}
	s.Require().NoError(team.NewStore().CreateTeam(context.Background(), &attackerTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/migrate",
		map[string]string{
			"appId":  victimApp.ID.String(),
			"teamId": attackerTeam.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(attacker.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)

	unchanged, err := app.NewStore().AppByID(context.Background(), victimApp.ID)
	s.Require().NoError(err)
	s.Equal(victimApp.TeamID, unchanged.TeamID)
}

func TestHandlerTeamsMigrateApp(t *testing.T) {
	suite.Run(t, &HandlerTeamsMigrateAppSuite{})
}
