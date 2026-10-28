package router_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/router"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type RouterSuite struct {
	suite.Suite

	conn databasetest.TestDB
}

func (s *RouterSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
}

func (s *RouterSuite) AfterTest(_, _ string) {
	config.SetIsStormkitCloud(false)
	admin.ResetMockLicense()
	router.AllowedHosts = []string{}
	s.conn.CloseTx()
}

// Test_ManagerRoutes_CloudOnly verifies that self-hosted instances do not
// serve the instance manager's endpoints.
func (s *RouterSuite) Test_ManagerRoutes_CloudOnly() {
	config.SetIsSelfHosted(true)
	response := shttptest.Request(router.Get().Handler(), shttp.MethodGet, "/cloud/auth/providers", nil)
	s.Equal(http.StatusNotFound, response.Code)

	config.SetIsStormkitCloud(true)
	response = shttptest.Request(router.Get().Handler(), shttp.MethodGet, "/cloud/auth/providers", nil)
	s.Equal(http.StatusOK, response.Code)
}

func TestRouterSuite(t *testing.T) {
	suite.Run(t, new(RouterSuite))
}
