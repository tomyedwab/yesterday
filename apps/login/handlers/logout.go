package handlers

import (
	"fmt"
	"net/http"

	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/apps/login/sessions"
)

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	sessionManager := r.Context().Value(sessions.SessionManagerKey).(*sessions.SessionManager)
	var err error

	refreshToken, err := r.Cookie("YRT")
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("missing refresh token"), http.StatusBadRequest)
		return
	}

	session, err := sessionManager.GetSessionByRefreshToken(refreshToken.Value)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("invalid refresh token"), http.StatusBadRequest)
		return
	}

	err = sessionManager.DeleteSessionsForUser(session.UserID)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to delete sessions: %v", err), http.StatusInternalServerError)
		return
	}

	httputils.HandleAPIResponse(w, r, nil, nil, http.StatusOK)
}
