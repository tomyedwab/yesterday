package auth

import (
	"fmt"

	"github.com/tomyedwab/yesterday/apps/login/sessions"
	"github.com/tomyedwab/yesterday/database"
)

func DoLogout(db *database.Database, sessionManager *sessions.SessionManager, refreshToken string) error {
	session, err := sessionManager.GetSessionByRefreshToken(refreshToken)
	if err != nil {
		return fmt.Errorf("failed to get session: %v", err)
	}

	err = session.DBDelete(db.GetDB())
	if err != nil {
		return fmt.Errorf("failed to delete session: %v", err)
	}

	return nil
}
