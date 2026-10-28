package identity

import (
	"context"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider signs people in with a Google OAuth client.
type GoogleProvider struct {
	Config      *oauth2.Config
	UserInfoURL string
}

// NewGoogle returns a Google provider that reads the OpenID Connect profile.
func NewGoogle(conf *oauth2.Config) *GoogleProvider {
	conf.Endpoint = google.Endpoint
	conf.Scopes = []string{"openid", "email", "profile"}

	return &GoogleProvider{Config: conf, UserInfoURL: "https://openidconnect.googleapis.com/v1/userinfo"}
}

func (g *GoogleProvider) AuthCodeURL(state string) string {
	return g.Config.AuthCodeURL(state)
}

func (g *GoogleProvider) Identity(ctx context.Context, code string) (*Identity, error) {
	token, err := g.Config.Exchange(ctx, code)

	if err != nil {
		return nil, err
	}

	api := apiClient{client: g.Config.Client(ctx, token)}

	profile := struct {
		Sub           string `json:"sub"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}{}

	if err := api.get(ctx, g.UserInfoURL, &profile); err != nil {
		return nil, err
	}

	emails := emailSet{}

	if profile.EmailVerified {
		emails.add(profile.Email)
	}

	return &Identity{
		Provider:       Google,
		ProviderUserID: profile.Sub,
		DisplayName:    profile.Name,
		AvatarURL:      profile.Picture,
		Emails:         emails.list(),
	}, nil
}
