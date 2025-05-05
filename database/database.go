package database

// Database is a service that manages the SQL connection and dispatches events
// to the appropriate handlers.

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/tomyedwab/yesterday/database/events"
)

// TODO(tom) Mechanism for backfilling new tables

const globalSchema = `
CREATE TABLE IF NOT EXISTS _versions (
	type TEXT PRIMARY KEY,
	version INTEGER,
	updated TIMESTAMP
)
`

const updateVersionSql = `
INSERT INTO _versions (type, version, updated)
VALUES ($1, $2, datetime())
ON CONFLICT (type)
DO UPDATE SET version = $2, updated = datetime();
`

type EventUpdateHandler func(tx *sqlx.Tx, event events.Event) (bool, error)

type Database struct {
	db             *sqlx.DB
	latestVersions map[string]int
	handlers       map[string]EventUpdateHandler
	version        string
	PublishEventCB func(event events.Event) error
}

// Connect creates a new database connection and initializes the database
// schema.
func Connect(driverName string, dataSourceName string, version string, handlers map[string]EventUpdateHandler) (*Database, error) {
	db, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	tx, err := db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(globalSchema)
	if err != nil {
		return nil, err
	}

	err = EventDBInit(tx)
	if err != nil {
		return nil, err
	}

	initEvent := &events.DBInitEvent{}
	for _, handler := range handlers {
		_, err := handler(tx, initEvent)
		if err != nil {
			return nil, err
		}
	}

	var versions []struct {
		Type    string `db:"type"`
		Version int    `db:"version"`
	}
	err = tx.Select(&versions, "SELECT type, version FROM _versions")
	if err != nil {
		return nil, err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	versionMap := make(map[string]int)
	for _, v := range versions {
		versionMap[v.Type] = v.Version
		fmt.Printf("Latest version of %s: %d\n", v.Type, v.Version)
	}

	fmt.Printf("Initialized application version: %s\n", version)

	return &Database{
		db:             db,
		latestVersions: versionMap,
		handlers:       handlers,
		version:        version,
	}, nil
}

// CreateEvent creates a new event in the database and updates the state of
// all handlers that are interested in this event.
func (db *Database) CreateEvent(event events.Event, eventData []byte, clientId string) (int, error) {
	// Clone the latest versions map
	versionsMap := make(map[string]int, len(db.latestVersions))
	for k, v := range db.latestVersions {
		versionsMap[k] = v
	}

	// Start a transaction before writing anything to the DB
	tx, err := db.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Create a new event, with an event ID
	eventId, err := EventDBCreateEvent(tx, eventData, clientId)
	if err != nil {
		return 0, err
	}
	event.SetId(eventId)

	// Update all handlers with the new event
	for handlerType, handler := range db.handlers {
		updated, err := handler(tx, event)
		if err != nil {
			return 0, err
		}
		if updated {
			// If the state changed as a result of this event, update the
			// version map for this handler name to the latest event ID
			versionsMap[handlerType] = eventId
			_, err = tx.Exec(updateVersionSql, handlerType, eventId)
			if err != nil {
				return 0, err
			}
			fmt.Printf("Updated %s version to %d\n", handlerType, eventId)
		}
	}

	// Commit the transaction
	err = tx.Commit()
	db.latestVersions = versionsMap
	return eventId, err
}

func (db *Database) GetDB() *sqlx.DB {
	return db.db
}
