package database

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/database/middleware"
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

func HandleAPIResponse(w http.ResponseWriter, r *http.Request, resp interface{}, err error) {
	if err != nil {
		fmt.Printf("%s - %s %s ERROR: %v\n",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			err,
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json, err := json.Marshal(resp)
	if err != nil {
		fmt.Printf("%s - %s %s ERROR: %v\n",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			err,
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func (db *Database) InitHandlers(mapper events.MapEventType) {
	initialEventId, err := db.CurrentEventV1()
	if err != nil {
		panic(err)
	}
	eventState := events.NewEventState(initialEventId)

	http.HandleFunc("/api/status", middleware.ApplyDefault(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "ok")
		},
	))

	http.HandleFunc("/api/publish", middleware.ApplyDefault(
		func(w http.ResponseWriter, r *http.Request) {
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
				HandleAPIResponse(w, r, nil, err)
			}
			event, err := events.ParseEvent(buf, mapper)
			if err != nil {
				HandleAPIResponse(w, r, nil, err)
			}

			newEventId, err := db.CreateEvent(event, buf, clientId)
			if err == nil {
				eventState.SetCurrentEventId(newEventId)
			}
			if err != nil {
				// Special case for duplicate errors. We return a 200 in this case.
				if duplicateErr, ok := err.(*DuplicateEventError); ok {
					HandleAPIResponse(w, r, map[string]interface{}{"status": "duplicate", "id": duplicateErr.Id, "clientId": clientId}, nil)
					return
				}
			}
			HandleAPIResponse(w, r, map[string]interface{}{"status": "success", "id": newEventId, "clientId": clientId}, err)
		},
	))

	http.HandleFunc("/api/poll", middleware.ApplyDefault(
		func(w http.ResponseWriter, r *http.Request) {
			if !waitForEventId(w, r, eventState) {
				return
			}
			HandleAPIResponse(w, r, map[string]interface{}{
				"id":      eventState.CurrentEventId,
				"version": db.version,
			}, nil)
		},
	))

}
