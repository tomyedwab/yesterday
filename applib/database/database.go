package database

// Database is a service that manages the SQL connection and dispatches events
// to the appropriate handlers.

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type EventHandler[T interface{}] func(tx *sqlx.Tx, event T) (bool, error)
type GenericEventHandler func(tx *sqlx.Tx, eventJson []byte) (bool, error)

type Database struct {
	db         *sqlx.DB
	handlers   map[string][]GenericEventHandler
	eventState *EventState
}

func Connect(driverName string, dataSourceName string) (*Database, error) {
	db, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Database{
		db:       db,
		handlers: make(map[string][]GenericEventHandler),
	}, nil
}

func AddEventHandler[T interface{}](db *Database, eventType string, handler EventHandler[T]) {
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
	var err error
	db.eventState, err = NewEventState(db.GetDB())
	if err != nil {
		return err
	}

	err = db.InitHandlers()
	if err != nil {
		return err
	}

	fmt.Println("Initialized application.")

	return nil
}

// HandleEvent updates the state of all handlers that are interested in the
// event type and updates the current event ID.
func (db *Database) HandleEvent(eventId int, eventType string, eventData []byte) error {
	// Start a transaction before writing anything to the DB
	tx, err := db.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update all handlers with the new event
	for _, handlerFunc := range db.handlers[eventType] {
		_, err := handlerFunc(tx, eventData)
		if err != nil {
			return err
		}
	}

	err = db.eventState.SetCurrentEventId(eventId, tx)
	if err != nil {
		return err
	}

	// Commit the transaction
	err = tx.Commit()
	return err
}

func (db *Database) GetDB() *sqlx.DB {
	return db.db
}
