package managerhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HandlerAccountGetSuite struct {
	managerSuite
}

func (s *HandlerAccountGetSuite) Test_Success() {
	response := s.request(shttp.MethodGet, "/cloud/account", nil, s.bearer(s.signIn()))

	s.Equal(http.StatusOK, response.Code)

	acc, _ := response.Map()["account"].(map[string]any)
	s.Equal("The Octocat", acc["displayName"])
	s.Equal("https://avatars.test/octocat", acc["avatarUrl"])
}

func (s *HandlerAccountGetSuite) Test_Unauthorized_NoToken() {
	s.Equal(http.StatusUnauthorized, s.request(shttp.MethodGet, "/cloud/account", nil, nil).Code)
}

func (s *HandlerAccountGetSuite) Test_Unauthorized_UnknownToken() {
	response := s.request(shttp.MethodGet, "/cloud/account", nil, s.bearer("skc_made-up"))

	s.Equal(http.StatusUnauthorized, response.Code)
}

// Test_Unauthorized_ExpiredSession verifies that a session stops working once
// it expires, even though its row is still there.
func (s *HandlerAccountGetSuite) Test_Unauthorized_ExpiredSession() {
	token := s.signIn()

	_, err := s.conn.Exec("UPDATE cloud.sessions SET expires_at = now() - interval '1 minute'")
	s.Require().NoError(err)

	s.Equal(http.StatusUnauthorized, s.request(shttp.MethodGet, "/cloud/account", nil, s.bearer(token)).Code)
}

func TestHandlerAccountGetSuite(t *testing.T) {
	suite.Run(t, new(HandlerAccountGetSuite))
}
