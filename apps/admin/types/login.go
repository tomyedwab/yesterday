package types

type AdminLoginRequest struct {
	Username string
	Password string
}

type AdminLoginResponse struct {
	Success bool
	UserID  int
}
