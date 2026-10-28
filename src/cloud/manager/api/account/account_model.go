// Package account holds the accounts of the instance manager: who signed in,
// with which providers, and their sessions.
package account

import (
	"errors"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// ErrAmbiguousEmail is returned when the verified emails of a new identity
// belong to more than one account, so there is no single account to sign in to.
var ErrAmbiguousEmail = errors.New("the verified emails belong to more than one account")

// Account is a person using the instance manager.
type Account struct {
	ID          types.ID
	DisplayName string
	AvatarURL   string
	Emails      []string
	CreatedAt   time.Time
}

// JSON returns the account as sent to the manager UI.
func (a *Account) JSON() map[string]any {
	return map[string]any{
		"id":          a.ID.String(),
		"displayName": a.DisplayName,
		"avatarUrl":   a.AvatarURL,
		"emails":      a.Emails,
		"createdAt":   a.CreatedAt.Unix(),
	}
}
