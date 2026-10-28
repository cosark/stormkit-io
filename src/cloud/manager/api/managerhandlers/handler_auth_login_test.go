package managerhandlers_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthLoginSuite struct {
	managerSuite
}

// Test_RedirectsWithStateCookie verifies that the state sent to the provider
// is also kept in an HttpOnly cookie scoped to the auth endpoints.
func (s *HandlerAuthLoginSuite) Test_RedirectsWithStateCookie() {
	response := s.request(shttp.MethodGet, "/cloud/auth/github", nil, nil)

	s.Equal(http.StatusFound, response.Code)
	s.True(strings.HasPrefix(response.Header().Get("Location"), "https://provider.test/authorize?state="))

	cookies := response.Result().Cookies()
	s.Require().Len(cookies, 1)
	s.True(cookies[0].HttpOnly)
	s.Equal(http.SameSiteLaxMode, cookies[0].SameSite)
	s.Equal("/cloud/auth", cookies[0].Path)
	s.True(strings.HasPrefix(cookies[0].Value, "github."))
}

func (s *HandlerAuthLoginSuite) Test_EachLoginGetsItsOwnState() {
	first, _ := s.login()
	second, _ := s.login()

	s.NotEqual(first, second)
}

func (s *HandlerAuthLoginSuite) Test_NotFound_UnconfiguredProvider() {
	response := s.request(shttp.MethodGet, "/cloud/auth/gitlab", nil, nil)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerAuthLoginSuite(t *testing.T) {
	suite.Run(t, new(HandlerAuthLoginSuite))
}
