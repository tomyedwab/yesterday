package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
	"github.com/tomyedwab/yesterday/apps/login/sessions"
	login_types "github.com/tomyedwab/yesterday/apps/login/types"
	"github.com/tomyedwab/yesterday/wasi/guest"
	"github.com/tomyedwab/yesterday/wasi/types"
)

func handle_login(sessionManager *sessions.SessionManager, params types.RequestParams) types.Response {
	var loginSession *sessions.Session
	var err error
	if refreshToken, ok := params.Cookies["YRT"]; params.Body == "" && ok {
		// If a refresh token is passed in, then the user is already logged in,
		// so just look up their login session and generate a new session for
		// the application
		loginSession, err = sessionManager.GetSessionByRefreshToken(refreshToken)
		if err != nil {
			return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("failed to look up refresh token: %v", err))
		}
		if loginSession == nil {
			return guest.RespondError(http.StatusUnauthorized, fmt.Errorf("invalid refresh token"))
		}
	} else {
		// If there is no refresh token, then we expect a username and password.
		// Make a cross-service request to the admin service to verify the
		// credentials before creating a new login session.
		var loginResponse admin_types.AdminLoginResponse
		statusCode, err := guest.CrossServiceRequest("/internal/dologin", "18736e4f-93f9-4606-a7be-863c7986ea5b", []byte(params.Body), &loginResponse)
		if err != nil {
			return guest.RespondError(statusCode, fmt.Errorf("failed to make cross-service request: %v", err))
		}

		if !loginResponse.Success {
			return guest.RespondError(http.StatusUnauthorized, fmt.Errorf("invalid username or password"))
		}

		loginSession, err = sessionManager.CreateSession(loginResponse.UserID, "login")
		if err != nil {
			return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("failed to create login session: %v", err))
		}
	}

	appSession, err := sessionManager.CreateSession(loginSession.UserID, "app")
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("failed to create app session: %v", err))
	}

	domain := guest.GetEnv("HOST")
	// Strip port number if it's in the host string
	if strings.Contains(domain, ":") {
		domain = strings.Split(domain, ":")[0]
	}

	responseJson, _ := json.Marshal(map[string]string{
		"domain":            domain,
		"app_refresh_token": appSession.RefreshToken,
	})
	return guest.RespondSuccessWithHeaders(string(responseJson), map[string]string{
		"Set-Cookie": "YRT=" + loginSession.RefreshToken + "; Path=/; Domain=" + domain + "; HttpOnly; Secure; SameSite=None",
	})
}

func handle_access_token(sessionManager *sessions.SessionManager, params types.RequestParams) types.Response {
	var tokenRequest login_types.AccessTokenRequest
	err := json.Unmarshal([]byte(params.Body), &tokenRequest)
	if err != nil {
		return guest.RespondError(http.StatusBadRequest, fmt.Errorf("invalid request body: %v", err))
	}

	session, err := sessionManager.GetSessionByRefreshToken(tokenRequest.RefreshToken)
	if err != nil || session == nil {
		return guest.RespondError(http.StatusUnauthorized, fmt.Errorf("invalid refresh token"))
	}

	accessRequestJson, _ := json.Marshal(&admin_types.AccessRequest{
		UserID:        session.UserID,
		ApplicationID: tokenRequest.ApplicationID,
	})
	var accessResponse admin_types.AccessResponse
	statusCode, err := guest.CrossServiceRequest("/internal/checkAccess", "18736e4f-93f9-4606-a7be-863c7986ea5b", accessRequestJson, &accessResponse)
	if err != nil || statusCode != http.StatusOK {
		return guest.RespondError(statusCode, fmt.Errorf("failed to make cross-service request: %v", err))
	}
	if !accessResponse.AccessGranted {
		return guest.RespondError(http.StatusForbidden, fmt.Errorf("access denied"))
	}

	response, err := sessionManager.CreateAccessToken(session, &tokenRequest)
	if err != nil {
		return guest.RespondError(http.StatusUnauthorized, err)
	}

	responseJson, _ := json.Marshal(response)
	return guest.RespondSuccess(string(responseJson))
}

//go:wasmexport init
func init() {
	guest.Init("0.0.1")
	db := guest.NewDB()
	sessionManager, err := sessions.NewManager(db, 10*time.Minute, 1*time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	guest.RegisterHandler("/public/login", func(params types.RequestParams) types.Response {
		return handle_login(sessionManager, params)
	})
	guest.RegisterHandler("/internal/access_token", func(params types.RequestParams) types.Response {
		return handle_access_token(sessionManager, params)
	})
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
