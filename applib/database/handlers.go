package database

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/nexushub/types"
)

func (db *Database) InitHandlers() error {
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		StatusInfo := types.ApplicationStatusInfo{
			CurrentEventId: db.eventState.CurrentEventId,
		}
		httputils.HandleAPIResponse(w, r, StatusInfo, nil, http.StatusOK)
	})

	http.HandleFunc("/internal/publish_event", func(w http.ResponseWriter, r *http.Request) {
		eventType := r.URL.Query().Get("type")
		if eventType == "" {
			http.Error(w, "Missing event type", http.StatusBadRequest)
			return
		}
		eventId := r.URL.Query().Get("id")
		if eventId == "" {
			http.Error(w, "Missing event ID", http.StatusBadRequest)
			return
		}
		eventIdInt, err := strconv.ParseInt(eventId, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid event ID %s", eventId), http.StatusBadRequest)
			return
		}
		if db.eventState.CurrentEventId >= int(eventIdInt) {
			http.Error(w, fmt.Sprintf("Event ID %d already published", eventIdInt), http.StatusConflict)
			return
		}

		eventData, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read event data: %v", err), http.StatusInternalServerError)
			return
		}

		if err := db.HandleEvent(int(eventIdInt), eventType, eventData); err != nil {
			http.Error(w, fmt.Sprintf("Failed to publish event ID %d: %v", eventIdInt, err), http.StatusInternalServerError)
			return
		}
		http.Error(w, fmt.Sprintf("Published event ID %d", eventIdInt), http.StatusCreated)
	})

	return nil
}
