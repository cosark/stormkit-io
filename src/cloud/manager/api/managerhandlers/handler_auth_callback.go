package managerhandlers

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/account"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

// Reasons a sign-in failed, passed to the manager UI as ?error=.
const (
	callbackErrorDenied    = "denied"
	callbackErrorState     = "state"
	callbackErrorProvider  = "provider"
	callbackErrorEmail     = "email"
	callbackErrorAmbiguous = "ambiguous"
	callbackErrorServer    = "server"
)

// handlerAuthCallback completes a sign-in. It signs the person in to their
// account, creating it on the first sign-in, and sends the browser back to
// the manager UI with a one-time code the UI trades for a session.
func handlerAuthCallback(req *shttp.RequestContext) *shttp.Response {
	name := req.Vars()["provider"]
	provider, ok := providers()[name]

	if !ok {
		return shttp.NotFound()
	}

	query := req.Query()

	if !(stateCookie{}).matches(req, name, query.Get("state")) {
		return callbackRedirect(url.Values{"error": {callbackErrorState}})
	}

	if query.Get("error") != "" || query.Get("code") == "" {
		return callbackRedirect(url.Values{"error": {callbackErrorDenied}})
	}

	ident, err := provider.Identity(req.Context(), query.Get("code"))

	if err != nil || ident == nil || ident.ProviderUserID == "" || ident.ProviderUserID == "0" {
		slog.Errorf("cloud sign-in with %s failed: %v", name, err)
		return callbackRedirect(url.Values{"error": {callbackErrorProvider}})
	}

	if len(ident.Emails) == 0 {
		return callbackRedirect(url.Values{"error": {callbackErrorEmail}})
	}

	acc, err := account.NewStore().SignIn(req.Context(), account.SignInParams{
		Provider:       ident.Provider,
		ProviderUserID: ident.ProviderUserID,
		DisplayName:    ident.DisplayName,
		AvatarURL:      ident.AvatarURL,
		Emails:         ident.Emails,
	})

	if errors.Is(err, account.ErrAmbiguousEmail) {
		slog.Infof("cloud sign-in with %s user %s rejected: %s", name, ident.ProviderUserID, err.Error())
		return callbackRedirect(url.Values{"error": {callbackErrorAmbiguous}})
	}

	if err != nil {
		slog.Errorf("cloud sign-in with %s: cannot sign in user %s: %s", name, ident.ProviderUserID, err.Error())
		return callbackRedirect(url.Values{"error": {callbackErrorServer}})
	}

	code, err := loginCodes{}.issue(req.Context(), acc.ID)

	if err != nil {
		slog.Errorf("cloud sign-in: cannot issue a login code: %s", err.Error())
		return callbackRedirect(url.Values{"error": {callbackErrorServer}})
	}

	return callbackRedirect(url.Values{"code": {code}})
}

// callbackRedirect sends the browser to the manager UI's callback page and
// clears the state cookie, which is single use.
func callbackRedirect(query url.Values) *shttp.Response {
	target := consoleURL() + "/auth/callback?" + query.Encode()

	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: &target,
		Cookies:  []http.Cookie{stateCookie{}.clear()},
	}
}
