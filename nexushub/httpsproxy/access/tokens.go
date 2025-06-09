package access

import (
	"time"

	"github.com/tomyedwab/yesterday/apps/login/types"
)

type AccessToken struct {
	AccessToken   string
	SessionID     string
	ApplicationID string
	Expiry        int64
}

var AccessTokenStore = make(map[string]AccessToken)

func createAccessToken(response *types.AccessTokenResponse) {
	AccessTokenStore[response.AccessToken] = AccessToken{
		AccessToken:   response.AccessToken,
		ApplicationID: response.ApplicationID,
		Expiry:        response.Expiry,
	}
}

func ValidateAccessToken(token, applicationID string) bool {
	_, ok := AccessTokenStore[token]
	if !ok {
		return false
	}
	if time.Now().Unix() > AccessTokenStore[token].Expiry {
		delete(AccessTokenStore, token)
		return false
	}
	if AccessTokenStore[token].ApplicationID != applicationID {
		return false
	}
	return true
}
