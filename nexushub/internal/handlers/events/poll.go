package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tomyedwab/yesterday/applib/httputils"
	httpsproxy_types "github.com/tomyedwab/yesterday/nexushub/httpsproxy/types"
	"github.com/tomyedwab/yesterday/nexushub/packages"
)

func HandleEventPoll(w http.ResponseWriter, r *http.Request, packageManager *packages.PackageManager, processManager httpsproxy_types.ProcessManagerInterface) {
	var query map[string]int
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&query)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, err, http.StatusBadRequest)
		return
	}

	response := make(map[string]int, len(query))
	haveUpdates := false

	for instanceID, currentEventID := range query {
		// If someone is polling on events, then we should probably mark the
		// package active
		err = packageManager.SetPackageActive(instanceID, processManager)
		if err != nil {
			httputils.HandleAPIResponse(w, r, nil, err, http.StatusInternalServerError)
			return
		}
		response[instanceID] = processManager.GetEventState(instanceID)
		if response[instanceID] > currentEventID {
			haveUpdates = true
		}
	}

	if haveUpdates {
		httputils.HandleAPIResponse(w, r, response, nil, http.StatusOK)
		return
	}

	expiryTimer := time.After(time.Second * 50)

	cbID, stateUpdated := processManager.AddEventStateCallback()
	defer processManager.RemoveEventStateCallback(cbID)

	for {
		select {
		case cbInfo := <-stateUpdated:
			// Only return if the event ID actually changed
			if _, ok := response[cbInfo.InstanceID]; ok && response[cbInfo.InstanceID] != cbInfo.EventID {
				response[cbInfo.InstanceID] = cbInfo.EventID
				httputils.HandleAPIResponse(w, r, response, nil, http.StatusOK)
				return
			}
		case <-expiryTimer:
			httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("Not modified"), http.StatusNotModified)
			return
		}
	}
}
