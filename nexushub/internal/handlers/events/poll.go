package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tomyedwab/yesterday/applib/httputils"
	httpsproxy_types "github.com/tomyedwab/yesterday/nexushub/httpsproxy/types"
)

func HandleEventPoll(w http.ResponseWriter, r *http.Request, processManager httpsproxy_types.ProcessManagerInterface) {
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
		response[instanceID] = processManager.GetEventState(instanceID)
		if response[instanceID] > currentEventID {
			haveUpdates = true
		}
	}

	if haveUpdates {
		httputils.HandleAPIResponse(w, r, response, nil, http.StatusOK)
		return
	}

	// Sleep until poll window is done
	// TODO(tom) Return early when event IDs change
	time.Sleep(time.Second * 50)

	httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("Not modified"), http.StatusNotModified)
}
