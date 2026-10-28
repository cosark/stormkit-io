package managerhandlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/cloud/manager/api/account"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// SessionCreateRequest is the body of a session request.
type SessionCreateRequest struct {
	Code string `json:"code"`
}

// handlerAuthSessionCreate trades a login code for a session. The bearer
// token is returned once; only its hash is stored.
func handlerAuthSessionCreate(req *shttp.RequestContext) *shttp.Response {
	data := SessionCreateRequest{}

	if err := req.Post(&data); err != nil || strings.TrimSpace(data.Code) == "" {
		return shttp.BadRequest(map[string]any{"error": "A login code is required"})
	}

	accountID, err := loginCodes{}.redeem(redeemLoginCodeParams{ctx: req.Context(), code: strings.TrimSpace(data.Code)})

	if err != nil {
		return shttp.Error(err)
	}

	if accountID == 0 {
		return shttp.NotAllowed()
	}

	store := account.NewStore()
	acc, err := store.AccountByID(req.Context(), accountID)

	if err != nil {
		return shttp.Error(err)
	}

	if acc == nil {
		return shttp.NotAllowed()
	}

	random, err := secrets{}.random()

	if err != nil {
		return shttp.Error(err)
	}

	token := sessionTokenPrefix + random
	expiresAt := time.Now().Add(SessionTTL)

	err = store.CreateSession(req.Context(), account.CreateSessionParams{
		AccountID: acc.ID,
		TokenHash: secrets{}.hash(token),
		ExpiresAt: expiresAt,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"token":     token,
			"expiresAt": expiresAt.Unix(),
			"account":   acc.JSON(),
		},
	}
}
