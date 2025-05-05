package auth

import (
	"errors"

	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/users/sessions"
)

func DoLogout(db *database.Database, sessionManager *sessions.SessionManager, refreshToken string) error {
	session, err := sessionManager.GetSessionByRefreshToken(refreshToken)
	if err != nil {
		return errors.New("failed to get session")
	}

	err = session.DBDelete(db.GetDB())
	if err != nil {
		return errors.New("failed to delete session")
	}

	return nil
}
