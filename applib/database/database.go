package database

// Database is a service that manages the SQL connection and dispatches events
// to the appropriate handlers.

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/tomyedwab/yesterday/applib/events"
)

type EventHandler[T events.Event] func(tx *sqlx.Tx, event T) (bool, error)
type GenericEventHandler func(tx *sqlx.Tx, eventJson []byte) (bool, error)

type Database struct {
	db             *sqlx.DB
	handlers       map[string][]GenericEventHandler
	version        string
	PublishEventCB func(event events.Event) error
}

func Connect(driverName string, dataSourceName, version string) (*Database, error) {
	db, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Database{
		db:       db,
		handlers: make(map[string][]GenericEventHandler),
		version:  version,
	}, nil
}

func AddEventHandler[T events.Event](db *Database, eventType string, handler EventHandler[T]) {
	db.handlers[eventType] = append(db.handlers[eventType], func(tx *sqlx.Tx, eventJson []byte) (bool, error) {
		var event T
		if err := json.Unmarshal(eventJson, &event); err != nil {
			return false, fmt.Errorf("failed to unmarshal event of type %s: %w", eventType, err)
		}
		return handler(tx, event)
	})
}

func AddGenericEventHandler(db *Database, eventType string, handler GenericEventHandler) {
	db.handlers[eventType] = append(db.handlers[eventType], handler)
}

// Connect creates a new database connection and initializes the database
// schema.
func (db *Database) Initialize() error {
	err := db.InitHandlers()
	if err != nil {
		return err
	}

	fmt.Printf("Initialized application version: %s\n", db.version)

	return nil
}

// CreateEvent creates a new event in the database and updates the state of
// all handlers that are interested in this event.
/* TODO STOPSHIP
func (db *Database) CreateEvent(event *events.GenericEvent, eventData []byte, clientId string) (int, error) {
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

	// Update all handlers with the new event
	for _, handlerFunc := range db.handlers[event.GetType()] {
		_, err := handlerFunc(tx, eventData)
		if err != nil {
			return 0, err
		}
	}

	// Commit the transaction
	err = tx.Commit()
	return eventId, err
}
*/

func (db *Database) GetDB() *sqlx.DB {
	return db.db
}
