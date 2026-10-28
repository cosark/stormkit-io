package account_test

import (
	"context"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/account"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stretchr/testify/suite"
)

type AccountStoreSuite struct {
	suite.Suite

	conn  databasetest.TestDB
	store *account.Store
	ctx   context.Context
}

func (s *AccountStoreSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.store = account.NewStore()
	s.ctx = context.Background()
}

func (s *AccountStoreSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *AccountStoreSuite) signIn(p account.SignInParams) *account.Account {
	acc, err := s.store.SignIn(s.ctx, p)
	s.Require().NoError(err)
	s.Require().NotNil(acc)

	return acc
}

func (s *AccountStoreSuite) Test_SignIn_CreatesAccount() {
	acc := s.signIn(account.SignInParams{
		Provider:       "github",
		ProviderUserID: "1",
		DisplayName:    "Jane",
		AvatarURL:      "https://avatars.test/jane",
		Emails:         []string{"jane@acme.test", "jane@home.test"},
	})

	s.NotZero(acc.ID)
	s.Equal("Jane", acc.DisplayName)
	s.Equal("https://avatars.test/jane", acc.AvatarURL)
	s.Equal([]string{"jane@acme.test", "jane@home.test"}, acc.Emails)
}

func (s *AccountStoreSuite) Test_SignIn_SameIdentityReturnsSameAccount() {
	first := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", Emails: []string{"jane@acme.test"}})
	second := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", Emails: []string{"jane@acme.test"}})

	s.Equal(first.ID, second.ID)
}

// Test_SignIn_OtherProviderJoinsByEmail verifies that someone signing in with
// Google lands in the account they created with GitHub.
func (s *AccountStoreSuite) Test_SignIn_OtherProviderJoinsByEmail() {
	github := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", Emails: []string{"jane@acme.test"}})
	google := s.signIn(account.SignInParams{Provider: "google", ProviderUserID: "abc", Emails: []string{"jane@acme.test", "jane@gmail.test"}})

	s.Equal(github.ID, google.ID)
	s.Equal([]string{"jane@acme.test", "jane@gmail.test"}, google.Emails)
}

// Test_SignIn_IdentityKeepsItsAccount verifies that a linked identity stays
// with its account even after its email changes at the provider.
func (s *AccountStoreSuite) Test_SignIn_IdentityKeepsItsAccount() {
	jane := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", Emails: []string{"jane@acme.test"}})
	again := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", Emails: []string{"jane@new.test"}})

	s.Equal(jane.ID, again.ID)
	s.Equal([]string{"jane@acme.test", "jane@new.test"}, again.Emails)
}

func (s *AccountStoreSuite) Test_SignIn_NewIdentityNewEmailsNewAccount() {
	jane := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", Emails: []string{"jane@acme.test"}})
	john := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "2", Emails: []string{"john@acme.test"}})

	s.NotEqual(jane.ID, john.ID)
}

// Test_SignIn_EmailsAreNotTakenFromAnotherAccount verifies that a known
// identity reporting an email another account owns does not move that email.
func (s *AccountStoreSuite) Test_SignIn_EmailsAreNotTakenFromAnotherAccount() {
	jane := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", Emails: []string{"jane@acme.test"}})
	john := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "2", Emails: []string{"john@acme.test"}})
	john = s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "2", Emails: []string{"john@acme.test", "jane@acme.test"}})

	s.Equal([]string{"john@acme.test"}, john.Emails)

	jane, err := s.store.AccountByID(s.ctx, jane.ID)
	s.Require().NoError(err)
	s.Equal([]string{"jane@acme.test"}, jane.Emails)
}

func (s *AccountStoreSuite) Test_SignIn_AmbiguousEmails() {
	s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", Emails: []string{"jane@acme.test"}})
	s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "2", Emails: []string{"jane@home.test"}})

	acc, err := s.store.SignIn(s.ctx, account.SignInParams{
		Provider:       "google",
		ProviderUserID: "abc",
		Emails:         []string{"jane@acme.test", "jane@home.test"},
	})

	s.ErrorIs(err, account.ErrAmbiguousEmail)
	s.Nil(acc)

	var identities int
	s.Require().NoError(s.conn.QueryRow("SELECT count(*) FROM cloud.account_identities WHERE provider = 'google'").Scan(&identities))
	s.Zero(identities)
}

// Test_SignIn_KeepsTheFirstDisplayName verifies that the name people see does
// not flip between providers that spell it differently.
func (s *AccountStoreSuite) Test_SignIn_KeepsTheFirstDisplayName() {
	s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", DisplayName: "Jane", Emails: []string{"jane@acme.test"}})
	acc := s.signIn(account.SignInParams{Provider: "google", ProviderUserID: "abc", DisplayName: "Jane Doe", Emails: []string{"jane@acme.test"}})

	s.Equal("Jane", acc.DisplayName)
}

func (s *AccountStoreSuite) Test_Sessions() {
	acc := s.signIn(account.SignInParams{Provider: "github", ProviderUserID: "1", Emails: []string{"jane@acme.test"}})

	s.Require().NoError(s.store.CreateSession(s.ctx, account.CreateSessionParams{
		AccountID: acc.ID,
		TokenHash: "live",
		ExpiresAt: time.Now().Add(time.Hour),
	}))

	s.Require().NoError(s.store.CreateSession(s.ctx, account.CreateSessionParams{
		AccountID: acc.ID,
		TokenHash: "expired",
		ExpiresAt: time.Now().Add(-time.Minute),
	}))

	live, err := s.store.SessionAccount(s.ctx, "live")
	s.Require().NoError(err)
	s.Require().NotNil(live)
	s.Equal(acc.ID, live.ID)

	expired, err := s.store.SessionAccount(s.ctx, "expired")
	s.NoError(err)
	s.Nil(expired)

	s.Require().NoError(s.store.DeleteSession(s.ctx, "live"))

	deleted, err := s.store.SessionAccount(s.ctx, "live")
	s.NoError(err)
	s.Nil(deleted)
}

func (s *AccountStoreSuite) Test_AccountByID_Unknown() {
	acc, err := s.store.AccountByID(s.ctx, 987654)

	s.NoError(err)
	s.Nil(acc)
}

func TestAccountStoreSuite(t *testing.T) {
	suite.Run(t, new(AccountStoreSuite))
}
