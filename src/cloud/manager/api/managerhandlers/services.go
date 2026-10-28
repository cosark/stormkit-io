// Package managerhandlers serves the API of the instance manager, the Stormkit
// Cloud console where customers manage their dedicated instances. It is
// registered only on Stormkit Cloud, under /cloud.
package managerhandlers

import (
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/limiter"
)

// Services installs the instance manager's endpoints.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/cloud/auth").
		Handler(shttp.MethodGet, "/providers", handlerAuthProviders).
		Handler(shttp.MethodGet, "/{provider:[a-z]+}", shttp.WithRateLimit(
			handlerAuthLogin,
			&limiter.Options{Limit: 30, Burst: 10, Duration: time.Minute},
		)).
		Handler(shttp.MethodGet, "/{provider:[a-z]+}/callback", shttp.WithRateLimit(
			handlerAuthCallback,
			&limiter.Options{Limit: 30, Burst: 10, Duration: time.Minute},
		)).
		Handler(shttp.MethodPost, "/session", shttp.WithRateLimit(
			handlerAuthSessionCreate,
			&limiter.Options{Limit: 20, Burst: 5, Duration: time.Minute},
		)).
		Handler(shttp.MethodDelete, "/session", WithAccount(handlerAuthSessionDelete))

	s.NewEndpoint("/cloud/account").
		Handler(shttp.MethodGet, "", WithAccount(handlerAccountGet))

	return s
}
