package database

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

type EventState struct {
	// The current application event ID
	CurrentEventId int
}

func NewEventState(db *sqlx.DB) (*EventState, error) {
	// Create users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS event_state (
		    id INTEGER PRIMARY KEY,
			current_event_id INTEGER
		)`)
	if err != nil {
		return nil, fmt.Errorf("failed to create event state table: %w", err)
	}

	// Create event state with event ID set to zero
	_, err = db.Exec(`
		INSERT INTO event_state (id, current_event_id)
		SELECT 0, 0
		ON CONFLICT DO NOTHING
		`)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event state: %w", err)
	}

	var initialEventId int
	err = db.Get(&initialEventId, `SELECT current_event_id FROM event_state WHERE id = 0`)
	if err != nil {
		return nil, fmt.Errorf("failed to get event state: %w", err)
	}

	log.Printf("Initialized event state with ID %d\n", initialEventId)

	return &EventState{
		CurrentEventId: initialEventId,
	}, nil
}

func (state *EventState) SetCurrentEventId(eventId int, tx *sqlx.Tx) error {
	_, err := tx.Exec(`UPDATE event_state SET current_event_id = $1 WHERE id = 0`, eventId)
	state.CurrentEventId = eventId
	return err
}
