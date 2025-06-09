package types

type AccessTokenRequest struct {
	RefreshToken  string
	ApplicationID string
}

type AccessTokenResponse struct {
	Expiry        int64
	RefreshToken  string
	AccessToken   string
	ApplicationID string
}
