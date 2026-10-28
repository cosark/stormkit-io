package managerhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/account"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAuthSessionDelete signs out of the session the request was made with.
func handlerAuthSessionDelete(req *RequestContext) *shttp.Response {
	if err := account.NewStore().DeleteSession(req.Context(), req.TokenHash); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
