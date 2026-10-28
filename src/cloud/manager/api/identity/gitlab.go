package identity

import (
	"context"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/gitlab"
)

// GitLabProvider signs people in with a GitLab.com OAuth app.
type GitLabProvider struct {
	Config *oauth2.Config
	APIURL string
}

// NewGitLab returns a GitLab provider that reads the profile and emails.
func NewGitLab(conf *oauth2.Config) *GitLabProvider {
	conf.Endpoint = gitlab.Endpoint
	conf.Scopes = []string{"read_user"}

	return &GitLabProvider{Config: conf, APIURL: "https://gitlab.com/api/v4"}
}

func (g *GitLabProvider) AuthCodeURL(state string) string {
	return g.Config.AuthCodeURL(state)
}

func (g *GitLabProvider) Identity(ctx context.Context, code string) (*Identity, error) {
	token, err := g.Config.Exchange(ctx, code)

	if err != nil {
		return nil, err
	}

	api := apiClient{client: g.Config.Client(ctx, token)}

	profile := struct {
		ID          int64   `json:"id"`
		Username    string  `json:"username"`
		Name        string  `json:"name"`
		AvatarURL   string  `json:"avatar_url"`
		Email       string  `json:"email"`
		ConfirmedAt *string `json:"confirmed_at"`
	}{}

	if err := api.get(ctx, g.APIURL+"/user", &profile); err != nil {
		return nil, err
	}

	// The secondary emails; the primary one is on the profile.
	records := []struct {
		Email       string  `json:"email"`
		ConfirmedAt *string `json:"confirmed_at"`
	}{}

	if err := api.get(ctx, g.APIURL+"/user/emails", &records); err != nil {
		return nil, err
	}

	emails := emailSet{}

	if profile.ConfirmedAt != nil && *profile.ConfirmedAt != "" {
		emails.add(profile.Email)
	}

	for _, record := range records {
		if record.ConfirmedAt != nil && *record.ConfirmedAt != "" {
			emails.add(record.Email)
		}
	}

	name := profile.Name

	if name == "" {
		name = profile.Username
	}

	return &Identity{
		Provider:       GitLab,
		ProviderUserID: strconv.FormatInt(profile.ID, 10),
		DisplayName:    name,
		AvatarURL:      profile.AvatarURL,
		Emails:         emails.list(),
	}, nil
}
