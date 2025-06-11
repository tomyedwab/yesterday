package login

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/apps/admin/state"
	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
	"github.com/tomyedwab/yesterday/wasi/guest"
	"github.com/tomyedwab/yesterday/wasi/types"
)

func HandleDoLogin(params types.RequestParams) types.Response {
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

	ret := admin_types.AdminLoginResponse{
		Success: true,
		UserID:  user.ID,
	}

	retJson, err := json.Marshal(ret)
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("error marshaling response: %v", err))
	}
	return guest.RespondSuccess(string(retJson))
}

func HandleCheckAccess(params types.RequestParams) types.Response {
	db, err := sqlx.Connect("sqlproxy", "")
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("sqlx.Connect failed: %v", err))
	}
	defer db.Close()

	var request admin_types.AccessRequest
	err = json.Unmarshal([]byte(params.Body), &request)
	if err != nil {
		return guest.RespondError(http.StatusBadRequest, fmt.Errorf("error parsing request: %v", err))
	}

	accessGranted, err := state.CheckUserAccess(db, request.ApplicationID, request.UserID)
	if err != nil {
		return guest.RespondError(http.StatusInternalServerError, fmt.Errorf("error checking user access: %v", err))
	}
	responseJson, _ := json.Marshal(admin_types.AccessResponse{
		AccessGranted: accessGranted,
	})
	return guest.RespondSuccess(string(responseJson))
}
