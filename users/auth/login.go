package auth

import (
	"errors"

	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/users/sessions"
	"github.com/tomyedwab/yesterday/users/state"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

// Login with given username and password. Returns a refresh token.
func DoLogin(
	db *database.Database,
	sessionManager *sessions.SessionManager,
	loginRequest LoginRequest,
	applicationId string,
) (string, error) {
	success, userId, err := state.AttemptLogin(db, loginRequest.Username, loginRequest.Password)
	if err != nil {
		return "", errors.New("invalid username or password")
	}
	if !success {
		return "", errors.New("invalid username or password")
	}

	profile, err := state.GetUserProfile(db.GetDB(), userId, applicationId)
	if err != nil {
		return "", errors.New("failed to get user profile")
	}
	if profile == nil {
		return "", errors.New("user does not have access to this application")
	}

	session, err := sessionManager.CreateSession(userId, applicationId)
	if err != nil {
		return "", errors.New("failed to create session")
	}

	return session.RefreshToken, nil
}
