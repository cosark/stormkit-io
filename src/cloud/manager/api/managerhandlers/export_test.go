package managerhandlers

import "github.com/stormkit-io/stormkit-io/src/cloud/manager/api/identity"

// SetProviders replaces the configured providers and returns a restore func.
func SetProviders(p map[string]identity.Provider) func() {
	original := providers
	providers = func() map[string]identity.Provider { return p }

	return func() { providers = original }
}
