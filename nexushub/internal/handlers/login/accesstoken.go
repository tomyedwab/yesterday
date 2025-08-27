package login

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tomyedwab/yesterday/applib/httputils"
	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
	"github.com/tomyedwab/yesterday/nexushub/httpsproxy/access"
	"github.com/tomyedwab/yesterday/nexushub/processes"
	"github.com/tomyedwab/yesterday/nexushub/sessions"
)

func HandleAccessToken(w http.ResponseWriter, r *http.Request, host string, instance *processes.AppInstance) {
	sessionManager := r.Context().Value(sessions.SessionManagerKey).(*sessions.SessionManager)

	// Get refresh token from cookie
	refreshToken, err := r.Cookie("YRT")
	if err != nil {
		// There is no refresh token cookie, redirect to login page
		// TODO(tom) STOPSHIP this is not going to be correct in general
		respJson, _ := json.Marshal(map[string]string{
			"login_url": fmt.Sprintf("https://login.%s/", host),
		})
		w.WriteHeader(http.StatusOK)
		w.Write(respJson)
		return
	}

	session, err := sessionManager.GetSessionByRefreshToken(refreshToken.Value)
	if err != nil || session == nil {
		// Invalid refresh token, redirect to login page
		// TODO(tom) STOPSHIP this is not going to be correct in general
		respJson, _ := json.Marshal(map[string]string{
			"login_url": fmt.Sprintf("https://login.%s/", host),
		})
		w.WriteHeader(http.StatusOK)
		w.Write(respJson)
		return
	}

	accessRequestJson, _ := json.Marshal(&admin_types.AccessRequest{
		UserID:        session.UserID,
		ApplicationID: instance.InstanceID,
	})
	var accessResponse admin_types.AccessResponse
	statusCode, err := httputils.CrossServiceRequest("/internal/checkAccess", "18736e4f-93f9-4606-a7be-863c7986ea5b", accessRequestJson, &accessResponse)
	if err != nil || statusCode != http.StatusOK {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to make cross-service request: %v", err), statusCode)
		return
	}
	if !accessResponse.AccessGranted {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("access denied"), http.StatusForbidden)
		return
	}

	response, err := sessionManager.CreateAccessToken(session, instance.InstanceID)
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
