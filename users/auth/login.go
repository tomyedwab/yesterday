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
func DoLogin(db *database.Database, sessionManager *sessions.SessionManager, loginRequest LoginRequest) (string, error) {
	success, userId, err := state.AttemptLogin(db, loginRequest.Username, loginRequest.Password)
	if err != nil {
		return "", errors.New("invalid username or password")
	}
	if !success {
		return "", errors.New("invalid username or password")
	}

	session, err := sessionManager.CreateSession(userId)
	if err != nil {
		return "", errors.New("failed to create session")
	}

	return session.RefreshToken, nil
}
