package managerhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthSessionDeleteSuite struct {
	managerSuite
}

func (s *HandlerAuthSessionDeleteSuite) Test_SignsOutThisSessionOnly() {
	signedOut := s.signIn()
	other := s.signIn()

	response := s.request(shttp.MethodDelete, "/cloud/auth/session", nil, s.bearer(signedOut))
	s.Equal(http.StatusOK, response.Code)

	s.Equal(http.StatusUnauthorized, s.request(shttp.MethodGet, "/cloud/account", nil, s.bearer(signedOut)).Code)
	s.Equal(http.StatusOK, s.request(shttp.MethodGet, "/cloud/account", nil, s.bearer(other)).Code)
}

func (s *HandlerAuthSessionDeleteSuite) Test_Unauthorized() {
	response := s.request(shttp.MethodDelete, "/cloud/auth/session", nil, nil)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerAuthSessionDeleteSuite(t *testing.T) {
	suite.Run(t, new(HandlerAuthSessionDeleteSuite))
}
