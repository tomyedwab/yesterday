package types

type AccessRequest struct {
	UserID        int
	ApplicationID string
}

type AccessResponse struct {
	AccessGranted bool
}
