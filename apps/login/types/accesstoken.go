package types

type AccessTokenRequest struct {
	RefreshToken string
}

type AccessTokenResponse struct {
	Expiry       int64
	RefreshToken string
	AccessToken  string
}
