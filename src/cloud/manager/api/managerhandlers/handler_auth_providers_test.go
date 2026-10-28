package managerhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthProvidersSuite struct {
	managerSuite
}

func (s *HandlerAuthProvidersSuite) Test_ListsConfiguredProviders() {
	response := s.request(shttp.MethodGet, "/cloud/auth/providers", nil, nil)

	s.Equal(http.StatusOK, response.Code)
	s.Equal([]any{"github"}, response.Map()["providers"])
}

func TestHandlerAuthProvidersSuite(t *testing.T) {
	suite.Run(t, new(HandlerAuthProvidersSuite))
}
