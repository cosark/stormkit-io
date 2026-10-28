package managerhandlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/identity"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	// ConsoleURLEnv overrides where the manager UI is served.
	ConsoleURLEnv = "STORMKIT_CLOUD_CONSOLE_URL"

	// SessionTTL is how long a manager session lasts.
	SessionTTL = 30 * 24 * time.Hour

	// LoginCodeTTL is how long the UI has to trade a login code for a session.
	LoginCodeTTL = time.Minute

	sessionTokenPrefix = "skc_"
	stateCookieName    = "sk_cloud_oauth_state"
	stateCookieTTL     = 10 * time.Minute
	redisTimeout       = 3 * time.Second
)

// providers returns the configured identity providers. Tests replace it.
var providers = func() map[string]identity.Provider {
	return identity.Providers(identity.ProvidersParams{
		CallbackURL: func(provider string) string {
			return admin.MustConfig().ApiURL("/cloud/auth/" + provider + "/callback")
		},
	})
}

// consoleURL returns the origin of the manager UI.
func consoleURL() string {
	if url := strings.TrimRight(strings.TrimSpace(os.Getenv(ConsoleURLEnv)), "/"); url != "" {
		return url
	}

	if config.IsDevelopment() {
		return "http://localhost:5500"
	}

	return "https://console.stormkit.io"
}

// secrets creates and hashes the random values of the sign-in flow.
type secrets struct{}

// random returns 32 random bytes, base64url-encoded.
func (secrets) random() string {
	b := make([]byte, 32)

	// crypto/rand.Read never returns an error since Go 1.24.
	_, _ = rand.Read(b)

	return base64.RawURLEncoding.EncodeToString(b)
}

// hash returns the hex SHA-256 of a token, which is what gets stored.
func (secrets) hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// errNoRedis is returned when login codes cannot be stored.
var errNoRedis = errors.New("the instance manager needs redis for login codes")

// loginCodes hands the UI a short-lived, single-use code after a provider
// signs someone in. The UI trades it for a session, so the session token
// never travels in a URL.
type loginCodes struct{}

func (loginCodes) key(code string) string {
	return "cloud:login-code:" + secrets{}.hash(code)
}

// issue stores a new code for the account and returns it.
func (lc loginCodes) issue(ctx context.Context, accountID types.ID) (string, error) {
	client := rediscache.Client()

	if client == nil {
		return "", errNoRedis
	}

	ctx, cancel := context.WithTimeout(ctx, redisTimeout)
	defer cancel()

	code := secrets{}.random()

	if err := client.Set(ctx, lc.key(code), accountID.String(), LoginCodeTTL).Err(); err != nil {
		return "", err
	}

	return code, nil
}

// redeem spends a code and returns its account, or 0 when the code is
// unknown, expired or already spent.
func (lc loginCodes) redeem(ctx context.Context, code string) (types.ID, error) {
	client := rediscache.Client()

	if client == nil {
		return 0, errNoRedis
	}

	ctx, cancel := context.WithTimeout(ctx, redisTimeout)
	defer cancel()

	value, err := client.GetDel(ctx, lc.key(code)).Result()

	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	return utils.StringToID(value), nil
}
