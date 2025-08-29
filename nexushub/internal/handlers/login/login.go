package login

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/apps/admin/types"
	"github.com/tomyedwab/yesterday/nexushub/sessions"
)

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	sessionManager := r.Context().Value(sessions.SessionManagerKey).(*sessions.SessionManager)
	var err error
	body, _ := io.ReadAll(r.Body)

	// Make a cross-service request to the admin service to verify the
	// credentials before creating a new session.
	var loginResponse types.AdminLoginResponse
	statusCode, err := httputils.CrossServiceRequest("/internal/dologin", "18736e4f-93f9-4606-a7be-863c7986ea5b", body, &loginResponse)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to make cross-service request: %v", err), statusCode)
		return
	}

	if !loginResponse.Success {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("invalid username or password"), http.StatusUnauthorized)
		return
	}

	session, err := sessionManager.CreateSession(loginResponse.UserID)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to create login session: %v", err), http.StatusInternalServerError)
		return
	}

	domain := os.Getenv("HOST")
	// Strip port number if it's in the host string
	if strings.Contains(domain, ":") {
		domain = strings.Split(domain, ":")[0]
	}

	w.Header().Set("Set-Cookie", "YRT="+session.RefreshToken+"; Path=/; Domain="+domain+"; HttpOnly; Secure; SameSite=None")
	w.Write([]byte("ok"))
}
