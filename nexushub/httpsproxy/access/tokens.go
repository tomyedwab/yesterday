package access

import (
	"time"

	"github.com/tomyedwab/yesterday/nexushub/types"
)

type AccessToken struct {
	AccessToken string
	SessionID   string
	Expiry      int64
}

var AccessTokenStore = make(map[string]AccessToken)

func CreateAccessToken(response *types.AccessTokenResponse) {
	AccessTokenStore[response.AccessToken] = AccessToken{
		AccessToken: response.AccessToken,
		Expiry:      response.Expiry,
	}
}

func ValidateAccessToken(token string) bool {
	_, ok := AccessTokenStore[token]
	if !ok {
		return false
	}
	if time.Now().Unix() > AccessTokenStore[token].Expiry {
		delete(AccessTokenStore, token)
		return false
	}
	return true
}
