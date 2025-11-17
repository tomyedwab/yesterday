package access

import (
	"fmt"
	"time"

	"github.com/tomyedwab/yesterday/nexushub/audit"
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

func ValidateAccessToken(token string, auditLogger *audit.Logger) bool {
	_, ok := AccessTokenStore[token]
	if !ok {
		return false
	}
	if time.Now().Unix() > AccessTokenStore[token].Expiry {
		// Log access token expiry
		if auditLogger != nil {
			if err := auditLogger.LogAccessTokenExpiry(token); err != nil {
				fmt.Printf("Failed to log access token expiry audit event: %v\n", err)
			}
		}
		delete(AccessTokenStore, token)
		return false
	}
	return true
}
