package login

import (
	"fmt"
	"net/http"

	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/nexushub/audit"
	"github.com/tomyedwab/yesterday/nexushub/sessions"
)

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	sessionManager := r.Context().Value(sessions.SessionManagerKey).(*sessions.SessionManager)
	auditLogger := r.Context().Value(audit.AuditLoggerKey).(*audit.Logger)
	var err error

	refreshToken, err := r.Cookie("YRT")
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("missing refresh token"), http.StatusBadRequest)
		return
	}

	session, err := sessionManager.GetSessionByRefreshToken(refreshToken.Value)
	if err != nil {
		// Log invalid refresh token attempt
		if auditErr := auditLogger.LogInvalidRefreshToken(refreshToken.Value); auditErr != nil {
			fmt.Printf("Failed to log invalid refresh token audit event: %v\n", auditErr)
		}
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("invalid refresh token"), http.StatusBadRequest)
		return
	}

	// Log logout before deleting the session
	if err := auditLogger.LogLogout(session.UserID, refreshToken.Value); err != nil {
		// Log the error but don't fail the request
		fmt.Printf("Failed to log logout audit event: %v\n", err)
	}

	err = sessionManager.DeleteSessionsForUser(session.UserID)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to delete sessions: %v", err), http.StatusInternalServerError)
		return
	}

	httputils.HandleAPIResponse(w, r, nil, nil, http.StatusOK)
}
