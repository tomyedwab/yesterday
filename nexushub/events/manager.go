package events

import (
	"log"

	"github.com/jmoiron/sqlx"
)

type EventManager struct {
	DB             *sqlx.DB
	LatestEventIds map[string]int
}

func CreateEventManager(db *sqlx.DB) (*EventManager, error) {
	err := EventDBInit(db)
	if err != nil {
		return nil, err
	}
	latestEventIds, err := EventDBGetCurrentEventIDs(db)
	if err != nil {
		return nil, err
	}
	maxEventId := 0
	for _, eventId := range latestEventIds {
		if eventId > maxEventId {
			maxEventId = eventId
		}
	}
	log.Printf("Event manager created. Latest event ID: %d", maxEventId)
	return &EventManager{
		DB:             db,
		LatestEventIds: latestEventIds,
	}, nil
}

func (em *EventManager) PublishEvent(clientID, eventType string, payload []byte) (int, error) {
	// TODO STOPSHIP: Validate event against schema

	newEventId, err := EventDBCreateEvent(em.DB, payload, clientID, eventType)
	if err != nil {
		return 0, err
	}
	em.LatestEventIds[eventType] = newEventId
	log.Printf("Published event of type %s with ID %d", eventType, newEventId)
	return newEventId, nil
}

func (em *EventManager) GetCurrentEventID(eventType string) int {
	return em.LatestEventIds[eventType]
}

func (em *EventManager) GetEvent(eventId int) (string, []byte, error) {
	return EventDBGetEvent(em.DB, eventId)
}
