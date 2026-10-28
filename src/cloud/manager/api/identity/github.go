package identity

import (
	"context"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHubProvider signs people in with a GitHub OAuth app.
type GitHubProvider struct {
	Config *oauth2.Config
	APIURL string
}

// NewGitHub returns a GitHub provider that reads the profile and emails.
func NewGitHub(conf *oauth2.Config) *GitHubProvider {
	conf.Endpoint = github.Endpoint
	conf.Scopes = []string{"read:user", "user:email"}

	return &GitHubProvider{Config: conf, APIURL: "https://api.github.com"}
}

func (g *GitHubProvider) AuthCodeURL(state string) string {
	return g.Config.AuthCodeURL(state)
}

func (g *GitHubProvider) Identity(ctx context.Context, code string) (*Identity, error) {
	token, err := g.Config.Exchange(ctx, code)

	if err != nil {
		return nil, err
	}

	api := apiClient{client: g.Config.Client(ctx, token)}

	profile := struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}{}

	if err := api.get(getParams{ctx: ctx, url: g.APIURL + "/user", out: &profile}); err != nil {
		return nil, err
	}

	records := []struct {
		Email    string `json:"email"`
		Verified bool   `json:"verified"`
	}{}

	if err := api.get(getParams{ctx: ctx, url: g.APIURL + "/user/emails", out: &records}); err != nil {
		return nil, err
	}

	emails := emailSet{}

	for _, record := range records {
		if record.Verified {
			emails.add(record.Email)
		}
	}

	name := profile.Name

	if name == "" {
		name = profile.Login
	}

	return &Identity{
		Provider:       GitHub,
		ProviderUserID: strconv.FormatInt(profile.ID, 10),
		DisplayName:    name,
		AvatarURL:      profile.AvatarURL,
		Emails:         emails.list(),
	}, nil
}
