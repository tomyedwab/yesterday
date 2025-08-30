package events

import (
	"database/sql"
	"fmt"
	"log"

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
	event_type TEXT NOT NULL,
	event_data JSONB NOT NULL
);
`

const getEventByClientIdV1Sql = `
SELECT id FROM event_v1 WHERE client_id = $1;
`

const insertEventV1Sql = `
INSERT INTO event_v1 (event_data, client_id, event_type)
VALUES ($1, $2, $3)
RETURNING id;
`

const getLatestEventIdV1Sql = `
SELECT event_type, MAX(id) FROM event_v1 GROUP BY event_type;
`

const getEventByIdV1Sql = `
SELECT event_data, event_type FROM event_v1 WHERE id = $1;
`

// EventDBInit initializes the event database schema. All events are stored as
// JSON blobs in the event_v1 table.
func EventDBInit(db *sqlx.DB) error {
	_, err := db.Exec(eventSchema)
	return err
}

// EventDB inserts a new event into the events table.
func EventDBCreateEvent(db *sqlx.DB, eventData []byte, clientId, eventType string) (int, error) {
	eventId := 0

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// First, a quick check if an event with the given client ID already exists
	// in the database. If so, this is a duplicate event, so just return the
	// existing ID.
	err = tx.QueryRow(getEventByClientIdV1Sql, clientId).Scan(&eventId)
	if err == nil {
		return 0, &DuplicateEventError{
			Id:       int(eventId),
			ClientId: clientId,
		}
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	err = tx.QueryRow(insertEventV1Sql, eventData, clientId, eventType).Scan(&eventId)
	if err != nil {
		return 0, err
	}
	log.Printf("Created new event with ID %d\n", eventId)
	tx.Commit()
	return int(eventId), nil
}

func EventDBGetCurrentEventIDs(db *sqlx.DB) (map[string]int, error) {
	ret := make(map[string]int)
	rows, err := db.Query(getLatestEventIdV1Sql)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var eventType string
		var id int
		err = rows.Scan(&eventType, &id)
		if err != nil {
			return nil, err
		}
		ret[eventType] = id
	}
	return ret, nil
}

func EventDBGetEvent(db *sqlx.DB, eventId int) (string, []byte, error) {
	var eventData []byte
	var eventType string

	err := db.QueryRow(getEventByIdV1Sql, eventId).Scan(&eventData, &eventType)
	if err != nil {
		return "", nil, err
	}
	return eventType, eventData, nil
}
