package skauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type ClientGitLabSuite struct {
	suite.Suite

	server         *httptest.Server
	originalBase   string
	userInfo       string
	userInfoStatus int
}

func (s *ClientGitLabSuite) SetupTest() {
	s.originalBase = skauth.GitLabBaseURL
	s.userInfoStatus = http.StatusOK
	s.userInfo = `{
		"sub": "1234567",
		"name": "Tanuki",
		"nickname": "tanuki",
		"email": "tanuki@home.test",
		"email_verified": true,
		"picture": "https://avatars.test/tanuki",
		"profile": "https://gitlab.com/tanuki"
	}`

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("Bearer test-access-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/oauth/userinfo" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(s.userInfoStatus)
		w.Write([]byte(s.userInfo))
	}))

	skauth.GitLabBaseURL = s.server.URL
}

func (s *ClientGitLabSuite) TearDownTest() {
	skauth.GitLabBaseURL = s.originalBase
	s.server.Close()
}

func (s *ClientGitLabSuite) client() skauth.Client {
	return skauth.NewGitLabClient(skauth.GitLabClientParams{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://app.example.com/_stormkit/auth/callback",
	})
}

func (s *ClientGitLabSuite) fetchUserInfo() (*skauth.UserInfo, error) {
	return s.client().UserInfo(context.Background(), &oauth2.Token{AccessToken: "test-access-token", TokenType: "Bearer"})
}

func (s *ClientGitLabSuite) Test_UserInfo() {
	info, err := s.fetchUserInfo()

	s.Require().NoError(err)
	s.Equal("1234567", info.AccountID)
	s.Equal("tanuki@home.test", info.Email)
	s.True(info.EmailVerified)
	s.Equal("Tanuki", info.FirstName)
	s.Equal("https://avatars.test/tanuki", info.Avatar)
	s.Equal("tanuki", info.Username)
	s.Equal("https://gitlab.com/tanuki", info.ProfileURL)
}

// Test_UserInfo_UnverifiedEmail verifies that GitLab's verification flag is
// passed on, so the callback refuses the sign-in.
func (s *ClientGitLabSuite) Test_UserInfo_UnverifiedEmail() {
	s.userInfo = `{"sub": "1234567", "nickname": "tanuki", "email": "tanuki@home.test", "email_verified": false}`

	info, err := s.fetchUserInfo()

	s.Require().NoError(err)
	s.False(info.EmailVerified)
	s.Equal("tanuki", info.FirstName)
}

func (s *ClientGitLabSuite) Test_UserInfo_InvalidJSON() {
	s.userInfo = `not json`

	info, err := s.fetchUserInfo()

	s.Error(err)
	s.Nil(info)
}

// Test_UserInfo_MissingScope verifies that an application without the openid
// scope fails with an error naming the fix.
func (s *ClientGitLabSuite) Test_UserInfo_MissingScope() {
	s.userInfoStatus = http.StatusForbidden
	s.userInfo = `{"error": "insufficient_scope"}`

	info, err := s.fetchUserInfo()

	s.Require().Error(err)
	s.Contains(err.Error(), "403")
	s.Contains(err.Error(), "openid")
	s.Nil(info)
}

func (s *ClientGitLabSuite) Test_AuthCodeURL() {
	skauth.GitLabBaseURL = "https://gitlab.com"

	authURL, err := s.client().AuthCodeURL(skauth.AuthCodeURLParams{EnvID: 1, ProviderName: skauth.ProviderGitLab, Secret: stateSecret})
	s.Require().NoError(err)

	parsed, err := url.Parse(authURL)
	s.Require().NoError(err)
	s.Equal("gitlab.com", parsed.Host)
	s.Equal("/oauth/authorize", parsed.Path)
	s.Equal("client-id", parsed.Query().Get("client_id"))
	s.Equal("https://app.example.com/_stormkit/auth/callback", parsed.Query().Get("redirect_uri"))
	s.Equal("openid profile email", parsed.Query().Get("scope"))

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:   parsed.Query().Get("state"),
		Secret:   stateSecret,
		Purposes: []user.Purpose{user.PurposeSkAuthOAuthState},
	})

	s.Equal(skauth.ProviderGitLab, claims["prv"])
}

func (s *ClientGitLabSuite) Test_ProviderClient() {
	p := &skauth.Provider{Name: skauth.ProviderGitLab}

	s.True(p.Supported())
	s.IsType(&skauth.GitLabClient{}, p.Client(skauth.ClientParams{RedirectURL: "https://app.example.com/_stormkit/auth/callback"}))
}

func TestClientGitLabSuite(t *testing.T) {
	suite.Run(t, new(ClientGitLabSuite))
}
