package database

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	http.HandleFunc("/internal/publish_events", func(w http.ResponseWriter, r *http.Request) {
		// Batch event processing
		var batchRequest struct {
			Events []struct {
				ID   int    `json:"id"`
				Type string `json:"type"`
				Data string `json:"data"`
			} `json:"events"`
		}

		if err := json.NewDecoder(r.Body).Decode(&batchRequest); err != nil {
			http.Error(w, fmt.Sprintf("Failed to decode batch request: %v", err), http.StatusBadRequest)
			return
		}

		if len(batchRequest.Events) == 0 {
			http.Error(w, "No events in batch", http.StatusBadRequest)
			return
		}

		// Process events in order
		lastProcessedId := 0
		for _, event := range batchRequest.Events {
			if db.eventState.CurrentEventId >= event.ID {
				continue // Skip already processed events
			}

			if err := db.HandleEvent(event.ID, event.Type, []byte(event.Data)); err != nil {
				http.Error(w, fmt.Sprintf("Failed to publish event ID %d: %v", event.ID, err), http.StatusInternalServerError)
				return
			}
			lastProcessedId = event.ID
		}

		if lastProcessedId > 0 {
			http.Error(w, fmt.Sprintf("Published batch of events up to ID %d", lastProcessedId), http.StatusCreated)
		} else {
			http.Error(w, "All events in batch were already processed", http.StatusOK)
		}
	})

	return nil
}
