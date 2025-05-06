package auth

import (
	"errors"
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

func DoRefresh(db *database.Database, sessionManager *sessions.SessionManager, refreshToken string, refreshRequest RefreshRequest) (*RefreshResponse, string, error) {
	if refreshRequest.ApplicationName == "" {
		fmt.Printf("DoRefresh failed: application name is required\n")
		return nil, "", errors.New("application name is required")
	}

	session, err := sessionManager.GetSessionByRefreshToken(refreshToken)
	if err != nil {
		fmt.Printf("DoRefresh failed to find session: %v\n", err)
		return nil, "", errors.New("failed to get session")
	}

	profile, err := state.GetUserProfile(db.GetDB(), session.UserID, refreshRequest.ApplicationName)
	if err != nil {
		fmt.Printf("DoRefresh failed to get user profile: %v\n", err)
		return nil, "", errors.New("failed to get user profile")
	}

	accessToken, refreshToken, err := sessionManager.RefreshAccessToken(session, refreshToken, refreshRequest.ApplicationName, profile)
	if err != nil {
		fmt.Printf("DoRefresh failed to refresh access token: %v\n", err)
		return nil, "", errors.New("failed to refresh access token")
	}

	return &RefreshResponse{
		AccessToken: accessToken,
	}, refreshToken, nil
}
