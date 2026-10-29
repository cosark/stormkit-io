package skauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"golang.org/x/oauth2"
)

// GitLabBaseURL is the GitLab instance users sign in with. Tests point it at
// a fake server.
var GitLabBaseURL = "https://gitlab.com"

// GitLabClient signs users in with GitLab through OpenID Connect. One GitLab
// application accepts several redirect URIs, one per line.
//
// Step 1: https://gitlab.com/-/user_settings/applications (or the group's Settings > Applications)
// Step 2: Redirect URI: https://sample.stormkit:8888/_stormkit/auth/callback
// Step 3: Keep "Confidential" checked and select the openid, profile and email scopes
// Step 4: Obtain the application ID and secret
type GitLabClient struct {
	oauth2Config *oauth2.Config
}

// GitLabClientParams configures a GitLab client.
type GitLabClientParams struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// NewGitLabClient returns a client that reads the user's OpenID Connect claims.
func NewGitLabClient(p GitLabClientParams) Client {
	config := &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		RedirectURL:  p.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  GitLabBaseURL + "/oauth/authorize",
			TokenURL: GitLabBaseURL + "/oauth/token",
		},
		Scopes: []string{"openid", "profile", "email"},
	}

	return &GitLabClient{
		oauth2Config: config,
	}
}

// GitLabUserInfo is the part of GitLab's OpenID Connect userinfo response
// that is used. Email is the user's primary email.
type GitLabUserInfo struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	Nickname      string `json:"nickname"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Picture       string `json:"picture"`
	Profile       string `json:"profile"`
}

// UserInfo returns the signed-in GitLab user.
func (g *GitLabClient) UserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := oauth2.NewClient(ctx, g.oauth2Config.TokenSource(ctx, token))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GitLabBaseURL+"/oauth/userinfo", nil)

	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// GitLab usually refuses missing scopes on its consent page already; a
	// token that lacks them is refused here.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab: GET /oauth/userinfo returned %d (the application needs the openid, profile and email scopes)", resp.StatusCode)
	}

	var userInfo GitLabUserInfo

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	name := userInfo.Name

	if name == "" {
		name = userInfo.Nickname
	}

	return &UserInfo{
		AccountID:     userInfo.Sub,
		Email:         userInfo.Email,
		EmailVerified: userInfo.EmailVerified,
		Avatar:        userInfo.Picture,
		FirstName:     name,
		UserMetadata: UserMetadata{
			Username:   userInfo.Nickname,
			ProfileURL: userInfo.Profile,
		},
	}, nil
}

// Exchange trades the authorization code for an access token.
func (g *GitLabClient) Exchange(ctx context.Context, req *shttp.RequestContext) (*oauth2.Token, error) {
	return g.oauth2Config.Exchange(ctx, req.FormValue("code"))
}

// AuthCodeURL returns GitLab's consent page with a state signed by the
// environment's auth secret.
func (g *GitLabClient) AuthCodeURL(params AuthCodeURLParams) (string, error) {
	state, err := user.JWT(user.JWTParams{Purpose: user.PurposeSkAuthOAuthState, Claims: params.Claims(), Secret: params.Secret})

	if err != nil {
		return "", err
	}

	return g.oauth2Config.AuthCodeURL(state), nil
}
