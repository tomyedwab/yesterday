package types

type AccessTokenResponse struct {
	Expiry       int64
	RefreshToken string
	AccessToken  string
}
