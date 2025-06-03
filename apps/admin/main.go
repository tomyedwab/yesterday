package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/apps/admin/state"
	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/wasi/guest"
	"github.com/tomyedwab/yesterday/wasi/types"
)

func handle_appinfo(params types.RequestParams) types.Response {
	db, err := sqlx.Connect("sqlproxy", "")
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("sqlx.Connect failed: %v", err))
	}
	defer db.Close()

	var request admin_types.AdminAppInfoRequest
	err = json.Unmarshal([]byte(params.Body), &request)
	if err != nil {
		return guest.RespondError(http.StatusBadRequest, fmt.Errorf("error parsing request: %v", err))
	}

	application, err := state.GetApplication(db, request.ApplicationID)
	if err != nil {
		return guest.RespondError(http.StatusBadRequest, fmt.Errorf("invalid application ID %s", request.ApplicationID))
	}

	ret := admin_types.AdminAppInfoResponse{
		ApplicationHostName: application.HostName,
	}

	retJson, err := json.Marshal(ret)
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("error marshaling response: %v", err))
	}
	return guest.RespondSuccess(string(retJson))
}

func handle_dologin(params types.RequestParams) types.Response {
	db, err := sqlx.Connect("sqlproxy", "")
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("sqlx.Connect failed: %v", err))
	}
	defer db.Close()

	var request admin_types.AdminLoginRequest
	err = json.Unmarshal([]byte(params.Body), &request)
	if err != nil {
		return guest.RespondError(http.StatusBadRequest, fmt.Errorf("error parsing request: %v", err))
	}

	application, err := state.GetApplication(db, request.ApplicationID)
	if err != nil {
		return guest.RespondError(http.StatusBadRequest, fmt.Errorf("invalid application ID %s", request.ApplicationID))
	}

	fmt.Printf("Attempting login for user: %s\n", request.Username)
	user, err := state.GetUser(db, request.Username)
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("failed to get user %s: %w", request.Username, err))
	}

	hasher := sha256.New()
	hasher.Write([]byte(user.Salt + request.Password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	if user.PasswordHash != passwordHash {
		fmt.Printf("Invalid password for user %s\n", request.Username)
		ret := admin_types.AdminLoginResponse{
			Success: false,
		}
		retJson, err := json.Marshal(ret)
		if err != nil {
			return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("error marshaling response: %v", err))
		}
		return guest.RespondSuccess(string(retJson))
	}

	profile, err := state.GetUserProfile(db, user.ID, request.ApplicationID)
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("failed to get user profile: %v", err))
	}
	if profile == nil {
		fmt.Printf("User %s does not have access to application %s\n", request.Username, request.ApplicationID)
		ret := admin_types.AdminLoginResponse{
			Success: false,
		}
		retJson, err := json.Marshal(ret)
		if err != nil {
			return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("error marshaling response: %v", err))
		}
		return guest.RespondSuccess(string(retJson))
	}

	ret := admin_types.AdminLoginResponse{
		Success:             true,
		UserID:              user.ID,
		ApplicationID:       request.ApplicationID,
		ApplicationHostName: application.HostName,
	}

	retJson, err := json.Marshal(ret)
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("error marshaling response: %v", err))
	}
	return guest.RespondSuccess(string(retJson))
}

//go:wasmexport init
func init() {
	guest.Init()
	guest.RegisterEventHandler(events.DBInitEventType, state.ApplicationsHandleInitEvent)
	guest.RegisterEventHandler(events.DBInitEventType, state.UsersHandleInitEvent)
	guest.RegisterEventHandler(events.DBInitEventType, state.UserProfilesHandleInitEvent)
	guest.RegisterHandler("/internal/appinfo", handle_appinfo)
	guest.RegisterHandler("/internal/dologin", handle_dologin)
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
