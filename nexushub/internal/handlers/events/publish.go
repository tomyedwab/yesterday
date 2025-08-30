package events

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/nexushub/events"
	"github.com/tomyedwab/yesterday/nexushub/types"
)

func HandleEventPublish(w http.ResponseWriter, r *http.Request, eventManager *events.EventManager) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, err, http.StatusInternalServerError)
		return
	}

	var publishData types.EventPublishData
	if err := json.Unmarshal(buf, &publishData); err != nil {
		httputils.HandleAPIResponse(w, r, nil, err, http.StatusInternalServerError)
		return
	}

	newEventId, err := eventManager.PublishEvent(publishData.ClientID, publishData.Type, buf)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, err, http.StatusInternalServerError)
		return
	}

	/*
		newEventId, err := db.CreateEvent(&event, buf, clientId)
		if err == nil {
			eventState.SetCurrentEventId(newEventId)
		}
		if err != nil {
			// Special case for duplicate errors. We return a 200 in this case.
			if duplicateErr, ok := err.(*DuplicateEventError); ok {
				httputils.HandleAPIResponse(w, r, map[string]any{"status": "duplicate", "id": duplicateErr.Id, "clientId": clientId}, nil, http.StatusOK)
				return
			}
		}
	*/
	httputils.HandleAPIResponse(w, r, map[string]any{"status": "success", "id": newEventId, "clientId": publishData.ClientID}, err, http.StatusInternalServerError)
}
