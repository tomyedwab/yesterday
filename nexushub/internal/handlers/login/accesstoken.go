package login

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tomyedwab/yesterday/applib/httputils"
	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
	"github.com/tomyedwab/yesterday/nexushub/httpsproxy/access"
	"github.com/tomyedwab/yesterday/nexushub/sessions"
)

func HandleAccessToken(w http.ResponseWriter, r *http.Request, adminServiceHost string) {
	sessionManager := r.Context().Value(sessions.SessionManagerKey).(*sessions.SessionManager)

	// Get refresh token from cookie
	refreshToken, err := r.Cookie("YRT")
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("missing refresh token"), http.StatusForbidden)
		return
	}

	session, err := sessionManager.GetSessionByRefreshToken(refreshToken.Value)
	if err != nil || session == nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("refresh token not found"), http.StatusForbidden)
		return
	}

	body, _ := json.Marshal(&admin_types.AccessRequest{
		UserID: session.UserID,
	})
	var accessResponse admin_types.AccessResponse
	resp, err := http.Post(adminServiceHost+"/internal/checkAccess", "application/json", io.NopCloser(bytes.NewReader([]byte(body))))
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to make cross-service request: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to make cross-service request: %v", err), resp.StatusCode)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&accessResponse)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to make cross-service request: %v", err), http.StatusInternalServerError)
		return
	}
	if !accessResponse.AccessGranted {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("access denied"), http.StatusForbidden)
		return
	}

	response, err := sessionManager.CreateAccessToken(session)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, err, http.StatusUnauthorized)
		return
	}

	access.CreateAccessToken(response)

	// Set the cookie with the refresh token
	targetDomain := r.Host
	if strings.Contains(targetDomain, ":") {
		targetDomain = strings.Split(targetDomain, ":")[0]
	}
	w.Header().Set("Set-Cookie", "YRT="+response.RefreshToken+"; Path=/; Domain="+targetDomain+"; HttpOnly; Secure; SameSite=None")
	w.WriteHeader(http.StatusOK)

	respJson, _ := json.Marshal(map[string]string{
		"access_token": response.AccessToken,
	})
	w.Write(respJson)
}
