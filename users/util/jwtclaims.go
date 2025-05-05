package util

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const ClaimsKey contextKey = "claims"

type YesterdayUserClaims struct {
	jwt.Claims
	SessionID   string `json:"session_id"`
	Expiry      int64  `json:"exp"`
	IssuedAt    int64  `json:"iat"`
	Application string `json:"app"`
	Profile     string `json:"pro"`
}

func (c YesterdayUserClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.Expiry, 0)), nil
}

func (c YesterdayUserClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.IssuedAt, 0)), nil
}

func (c YesterdayUserClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.IssuedAt, 0)), nil
}

func (c YesterdayUserClaims) GetIssuer() (string, error) {
	return "", nil
}

func (c YesterdayUserClaims) GetSubject() (string, error) {
	return "", nil
}

func (c YesterdayUserClaims) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil
}
