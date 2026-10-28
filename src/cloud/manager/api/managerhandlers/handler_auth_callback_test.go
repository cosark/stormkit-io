package managerhandlers_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/account"
	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/identity"
	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/managerhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthCallbackSuite struct {
	managerSuite
}

func (s *HandlerAuthCallbackSuite) Test_Success_RedirectsWithLoginCode() {
	query := s.callback("good")

	s.NotEmpty(query.Get("code"))
	s.Empty(query.Get("error"))
}

func (s *HandlerAuthCallbackSuite) Test_Success_ClearsStateCookie() {
	state, cookie := s.login()
	target := "/cloud/auth/github/callback?" + url.Values{"state": {state}, "code": {"good"}}.Encode()
	response := s.request(shttp.MethodGet, target, nil, map[string]string{"Cookie": cookie})

	cookies := response.Result().Cookies()
	s.Require().Len(cookies, 1)
	s.Equal(-1, cookies[0].MaxAge)
}

// Test_StateWithoutCookie verifies that a callback the browser did not start,
// such as a link someone else sends, signs nobody in.
func (s *HandlerAuthCallbackSuite) Test_StateWithoutCookie() {
	state, _ := s.login()
	target := "/cloud/auth/github/callback?" + url.Values{"state": {state}, "code": {"good"}}.Encode()
	response := s.request(shttp.MethodGet, target, nil, nil)

	s.Equal(http.StatusFound, response.Code)
	s.Contains(response.Header().Get("Location"), "error=state")
	s.NotContains(response.Header().Get("Location"), "code=")
}

func (s *HandlerAuthCallbackSuite) Test_StateMismatch() {
	_, cookie := s.login()
	target := "/cloud/auth/github/callback?" + url.Values{"state": {"forged"}, "code": {"good"}}.Encode()
	response := s.request(shttp.MethodGet, target, nil, map[string]string{"Cookie": cookie})

	s.Contains(response.Header().Get("Location"), "error=state")
}

// Test_CookieOfAnotherProvider verifies that a state started with one
// provider cannot complete a sign-in with another.
func (s *HandlerAuthCallbackSuite) Test_CookieOfAnotherProvider() {
	gitlab := &fakeProvider{identities: map[string]*identity.Identity{"good": s.identity}}
	s.restore()
	s.restore = managerhandlers.SetProviders(map[string]identity.Provider{identity.GitHub: s.github, identity.GitLab: gitlab})

	state, cookie := s.login()
	target := "/cloud/auth/gitlab/callback?" + url.Values{"state": {state}, "code": {"good"}}.Encode()
	response := s.request(shttp.MethodGet, target, nil, map[string]string{"Cookie": cookie})

	s.Contains(response.Header().Get("Location"), "error=state")
}

func (s *HandlerAuthCallbackSuite) Test_Denied() {
	state, cookie := s.login()
	target := "/cloud/auth/github/callback?" + url.Values{"state": {state}, "error": {"access_denied"}}.Encode()
	response := s.request(shttp.MethodGet, target, nil, map[string]string{"Cookie": cookie})

	s.Contains(response.Header().Get("Location"), "error=denied")
}

func (s *HandlerAuthCallbackSuite) Test_ProviderRejectsCode() {
	s.Equal("provider", s.callback("bad").Get("error"))
}

func (s *HandlerAuthCallbackSuite) Test_NoVerifiedEmail() {
	s.identity.Emails = []string{}

	s.Equal("email", s.callback("good").Get("error"))
}

func (s *HandlerAuthCallbackSuite) Test_AmbiguousEmails() {
	store := account.NewStore()

	for _, p := range []account.SignInParams{
		{Provider: identity.Google, ProviderUserID: "1", Emails: []string{"octocat@acme.test"}},
		{Provider: identity.Google, ProviderUserID: "2", Emails: []string{"octo@work.test"}},
	} {
		_, err := store.SignIn(context.Background(), p)
		s.Require().NoError(err)
	}

	s.identity.Emails = []string{"octocat@acme.test", "octo@work.test"}

	s.Equal("ambiguous", s.callback("good").Get("error"))
}

func TestHandlerAuthCallbackSuite(t *testing.T) {
	suite.Run(t, new(HandlerAuthCallbackSuite))
}
