package identity_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/identity"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type IdentitySuite struct {
	suite.Suite

	server *httptest.Server
	routes map[string]any
}

func (s *IdentitySuite) SetupTest() {
	s.routes = map[string]any{}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			s.Equal("the-code", r.FormValue("code"))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "the-token", "token_type": "bearer"})
			return
		}

		s.Equal("Bearer the-token", r.Header.Get("Authorization"))

		body, ok := s.routes[r.URL.Path]

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(body)
	}))
}

func (s *IdentitySuite) TearDownTest() {
	s.server.Close()
}

func (s *IdentitySuite) config() *oauth2.Config {
	return &oauth2.Config{ClientID: "id", ClientSecret: "secret", RedirectURL: "https://api.test/callback"}
}

func (s *IdentitySuite) endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{AuthURL: s.server.URL + "/authorize", TokenURL: s.server.URL + "/token"}
}

func (s *IdentitySuite) Test_GitHub_KeepsOnlyVerifiedEmails() {
	s.routes["/user"] = map[string]any{"id": 42, "login": "jane", "name": "", "avatar_url": "https://avatars.test/jane"}
	s.routes["/user/emails"] = []map[string]any{
		{"email": "Jane@Acme.test", "verified": true, "primary": true},
		{"email": "jane@unverified.test", "verified": false},
		{"email": "42+jane@users.noreply.github.com", "verified": true},
	}

	github := identity.NewGitHub(s.config())
	github.Config.Endpoint = s.endpoint()
	github.APIURL = s.server.URL

	ident, err := github.Identity(context.Background(), "the-code")
	s.Require().NoError(err)

	s.Equal(&identity.Identity{
		Provider:       identity.GitHub,
		ProviderUserID: "42",
		DisplayName:    "jane",
		AvatarURL:      "https://avatars.test/jane",
		Emails:         []string{"jane@acme.test"},
	}, ident)
}

func (s *IdentitySuite) Test_GitHub_ApiError() {
	github := identity.NewGitHub(s.config())
	github.Config.Endpoint = s.endpoint()
	github.APIURL = s.server.URL

	_, err := github.Identity(context.Background(), "the-code")
	s.Error(err)
}

func (s *IdentitySuite) Test_GitLab_KeepsOnlyConfirmedEmails() {
	s.routes["/user"] = map[string]any{
		"id": 7, "username": "jane", "name": "Jane Doe", "avatar_url": "https://avatars.test/jane",
		"email": "jane@acme.test", "confirmed_at": "2024-01-01T00:00:00Z",
	}
	s.routes["/user/emails"] = []map[string]any{
		{"email": "jane@home.test", "confirmed_at": "2024-02-01T00:00:00Z"},
		{"email": "jane@pending.test", "confirmed_at": nil},
	}

	gitlab := identity.NewGitLab(s.config())
	gitlab.Config.Endpoint = s.endpoint()
	gitlab.APIURL = s.server.URL

	ident, err := gitlab.Identity(context.Background(), "the-code")
	s.Require().NoError(err)

	s.Equal("7", ident.ProviderUserID)
	s.Equal("Jane Doe", ident.DisplayName)
	s.Equal([]string{"jane@acme.test", "jane@home.test"}, ident.Emails)
}

func (s *IdentitySuite) Test_GitLab_UnconfirmedPrimaryEmail() {
	s.routes["/user"] = map[string]any{"id": 7, "username": "jane", "email": "jane@acme.test"}
	s.routes["/user/emails"] = []map[string]any{}

	gitlab := identity.NewGitLab(s.config())
	gitlab.Config.Endpoint = s.endpoint()
	gitlab.APIURL = s.server.URL

	ident, err := gitlab.Identity(context.Background(), "the-code")
	s.Require().NoError(err)

	s.Empty(ident.Emails)
}

func (s *IdentitySuite) Test_Google() {
	s.routes["/userinfo"] = map[string]any{
		"sub": "1098", "name": "Jane Doe", "picture": "https://avatars.test/jane",
		"email": "jane@gmail.test", "email_verified": true,
	}

	google := identity.NewGoogle(s.config())
	google.Config.Endpoint = s.endpoint()
	google.UserInfoURL = s.server.URL + "/userinfo"

	ident, err := google.Identity(context.Background(), "the-code")
	s.Require().NoError(err)

	s.Equal("1098", ident.ProviderUserID)
	s.Equal([]string{"jane@gmail.test"}, ident.Emails)
}

func (s *IdentitySuite) Test_Google_UnverifiedEmail() {
	s.routes["/userinfo"] = map[string]any{"sub": "1098", "email": "jane@gmail.test", "email_verified": false}

	google := identity.NewGoogle(s.config())
	google.Config.Endpoint = s.endpoint()
	google.UserInfoURL = s.server.URL + "/userinfo"

	ident, err := google.Identity(context.Background(), "the-code")
	s.Require().NoError(err)

	s.Empty(ident.Emails)
}

func (s *IdentitySuite) Test_Providers_OnlyConfiguredOnes() {
	s.T().Setenv("STORMKIT_CLOUD_GITHUB_CLIENT_ID", "id")
	s.T().Setenv("STORMKIT_CLOUD_GITHUB_CLIENT_SECRET", "secret")
	s.T().Setenv("STORMKIT_CLOUD_GOOGLE_CLIENT_ID", "id")
	s.T().Setenv("STORMKIT_CLOUD_GOOGLE_CLIENT_SECRET", "")
	s.T().Setenv("STORMKIT_CLOUD_GITLAB_CLIENT_ID", "")

	providers := identity.Providers(identity.ProvidersParams{
		CallbackURL: func(provider string) string { return "https://api.test/cloud/auth/" + provider + "/callback" },
	})

	s.Equal([]string{"github"}, identity.Names(providers))
	s.Contains(providers["github"].AuthCodeURL("xyz"), "redirect_uri=https%3A%2F%2Fapi.test%2Fcloud%2Fauth%2Fgithub%2Fcallback")
}

func TestIdentitySuite(t *testing.T) {
	suite.Run(t, new(IdentitySuite))
}
