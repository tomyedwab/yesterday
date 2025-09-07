package applications

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/nexushub/packages"
)

func HandleRegistration(w http.ResponseWriter, r *http.Request, packageManager *packages.PackageManager) {
	var registrationData map[string]string
	err := json.NewDecoder(r.Body).Decode(&registrationData)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to parse registration data: %v", err), http.StatusBadRequest)
		return
	}

	response := make(map[string]*string)
	for appName, hash := range registrationData {
		pkg, err := packageManager.GetPackageByHash(hash)
		if err != nil {
			httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to look up package info: %v", err), http.StatusInternalServerError)
			return
		}
		if pkg != nil {
			response[appName] = &pkg.InstanceID
		} else {
			response[appName] = nil
		}
	}

	httputils.HandleAPIResponse(w, r, response, nil, http.StatusOK)
}
