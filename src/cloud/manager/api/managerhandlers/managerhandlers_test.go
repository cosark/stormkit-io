package managerhandlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/identity"
	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/managerhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

// fakeProvider authenticates whoever presents one of its codes.
type fakeProvider struct {
	identities map[string]*identity.Identity
}

func (f *fakeProvider) AuthCodeURL(state string) string {
	return "https://provider.test/authorize?state=" + url.QueryEscape(state)
}

func (f *fakeProvider) Identity(_ context.Context, code string) (*identity.Identity, error) {
	if ident, ok := f.identities[code]; ok {
		return ident, nil
	}

	return nil, errors.New("bad code")
}

// managerSuite holds what every manager handler suite needs: a database
// transaction and a fake GitHub that signs in "octocat" for the code "good".
type managerSuite struct {
	suite.Suite

	conn     databasetest.TestDB
	github   *fakeProvider
	restore  func()
	console  string
	identity *identity.Identity
}

func (s *managerSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.console = "https://console.stormkit.test"
	s.T().Setenv(managerhandlers.ConsoleURLEnv, s.console)

	s.identity = &identity.Identity{
		Provider:       identity.GitHub,
		ProviderUserID: "583231",
		DisplayName:    "The Octocat",
		AvatarURL:      "https://avatars.test/octocat",
		Emails:         []string{"octocat@acme.test"},
	}

	s.github = &fakeProvider{identities: map[string]*identity.Identity{"good": s.identity}}
	s.restore = managerhandlers.SetProviders(map[string]identity.Provider{identity.GitHub: s.github})
}

func (s *managerSuite) AfterTest(_, _ string) {
	s.restore()
	s.conn.CloseTx()
}

func (s *managerSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(managerhandlers.Services).Router().Handler()
}

func (s *managerSuite) request(method, target string, body any, headers map[string]string) shttptest.Response {
	return shttptest.RequestWithHeaders(s.handler(), method, target, body, headers)
}

// login starts a GitHub sign-in and returns the state and its cookie.
func (s *managerSuite) login() (state string, cookie string) {
	response := s.request(shttp.MethodGet, "/cloud/auth/github", nil, nil)
	s.Require().Equal(http.StatusFound, response.Code)

	location, err := url.Parse(response.Header().Get("Location"))
	s.Require().NoError(err)

	cookies := response.Result().Cookies()
	s.Require().Len(cookies, 1)

	return location.Query().Get("state"), cookies[0].Name + "=" + cookies[0].Value
}

// callback completes a sign-in and returns the query the UI receives.
func (s *managerSuite) callback(code string) url.Values {
	state, cookie := s.login()
	target := "/cloud/auth/github/callback?" + url.Values{"state": {state}, "code": {code}}.Encode()
	response := s.request(shttp.MethodGet, target, nil, map[string]string{"Cookie": cookie})

	s.Require().Equal(http.StatusFound, response.Code)

	location := response.Header().Get("Location")
	s.Require().True(strings.HasPrefix(location, s.console+"/auth/callback?"), location)

	parsed, err := url.Parse(location)
	s.Require().NoError(err)

	return parsed.Query()
}

// signIn runs the whole sign-in and returns a session token.
func (s *managerSuite) signIn() string {
	code := s.callback("good").Get("code")
	s.Require().NotEmpty(code)

	response := s.request(shttp.MethodPost, "/cloud/auth/session", map[string]string{"code": code}, nil)
	s.Require().Equal(http.StatusOK, response.Code)

	token, _ := response.Map()["token"].(string)
	s.Require().NotEmpty(token)

	return token
}

func (s *managerSuite) bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}
