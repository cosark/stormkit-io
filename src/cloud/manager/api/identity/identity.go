// Package identity signs people in to the instance manager with GitHub, GitLab
// or Google. Each provider is an OAuth app of the manager, separate from the
// apps the multi-tenant dashboard uses to reach repositories.
package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"golang.org/x/oauth2"
)

const (
	GitHub = "github"
	GitLab = "gitlab"
	Google = "google"
)

// Identity is someone a provider has authenticated.
type Identity struct {
	Provider       string
	ProviderUserID string
	DisplayName    string
	AvatarURL      string

	// Emails are the emails the provider says the person verified, lowercased.
	Emails []string
}

// Provider signs people in with one identity provider.
type Provider interface {
	// AuthCodeURL returns the provider's consent page.
	AuthCodeURL(state string) string

	// Identity exchanges an authorization code for the person it belongs to.
	Identity(ctx context.Context, code string) (*Identity, error)
}

// ProvidersParams configures the providers.
type ProvidersParams struct {
	// CallbackURL returns the redirect URL registered with a provider.
	CallbackURL func(provider string) string
}

// Providers returns the providers whose client ID and secret are set through
// STORMKIT_CLOUD_<PROVIDER>_CLIENT_ID and STORMKIT_CLOUD_<PROVIDER>_CLIENT_SECRET.
func Providers(p ProvidersParams) map[string]Provider {
	providers := map[string]Provider{}

	for _, name := range []string{GitHub, GitLab, Google} {
		prefix := "STORMKIT_CLOUD_" + strings.ToUpper(name)
		id := strings.TrimSpace(os.Getenv(prefix + "_CLIENT_ID"))
		secret := strings.TrimSpace(os.Getenv(prefix + "_CLIENT_SECRET"))

		if id == "" || secret == "" {
			continue
		}

		conf := &oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			RedirectURL:  p.CallbackURL(name),
		}

		switch name {
		case GitHub:
			providers[name] = NewGitHub(conf)
		case GitLab:
			providers[name] = NewGitLab(conf)
		case Google:
			providers[name] = NewGoogle(conf)
		}
	}

	return providers
}

// Names returns the names of the given providers, sorted.
func Names(providers map[string]Provider) []string {
	names := []string{}

	for name := range providers {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// apiClient reads a provider's JSON API with an access token.
type apiClient struct {
	client *http.Client
}

func (c apiClient) get(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")

	res, err := c.client.Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned %d", url, res.StatusCode)
	}

	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(out)
}

// emailSet collects verified emails, lowercased and without duplicates.
type emailSet struct {
	emails []string
	seen   map[string]bool
}

func (e *emailSet) add(email string) {
	email = strings.ToLower(strings.TrimSpace(email))

	// Relay addresses such as <id>@users.noreply.github.com cannot receive
	// mail and say nothing about who owns an account.
	if email == "" || strings.Contains(email, "noreply") || strings.Contains(email, "no-reply") {
		return
	}

	if e.seen == nil {
		e.seen = map[string]bool{}
	}

	if !e.seen[email] {
		e.seen[email] = true
		e.emails = append(e.emails, email)
	}
}

func (e *emailSet) list() []string {
	if e.emails == nil {
		return []string{}
	}

	return e.emails
}
