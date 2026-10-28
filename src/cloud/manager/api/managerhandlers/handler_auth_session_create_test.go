package managerhandlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/account"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthSessionCreateSuite struct {
	managerSuite
}

func (s *HandlerAuthSessionCreateSuite) Test_Success() {
	code := s.callback("good").Get("code")
	response := s.request(shttp.MethodPost, "/cloud/auth/session", map[string]string{"code": code}, nil)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()
	token, _ := body["token"].(string)
	acc, _ := body["account"].(map[string]any)

	s.True(strings.HasPrefix(token, "skc_"))
	s.NotZero(body["expiresAt"])
	s.Equal("The Octocat", acc["displayName"])
	s.Equal([]any{"octocat@acme.test"}, acc["emails"])
}

// Test_StoresOnlyTheTokenHash verifies that a leaked sessions table does not
// hand out working tokens.
func (s *HandlerAuthSessionCreateSuite) Test_StoresOnlyTheTokenHash() {
	token := s.signIn()

	var count int
	err := s.conn.QueryRow("SELECT count(*) FROM cloud.sessions WHERE token_hash = $1", token).Scan(&count)
	s.Require().NoError(err)
	s.Zero(count)

	acc, err := account.NewStore().SessionAccount(context.Background(), token)
	s.Require().NoError(err)
	s.Nil(acc)
}

func (s *HandlerAuthSessionCreateSuite) Test_CodeIsSingleUse() {
	code := s.callback("good").Get("code")

	first := s.request(shttp.MethodPost, "/cloud/auth/session", map[string]string{"code": code}, nil)
	second := s.request(shttp.MethodPost, "/cloud/auth/session", map[string]string{"code": code}, nil)

	s.Equal(http.StatusOK, first.Code)
	s.Equal(http.StatusUnauthorized, second.Code)
}

func (s *HandlerAuthSessionCreateSuite) Test_UnknownCode() {
	response := s.request(shttp.MethodPost, "/cloud/auth/session", map[string]string{"code": "made-up"}, nil)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerAuthSessionCreateSuite) Test_MissingCode() {
	response := s.request(shttp.MethodPost, "/cloud/auth/session", map[string]string{}, nil)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerAuthSessionCreateSuite(t *testing.T) {
	suite.Run(t, new(HandlerAuthSessionCreateSuite))
}
