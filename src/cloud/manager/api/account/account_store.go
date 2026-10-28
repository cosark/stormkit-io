package account

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

var stmt = struct {
	identityAccount  string
	emailAccounts    string
	insertAccount    string
	insertIdentity   string
	insertEmails     string
	updateLastLogin  string
	selectAccount    string
	insertSession    string
	sessionAccountID string
	deleteSession    string
}{
	identityAccount: `
		SELECT account_id FROM cloud.account_identities
		WHERE provider = $1 AND provider_user_id = $2;
	`,

	emailAccounts: `
		SELECT DISTINCT account_id FROM cloud.account_emails
		WHERE email = ANY($1);
	`,

	insertAccount: `
		INSERT INTO cloud.accounts (display_name, avatar_url)
		VALUES ($1, $2)
		RETURNING account_id;
	`,

	insertIdentity: `
		INSERT INTO cloud.account_identities (account_id, provider, provider_user_id)
		VALUES ($1, $2, $3);
	`,

	// Emails another account already owns stay with that account.
	insertEmails: `
		INSERT INTO cloud.account_emails (account_id, email)
		SELECT $1, unnest($2::text[])
		ON CONFLICT (email) DO NOTHING;
	`,

	updateLastLogin: `
		UPDATE cloud.accounts SET
			last_login_at = (now() AT TIME ZONE 'UTC'),
			display_name = CASE WHEN display_name = '' THEN $2 ELSE display_name END,
			avatar_url = CASE WHEN $3 <> '' THEN $3 ELSE avatar_url END
		WHERE account_id = $1;
	`,

	selectAccount: `
		SELECT
			a.account_id,
			a.display_name,
			a.avatar_url,
			a.created_at,
			COALESCE(
				(SELECT array_agg(e.email ORDER BY e.email_id) FROM cloud.account_emails e WHERE e.account_id = a.account_id),
				'{}'
			)
		FROM cloud.accounts a
		WHERE a.account_id = $1;
	`,

	insertSession: `
		INSERT INTO cloud.sessions (account_id, token_hash, expires_at)
		VALUES ($1, $2, $3);
	`,

	sessionAccountID: `
		SELECT account_id FROM cloud.sessions
		WHERE token_hash = $1 AND expires_at > (now() AT TIME ZONE 'UTC');
	`,

	deleteSession: `
		DELETE FROM cloud.sessions WHERE token_hash = $1;
	`,
}

// Store reads and writes accounts and sessions.
type Store struct {
	*database.Store
}

// NewStore returns a new account store.
func NewStore() *Store {
	return &Store{Store: database.NewStore()}
}

// SignInParams describes someone a provider has authenticated.
type SignInParams struct {
	Provider       string
	ProviderUserID string
	DisplayName    string
	AvatarURL      string

	// Emails are the verified emails the provider vouches for, lowercased.
	Emails []string
}

// SignIn returns the account of an authenticated identity, creating it on the
// first sign-in. A new identity joins the account that owns one of its
// verified emails, so signing in with GitHub and later with Google lands in
// the same account. It returns ErrAmbiguousEmail when those emails belong to
// more than one account.
func (s *Store) SignIn(ctx context.Context, p SignInParams) (*Account, error) {
	tx, err := s.Conn.BeginTx(ctx, nil)

	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	var accountID types.ID

	err = tx.QueryRowContext(ctx, stmt.identityAccount, p.Provider, p.ProviderUserID).Scan(&accountID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if accountID == 0 {
		if accountID, err = s.linkIdentity(ctx, linkIdentityParams{tx: tx, signIn: p}); err != nil {
			return nil, err
		}
	}

	if _, err := tx.ExecContext(ctx, stmt.insertEmails, accountID, pq.Array(p.Emails)); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, stmt.updateLastLogin, accountID, p.DisplayName, p.AvatarURL); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.AccountByID(ctx, accountID)
}

type linkIdentityParams struct {
	tx     *sql.Tx
	signIn SignInParams
}

// linkIdentity attaches a new identity to the account owning its emails, or
// to a new account when no account owns them.
func (s *Store) linkIdentity(ctx context.Context, p linkIdentityParams) (types.ID, error) {
	rows, err := p.tx.QueryContext(ctx, stmt.emailAccounts, pq.Array(p.signIn.Emails))

	if err != nil {
		return 0, err
	}

	ids := []types.ID{}

	for rows.Next() {
		var id types.ID

		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}

		ids = append(ids, id)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(ids) > 1 {
		return 0, ErrAmbiguousEmail
	}

	var accountID types.ID

	if len(ids) == 1 {
		accountID = ids[0]
	} else {
		err := p.tx.QueryRowContext(ctx, stmt.insertAccount, p.signIn.DisplayName, p.signIn.AvatarURL).Scan(&accountID)

		if err != nil {
			return 0, err
		}
	}

	if _, err := p.tx.ExecContext(ctx, stmt.insertIdentity, accountID, p.signIn.Provider, p.signIn.ProviderUserID); err != nil {
		return 0, err
	}

	return accountID, nil
}

// AccountByID returns the account with the given ID, or nil when there is none.
func (s *Store) AccountByID(ctx context.Context, id types.ID) (*Account, error) {
	row, err := s.QueryRow(ctx, stmt.selectAccount, id)

	if err != nil {
		return nil, err
	}

	a := &Account{}
	emails := pq.StringArray{}

	if err := row.Scan(&a.ID, &a.DisplayName, &a.AvatarURL, &a.CreatedAt, &emails); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	a.Emails = emails

	return a, nil
}

// CreateSessionParams describes a new session.
type CreateSessionParams struct {
	AccountID types.ID
	TokenHash string
	ExpiresAt time.Time
}

// CreateSession stores a session under the hash of its bearer token.
func (s *Store) CreateSession(ctx context.Context, p CreateSessionParams) error {
	_, err := s.Exec(ctx, stmt.insertSession, p.AccountID, p.TokenHash, p.ExpiresAt.UTC())
	return err
}

// SessionAccount returns the account of an unexpired session, or nil when no
// such session exists.
func (s *Store) SessionAccount(ctx context.Context, tokenHash string) (*Account, error) {
	row, err := s.QueryRow(ctx, stmt.sessionAccountID, tokenHash)

	if err != nil {
		return nil, err
	}

	var accountID types.ID

	if err := row.Scan(&accountID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return s.AccountByID(ctx, accountID)
}

// DeleteSession ends a session. Ending one that does not exist is not an error.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.Exec(ctx, stmt.deleteSession, tokenHash)
	return err
}
