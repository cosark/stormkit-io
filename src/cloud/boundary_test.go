package cloud_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

const (
	cloudPkg   = "github.com/stormkit-io/stormkit-io/src/cloud"
	routerPkg  = "github.com/stormkit-io/stormkit-io/src/ce/api/router"
	managerPkg = "github.com/stormkit-io/stormkit-io/src/cloud/manager/api/managerhandlers"
)

// BoundarySuite keeps Stormkit Cloud's control-plane code out of the product.
// The API router is the one exception: it registers the instance manager's
// endpoints, which only run on Stormkit Cloud builds.
type BoundarySuite struct {
	suite.Suite
}

func (s *BoundarySuite) Test_OnlyTheRouterImportsCloud() {
	gomod, err := exec.Command("go", "env", "GOMOD").Output()
	s.Require().NoError(err)

	cmd := exec.Command("go", "list", "-tags=imageopt,alibaba", "-f", "{{.ImportPath}}{{range .Imports}} {{.}}{{end}}",
		"./src/ce/...", "./src/ee/...", "./src/lib/...", "./src/migrations/...")
	cmd.Dir = filepath.Dir(strings.TrimSpace(string(gomod)))

	out, err := cmd.Output()
	s.Require().NoError(err)

	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		pkg, imports := fields[0], fields[1:]

		for _, imp := range imports {
			if imp != cloudPkg && !strings.HasPrefix(imp, cloudPkg+"/") {
				continue
			}

			s.True(pkg == routerPkg && imp == managerPkg, "%s must not import %s", pkg, imp)
		}
	}
}

func TestBoundarySuite(t *testing.T) {
	suite.Run(t, new(BoundarySuite))
}
