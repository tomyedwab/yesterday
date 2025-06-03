package types

type AdminAppInfoRequest struct {
	ApplicationID string
}

type AdminAppInfoResponse struct {
	ApplicationHostName string
}

type AdminLoginRequest struct {
	Username      string
	Password      string
	ApplicationID string
}

type AdminLoginResponse struct {
	Success             bool
	UserID              int
	ApplicationID       string
	ApplicationHostName string
}
