package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tomyedwab/yesterday/applib/httputils"
	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
)

func HandleCheckAccess(w http.ResponseWriter, r *http.Request) {
	var request admin_types.AccessRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("error parsing request: %v", err), http.StatusBadRequest)
		return
	}

	// TODO(tom): user-application permissions check
	httputils.HandleAPIResponse(w, r, admin_types.AccessResponse{
		AccessGranted: true,
	}, nil, http.StatusOK)
}
