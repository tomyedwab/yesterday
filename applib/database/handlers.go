package database

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/tomyedwab/yesterday/applib/events"
	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/nexushub/types"
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

func (db *Database) InitHandlers() error {
	eventState, err := events.NewEventState(db.GetDB())
	if err != nil {
		return err
	}

	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		StatusInfo := types.ApplicationStatusInfo{
			CurrentEventId: eventState.CurrentEventId,
		}
		httputils.HandleAPIResponse(w, r, StatusInfo, nil, http.StatusOK)
	})

	return nil

	/* TODO STOPSHIP
	initialEventId, err := db.CurrentEventV1()
	if err != nil {
		panic(err)
	}

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
	*/

}
