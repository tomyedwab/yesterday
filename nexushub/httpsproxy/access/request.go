package access

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/tomyedwab/yesterday/apps/login/types"
	httpsproxy_types "github.com/tomyedwab/yesterday/nexushub/httpsproxy/types"
	"github.com/tomyedwab/yesterday/nexushub/processes"
)

func HandleAccessTokenRequest(
	pm httpsproxy_types.ProcessManagerInterface,
	instance *processes.AppInstance,
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
) int {
	loginInstance, loginPort, err := pm.GetAppInstanceByID("3bf3e3c0-6e51-482a-b180-00f6aa568ee9")
	if err != nil {
		log.Printf("<%s> Error resolving login service: %v", traceID, err)
		http.Error(w, "Login service not found", http.StatusNotFound)
		return http.StatusNotFound
	}
	if loginInstance == nil {
		log.Printf("<%s> No active login instance found", traceID)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return http.StatusServiceUnavailable
	}

	// Get refresh token from cookie
	refreshToken, err := r.Cookie("YRT")
	if err != nil {
		// There is no refresh token cookie, redirect to login service
		respJson, _ := json.Marshal(map[string]string{
			"login_url": fmt.Sprintf("https://%s/", loginInstance.HostName),
		})
		w.WriteHeader(http.StatusOK)
		w.Write(respJson)
		return http.StatusOK
	}

	req := types.AccessTokenRequest{
		ApplicationID: instance.InstanceID,
		RefreshToken:  refreshToken.Value,
	}
	reqJson, _ := json.Marshal(req)
	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/internal/access_token", loginPort), "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		http.Error(w, fmt.Sprintf("error while refreshing token: %v", err), http.StatusServiceUnavailable)
		return http.StatusServiceUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized {
			respJson, _ := json.Marshal(map[string]string{
				"login_url": fmt.Sprintf("https://%s/", loginInstance.HostName),
			})
			w.WriteHeader(http.StatusOK)
			w.Write(respJson)
			return http.StatusOK
		}
		http.Error(w, fmt.Sprintf("error while refreshing token: %v: %s", resp.Status, string(message)), resp.StatusCode)
		return resp.StatusCode
	}
	var tokenResponse types.AccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		http.Error(w, fmt.Sprintf("error while decoding token response: %v", err), http.StatusInternalServerError)
		return http.StatusInternalServerError
	}

	createAccessToken(&tokenResponse)

	// Set the cookie with the refresh token
	targetDomain := r.Host
	if strings.Contains(targetDomain, ":") {
		targetDomain = strings.Split(targetDomain, ":")[0]
	}
	w.Header().Set("Set-Cookie", "YRT="+tokenResponse.RefreshToken+"; Path=/; Domain="+targetDomain+"; HttpOnly; Secure; SameSite=None")
	w.WriteHeader(http.StatusOK)

	respJson, _ := json.Marshal(map[string]string{
		"access_token": tokenResponse.AccessToken,
	})
	w.Write(respJson)
	return http.StatusOK
}
