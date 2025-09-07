package applications

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tomyedwab/yesterday/applib/httputils"
)

func HandleRegistration(w http.ResponseWriter, r *http.Request) {
	var registrationData map[string]string
	err := json.NewDecoder(r.Body).Decode(&registrationData)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to parse registration data: %v", err), http.StatusBadRequest)
		return
	}

	response := make(map[string]*string)
	for appName, _ := range registrationData {
		response[appName] = nil
	}

	httputils.HandleAPIResponse(w, r, response, nil, http.StatusOK)
}
