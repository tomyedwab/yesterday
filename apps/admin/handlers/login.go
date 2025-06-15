package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/applib"
	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/apps/admin/state"
	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
)

func HandleDoLogin(w http.ResponseWriter, r *http.Request) {
	db := r.Context().Value(applib.ContextSqliteDatabaseKey).(*sqlx.DB)

	var request admin_types.AdminLoginRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("error parsing request: %v", err), http.StatusBadRequest)
		return
	}

	fmt.Printf("Attempting login for user: %s\n", request.Username)
	user, err := state.GetUser(db, request.Username)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to get user %s: %w", request.Username, err), http.StatusBadRequest)
		return
	}

	hasher := sha256.New()
	hasher.Write([]byte(user.Salt + request.Password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	if user.PasswordHash != passwordHash {
		fmt.Printf("Invalid password for user %s\n", request.Username)
		httputils.HandleAPIResponse(w, r, admin_types.AdminLoginResponse{
			Success: false,
		}, nil, http.StatusOK)
	}

	httputils.HandleAPIResponse(w, r, admin_types.AdminLoginResponse{
		Success: true,
		UserID:  user.ID,
	}, nil, http.StatusOK)
}
