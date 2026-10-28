package managerhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/identity"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAuthProviders lists the providers the manager UI can offer.
func handlerAuthProviders(req *shttp.RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"providers": identity.Names(providers()),
		},
	}
}
