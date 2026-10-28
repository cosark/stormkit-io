package cloud_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

const (
	cloudPkg      = "github.com/stormkit-io/stormkit-io/src/cloud"
	routerPkg     = "github.com/stormkit-io/stormkit-io/src/ce/api/router"
	managerPkg    = "github.com/stormkit-io/stormkit-io/src/cloud/manager/api/managerhandlers"
	managerAPIPkg = "github.com/stormkit-io/stormkit-io/src/cloud/manager/api/"
)

// BoundarySuite keeps Stormkit Cloud's control-plane code out of the product.
// The API router is the one exception: it registers the instance manager's
// endpoints, which only run on Stormkit Cloud builds.
type BoundarySuite struct {
	suite.Suite
}

// goList runs go list over the product packages with the given extra flags.
func (s *BoundarySuite) goList(flags ...string) []string {
	gomod, err := exec.Command("go", "env", "GOMOD").Output()
	s.Require().NoError(err)

	args := append([]string{"list", "-tags=imageopt,alibaba"}, flags...)
	args = append(args, "./src/ce/...", "./src/ee/...", "./src/lib/...", "./src/migrations/...")

	cmd := exec.Command("go", args...)
	cmd.Dir = filepath.Dir(strings.TrimSpace(string(gomod)))

	out, err := cmd.Output()
	s.Require().NoError(err)

	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

func (s *BoundarySuite) Test_OnlyTheRouterImportsCloud() {
	for _, line := range s.goList("-f", "{{.ImportPath}}{{range .Imports}} {{.}}{{end}}") {
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

// Test_ProductCompilesOnlyTheManagerAPI verifies that the manager does not
// pull the rest of the control plane, such as the provisioners and their
// credentials handling, into self-hosted builds.
func (s *BoundarySuite) Test_ProductCompilesOnlyTheManagerAPI() {
	for _, pkg := range s.goList("-deps", "-f", "{{.ImportPath}}") {
		if pkg != cloudPkg && !strings.HasPrefix(pkg, cloudPkg+"/") {
			continue
		}

		s.True(strings.HasPrefix(pkg, managerAPIPkg), "product code depends on %s", pkg)
	}
}

func TestBoundarySuite(t *testing.T) {
	suite.Run(t, new(BoundarySuite))
}
