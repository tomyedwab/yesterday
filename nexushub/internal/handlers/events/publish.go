package events

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/nexushub/events"
	httpsproxy_types "github.com/tomyedwab/yesterday/nexushub/httpsproxy/types"
	"github.com/tomyedwab/yesterday/nexushub/types"
)

func HandleEventPublish(w http.ResponseWriter, r *http.Request, eventManager *events.EventManager, processManager httpsproxy_types.ProcessManagerInterface) {
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

	processManager.EventPublished()

	httputils.HandleAPIResponse(w, r, map[string]any{"status": "success", "id": newEventId, "clientId": publishData.ClientID}, err, http.StatusInternalServerError)
}
