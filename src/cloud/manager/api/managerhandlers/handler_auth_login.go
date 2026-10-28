package managerhandlers

import (
	"crypto/subtle"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAuthLogin sends the browser to the provider's consent page. The
// state it passes along is also kept in a cookie, so the callback only
// accepts the sign-in this browser started.
func handlerAuthLogin(req *shttp.RequestContext) *shttp.Response {
	name := req.Vars()["provider"]
	provider, ok := providers()[name]

	if !ok {
		return shttp.NotFound()
	}

	state := secrets{}.random()
	url := provider.AuthCodeURL(state)

	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: &url,
		Cookies:  []http.Cookie{stateCookie{}.set(name, state)},
	}
}

// stateCookie binds an OAuth state to the browser that started the sign-in.
type stateCookie struct{}

func (stateCookie) value(provider, state string) string {
	return provider + "." + state
}

func (sc stateCookie) set(provider, state string) http.Cookie {
	return http.Cookie{
		Name:     stateCookieName,
		Value:    sc.value(provider, state),
		Path:     "/cloud/auth",
		MaxAge:   int(stateCookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   !config.IsDevelopment(),
		SameSite: http.SameSiteLaxMode,
	}
}

func (sc stateCookie) clear() http.Cookie {
	cookie := sc.set("", "")
	cookie.Value = ""
	cookie.MaxAge = -1

	return cookie
}

// matches reports whether the request carries the cookie of this state.
func (sc stateCookie) matches(req *shttp.RequestContext, provider, state string) bool {
	cookie, err := req.Cookie(stateCookieName)

	if err != nil || state == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(sc.value(provider, state))) == 1
}
