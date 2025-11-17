package login

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/apps/admin/types"
	"github.com/tomyedwab/yesterday/nexushub/audit"
	"github.com/tomyedwab/yesterday/nexushub/sessions"
)

func HandleLogin(w http.ResponseWriter, r *http.Request, adminServiceHost string) {
	sessionManager := r.Context().Value(sessions.SessionManagerKey).(*sessions.SessionManager)
	auditLogger := r.Context().Value(audit.AuditLoggerKey).(*audit.Logger)
	var err error
	body, _ := io.ReadAll(r.Body)

	// Make a service request to the admin service to verify the credentials
	// before creating a new session.
	var loginResponse types.AdminLoginResponse
	resp, err := http.Post(adminServiceHost+"/internal/dologin", "application/json", io.NopCloser(bytes.NewReader([]byte(body))))
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to make cross-service request: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to make cross-service request: %v", err), resp.StatusCode)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to make cross-service request: %v", err), http.StatusInternalServerError)
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

	domain := r.Host
	// Strip port number if it's in the host string
	if strings.Contains(domain, ":") {
		domain = strings.Split(domain, ":")[0]
	}

	// Log successful login
	if err := auditLogger.LogLogin(loginResponse.UserID, session.RefreshToken); err != nil {
		// Log the error but don't fail the request
		fmt.Printf("Failed to log login audit event: %v\n", err)
	}

	w.Header().Set("Set-Cookie", "YRT="+session.RefreshToken+"; Path=/; Domain="+domain+"; HttpOnly; Secure; SameSite=None")
	w.Write([]byte("ok"))
}
