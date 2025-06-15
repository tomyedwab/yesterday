package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/applib"
	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/apps/admin/state"
	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
)

func HandleCheckAccess(w http.ResponseWriter, r *http.Request) {
	db := r.Context().Value(applib.ContextSqliteDatabaseKey).(*sqlx.DB)

	var request admin_types.AccessRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("error parsing request: %v", err), http.StatusBadRequest)
		return
	}

	accessGranted, err := state.CheckUserAccess(db, request.ApplicationID, request.UserID)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("error checking user access: %v", err), http.StatusInternalServerError)
		return
	}
	httputils.HandleAPIResponse(w, r, admin_types.AccessResponse{
		AccessGranted: accessGranted,
	}, nil, http.StatusOK)
}
