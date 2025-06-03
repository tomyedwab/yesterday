package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"

	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
	"github.com/tomyedwab/yesterday/apps/login/sessions"
	"github.com/tomyedwab/yesterday/wasi/guest"
	"github.com/tomyedwab/yesterday/wasi/types"
)

func handle_appinfo(params types.RequestParams) types.Response {
	// Make a cross-service request to the admin service
	var appInfoResponse admin_types.AdminAppInfoResponse
	statusCode, err := guest.CrossServiceRequest("/internal/appinfo", "18736e4f-93f9-4606-a7be-863c7986ea5b", []byte(params.Body), &appInfoResponse)
	if err != nil {
		return guest.RespondError(statusCode, fmt.Errorf("failed to make cross-service request: %v", err))
	}

	return guest.RespondSuccess(appInfoResponse.ApplicationHostName)
}

func handle_login(sessionManager *sessions.SessionManager, params types.RequestParams) types.Response {
	// Make a cross-service request to the admin service
	var loginResponse admin_types.AdminLoginResponse
	statusCode, err := guest.CrossServiceRequest("/internal/dologin", "18736e4f-93f9-4606-a7be-863c7986ea5b", []byte(params.Body), &loginResponse)
	if err != nil {
		return guest.RespondError(statusCode, fmt.Errorf("failed to make cross-service request: %v", err))
	}

	if !loginResponse.Success {
		return guest.RespondError(http.StatusUnauthorized, fmt.Errorf("invalid username or password"))
	}

	session, err := sessionManager.CreateSession(loginResponse.UserID, loginResponse.ApplicationID)
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("failed to create session"))
	}

	responseJson, err := json.Marshal(map[string]string{
		"domain": loginResponse.ApplicationHostName,
	})
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("Failed to marshal response: %v", err))
	}

	return guest.RespondSuccessWithHeaders(string(responseJson), map[string]string{
		"Set-Cookie": "YRT=" + session.RefreshToken + "; Path=/; Domain=" + loginResponse.ApplicationHostName + "; HttpOnly; Secure; SameSite=None",
	})
}

//go:wasmexport init
func init() {
	guest.Init()
	db, err := sqlx.Connect("sqlproxy", "")
	if err != nil {
		log.Fatal(err)
	}
	sessionManager, err := sessions.NewManager(db, 10*time.Minute, 1*time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	guest.RegisterHandler("/api/appinfo", handle_appinfo)
	guest.RegisterHandler("/api/login", func(params types.RequestParams) types.Response {
		return handle_login(sessionManager, params)
	})
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
