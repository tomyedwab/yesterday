package database

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/tomyedwab/yesterday/applib/events"
	"github.com/tomyedwab/yesterday/applib/httputils"
)

func waitForEventId(w http.ResponseWriter, r *http.Request, eventState *events.EventState) bool {
	eventIdStr := r.URL.Query().Get("e")
	eventId, err := strconv.Atoi(eventIdStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid event ID %s", eventIdStr), http.StatusBadRequest)
		return false
	}
	// Wait for up to 50 seconds while polling eventState.CurrentEventId
	// to see if we have caught up to the requested event ID
	if eventState.PollForEventId(eventId) {
		return true
	}
	// Client is speculatively polling for a new event, and we didn't see
	// one. Return a 304 Not Modified.
	http.Error(w, fmt.Sprintf("Timed out while waiting for event ID %d", eventId), http.StatusNotModified)
	return false
}

func (db *Database) InitHandlers() {
	initialEventId, err := db.CurrentEventV1()
	if err != nil {
		panic(err)
	}
	eventState := events.NewEventState(initialEventId)

	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/api/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
			return
		}
		clientId := r.URL.Query().Get("cid")
		if clientId == "" {
			http.Error(w, "Missing client ID", http.StatusBadRequest)
			return
		}

		buf, err := io.ReadAll(r.Body)
		if err != nil {
			httputils.HandleAPIResponse(w, r, nil, err, http.StatusInternalServerError)
			return
		}

		var event events.GenericEvent
		if err := json.Unmarshal(buf, &event); err != nil {
			httputils.HandleAPIResponse(w, r, nil, err, http.StatusInternalServerError)
			return
		}

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
		httputils.HandleAPIResponse(w, r, map[string]any{"status": "success", "id": newEventId, "clientId": clientId}, err, http.StatusInternalServerError)
	})

	http.HandleFunc("/api/poll", func(w http.ResponseWriter, r *http.Request) {
		if !waitForEventId(w, r, eventState) {
			return
		}
		httputils.HandleAPIResponse(w, r, map[string]any{
			"id":      eventState.CurrentEventId,
			"version": db.version,
		}, nil, http.StatusOK)
	})

	// Special case for internally-generated events
	db.PublishEventCB = func(event events.Event) error {
		buf, err := json.Marshal(event)
		if err != nil {
			return err
		}
		clientId := "internal" + time.Now().Format(time.RFC3339)
		newEventId, err := db.CreateEvent(event.(*events.GenericEvent), buf, clientId)
		if err != nil {
			return err
		}
		eventState.SetCurrentEventId(newEventId)
		return nil
	}

}
