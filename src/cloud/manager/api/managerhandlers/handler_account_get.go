package managerhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAccountGet returns the signed-in account.
func handlerAccountGet(req *RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"account": req.Account.JSON(),
		},
	}
}
