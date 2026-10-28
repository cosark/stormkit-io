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

	state, err := secrets{}.random()

	if err != nil {
		return shttp.Error(err)
	}

	url := provider.AuthCodeURL(state)

	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: &url,
		Cookies:  []http.Cookie{stateCookie{}.set(stateCookieParams{provider: name, state: state})},
	}
}

// stateCookie binds an OAuth state to the browser that started the sign-in.
type stateCookie struct{}

type stateCookieParams struct {
	provider string
	state    string
}

func (stateCookie) value(p stateCookieParams) string {
	return p.provider + "." + p.state
}

func (sc stateCookie) set(p stateCookieParams) http.Cookie {
	return http.Cookie{
		Name:     stateCookieName,
		Value:    sc.value(p),
		Path:     "/cloud/auth",
		MaxAge:   int(stateCookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   !config.IsDevelopment(),
		SameSite: http.SameSiteLaxMode,
	}
}

func (sc stateCookie) clear() http.Cookie {
	cookie := sc.set(stateCookieParams{})
	cookie.Value = ""
	cookie.MaxAge = -1

	return cookie
}

type matchesStateParams struct {
	req      *shttp.RequestContext
	provider string
	state    string
}

// matches reports whether the request carries the cookie of this state.
func (sc stateCookie) matches(p matchesStateParams) bool {
	cookie, err := p.req.Cookie(stateCookieName)

	if err != nil || p.state == "" {
		return false
	}

	expected := sc.value(stateCookieParams{provider: p.provider, state: p.state})

	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(expected)) == 1
}
