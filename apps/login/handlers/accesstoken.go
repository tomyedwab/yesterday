package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tomyedwab/yesterday/applib/httputils"
	admin_types "github.com/tomyedwab/yesterday/apps/admin/types"
	"github.com/tomyedwab/yesterday/apps/login/sessions"
	"github.com/tomyedwab/yesterday/apps/login/types"
)

func HandleAccessToken(w http.ResponseWriter, r *http.Request) {
	sessionManager := r.Context().Value(sessions.SessionManagerKey).(*sessions.SessionManager)

	var tokenRequest types.AccessTokenRequest
	err := json.NewDecoder(r.Body).Decode(&tokenRequest)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	session, err := sessionManager.GetSessionByRefreshToken(tokenRequest.RefreshToken)
	if err != nil || session == nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("invalid refresh token"), http.StatusUnauthorized)
		return
	}

	accessRequestJson, _ := json.Marshal(&admin_types.AccessRequest{
		UserID:        session.UserID,
		ApplicationID: tokenRequest.ApplicationID,
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

	response, err := sessionManager.CreateAccessToken(session, &tokenRequest)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, err, http.StatusUnauthorized)
		return
	}

	httputils.HandleAPIResponse(w, r, response, nil, http.StatusOK)
}
