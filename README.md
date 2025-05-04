# yesterday
A simple event-based SQL database framework for self-hosted applications

## Core Concept

`yesterday` is a Go framework built around an **event-sourcing** pattern using a SQL database (specifically SQLite is assumed by default, but pluggable via `sqlx`). Instead of directly modifying application state in the database, applications publish events. These events are stored chronologically in an immutable `event_v1` table. Application-specific "handlers" then process these events transactionally to update materialized views or specific state tables.

## Key Features

*   **Immutable Event Log:** Events (as JSON blobs) are appended to an `event_v1` table with a unique, auto-incrementing ID and a client-provided ID for deduplication.
*   **Transactional State Updates:** Application-specific "handlers" (`EventUpdateHandler`) are registered during initialization. When a new event is published, it's first logged, and then *transactionally*, each handler is called to update its own state tables based on the event.
*   **Versioning:** The framework tracks the last processed event ID for each handler type in a `_versions` table.
*   **Built-in HTTP API:** Provides endpoints for event publishing (`/api/publish`) and long-polling for state updates (`/api/poll`). Includes basic middleware for logging and CORS.
*   **Pluggable Event Types:** Define your own event structures and provide a mapping function (`MapEventType`) to parse incoming JSON events.
*   **Database Agnostic (via sqlx):** While examples use SQLite, any database supported by `sqlx` can be used.

## Basic Usage Example

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/database/events"
)

// Define our custom event type
type MyEvent struct {
	events.GenericEvent // Embed generic fields like Id, Type, Timestamp
	Payload             string `json:"payload"`
}

// Implement the events.Event interface (partially covered by embedding GenericEvent)
// We only need to ensure our struct can be used where events.Event is needed.

// Define our state handler
func myEventHandler(tx *sqlx.Tx, event events.Event) (bool, error) {
	fmt.Printf("Handler received event ID: %d, Type: %s\n", event.GetId(), event.GetType())

	// On initial DB setup, create our state table
	if _, ok := event.(*events.DBInitEvent); ok {
		_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS my_state (last_payload TEXT)`)
		if err != nil {
			return false, fmt.Errorf("failed to create my_state table: %w", err)
		}
		// Insert initial empty state if needed
		_, err = tx.Exec(`INSERT INTO my_state (last_payload) SELECT '' WHERE NOT EXISTS (SELECT 1 FROM my_state)`)
		return true, err // Indicate state was potentially modified (table created)
	}

	// Process our specific event type
	if myEvent, ok := event.(*MyEvent); ok {
		if myEvent.GetType() == "MY_EVENT" {
			fmt.Printf("Processing MY_EVENT with payload: %s\n", myEvent.Payload)
			// Update our application state table
			_, err := tx.Exec(`UPDATE my_state SET last_payload = $1`, myEvent.Payload)
			if err != nil {
				return false, fmt.Errorf("failed to update my_state: %w", err)
			}
			return true, nil // Indicate state was modified
		}
	}

	// Event was not relevant to this handler, state not modified
	return false, nil
}

// Map incoming JSON to our Go event types
func mapEventType(rawMessage *json.RawMessage, generic *events.GenericEvent) (events.Event, error) {
	switch generic.GetType() {
	case "MY_EVENT":
		var specificEvent MyEvent
		if err := json.Unmarshal(*rawMessage, &specificEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MyEvent: %w", err)
		}
		// Copy generic fields that aren't automatically unmarshalled if needed (like Id)
		specificEvent.GenericEvent = *generic
		return &specificEvent, nil
	default:
		// Return the generic event if the type is unknown or unhandled here
		fmt.Printf("Unknown event type: %s\n", generic.GetType())
		return generic, nil
	}
}

func main() {
	dbPath := "./my_app.db"
	os.Remove(dbPath) // Start fresh for example

	// Define handlers
	handlers := map[string]database.EventUpdateHandler{
		"myStateBuilder": myEventHandler,
		// Add more handlers for different state representations if needed
	}

	// Connect to the database and initialize schema/handlers
	db, err := database.Connect("sqlite3", dbPath, "v1.0.0", handlers)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize HTTP endpoints
	db.InitHandlers(mapEventType)

	fmt.Println("Database initialized. Starting server on :8080")
	fmt.Println("Try: curl -X POST -H 'Content-Type: application/json' -d '{\"type\": \"MY_EVENT\", \"payload\": \"hello world\"}' 'http://localhost:8080/api/publish?cid=client123'")
	fmt.Println("Then: curl 'http://localhost:8080/api/poll?e=1'") // Poll for event 1

	log.Fatal(http.ListenAndServe(":8080", nil))
}
