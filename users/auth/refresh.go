package auth

import (
	"errors"
	"fmt"

	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/users/sessions"
	"github.com/tomyedwab/yesterday/users/state"
)

type RefreshRequest struct {
	RefreshToken    string `json:"refresh_token"`
	ApplicationName string `json:"application"`
}

type RefreshResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

func DoRefresh(db *database.Database, sessionManager *sessions.SessionManager, refreshRequest RefreshRequest) (*RefreshResponse, error) {
	if refreshRequest.ApplicationName == "" {
		fmt.Printf("DoRefresh failed: application name is required\n")
		return nil, errors.New("application name is required")
	}

	session, err := sessionManager.GetSessionByRefreshToken(refreshRequest.RefreshToken)
	if err != nil {
		fmt.Printf("DoRefresh failed to find session: %v\n", err)
		return nil, errors.New("failed to get session")
	}

	profile, err := state.GetUserProfile(db.GetDB(), session.UserID, refreshRequest.ApplicationName)
	if err != nil {
		fmt.Printf("DoRefresh failed to get user profile: %v\n", err)
		return nil, errors.New("failed to get user profile")
	}

	accessToken, refreshToken, err := sessionManager.RefreshAccessToken(session, refreshRequest.RefreshToken, refreshRequest.ApplicationName, profile)
	if err != nil {
		fmt.Printf("DoRefresh failed to refresh access token: %v\n", err)
		return nil, errors.New("failed to refresh access token")
	}

	return &RefreshResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}
