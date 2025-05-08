package auth

import (
	"fmt"

	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/users/sessions"
	"github.com/tomyedwab/yesterday/users/state"
)

type RefreshRequest struct {
	ApplicationName string `json:"application"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

func DoRefresh(
	db *database.Database,
	sessionManager *sessions.SessionManager,
	refreshToken string,
) (*RefreshResponse, string, error) {
	session, err := sessionManager.GetSessionByRefreshToken(refreshToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get session: %v", err)
	}

	profile, err := state.GetUserProfile(db.GetDB(), session.UserID, session.ApplicationID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user profile: %v", err)
	}
	if profile == nil {
		fmt.Printf("DoRefresh user profile does not exist for application %s\n", session.ApplicationID)
		return nil, "", fmt.Errorf("user does not have access to this application")
	}

	accessToken, refreshToken, err := sessionManager.RefreshAccessToken(session, refreshToken, session.ApplicationID, *profile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to refresh access token: %v", err)
	}

	return &RefreshResponse{
		AccessToken: accessToken,
	}, refreshToken, nil
}
