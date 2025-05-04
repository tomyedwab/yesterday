package database

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type DuplicateEventError struct {
	Id       int
	ClientId string
}

func (e *DuplicateEventError) Error() string {
	return fmt.Sprintf("Duplicate event with client ID %s", e.ClientId)
}

const eventSchema = `
CREATE TABLE IF NOT EXISTS event_v1 (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	client_id TEXT NOT NULL,
	event_data JSONB NOT NULL
);
`

const getEventByClientIdV1Sql = `
SELECT id FROM event_v1 WHERE client_id = $1;
`

const insertEventV1Sql = `
INSERT INTO event_v1 (event_data, client_id)
VALUES ($1, $2)
RETURNING id;
`

const getLatestEventIdV1Sql = `
SELECT COALESCE(MAX(id), 0) FROM event_v1;
`

// EventDBInit initializes the event database schema. All events are stored as
// JSON blobs in the event_v1 table.
func EventDBInit(tx *sqlx.Tx) error {
	_, err := tx.Exec(eventSchema)
	return err
}

// EventDB inserts a new event into the events table.
func EventDBCreateEvent(tx *sqlx.Tx, eventData []byte, clientId string) (int, error) {
	fmt.Println("Event v1: Create event")
	eventId := 0

	// First, a quick check if an event with the given client ID already exists
	// in the database. If so, this is a duplicate event, so just return the
	// existing ID.
	err := tx.Get(&eventId, getEventByClientIdV1Sql, clientId)
	if err == nil {
		return 0, &DuplicateEventError{
			Id:       int(eventId),
			ClientId: clientId,
		}
	}
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	err = tx.QueryRow(insertEventV1Sql, eventData, clientId).Scan(&eventId)
	if err != nil {
		return 0, err
	}
	fmt.Printf("Created new event with ID %d\n", eventId)
	return int(eventId), nil
}

func (db *Database) CurrentEventV1() (int, error) {
	var id int
	err := db.db.Get(&id, getLatestEventIdV1Sql)
	if err != nil {
		return 0, err
	}
	return id, nil
}
