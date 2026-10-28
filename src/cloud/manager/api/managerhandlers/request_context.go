package managerhandlers

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/account"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// RequestContext is a request from a signed-in account.
type RequestContext struct {
	*shttp.RequestContext

	// Account is the signed-in account.
	Account *account.Account

	// TokenHash identifies the session the request was made with.
	TokenHash string
}

// WithAccount lets the handler run only for requests carrying the bearer
// token of an unexpired session.
func WithAccount(handler func(*RequestContext) *shttp.Response) shttp.RequestFunc {
	return func(req *shttp.RequestContext) *shttp.Response {
		token, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")

		if !ok || strings.TrimSpace(token) == "" {
			return shttp.NotAllowed()
		}

		hash := secrets{}.hash(strings.TrimSpace(token))
		acc, err := account.NewStore().SessionAccount(req.Context(), hash)

		if err != nil {
			return shttp.Error(err)
		}

		if acc == nil {
			return shttp.NotAllowed()
		}

		return handler(&RequestContext{RequestContext: req, Account: acc, TokenHash: hash})
	}
}
