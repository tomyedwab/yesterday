package state

import (
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/database/middleware"
)

// Application represents an application in the system.
type Application struct {
	ID          string `db:"id"`
	DisplayName string `db:"display_name"`
	HostName    string `db:"host_name"`
}

// --- Event Types ---

// ApplicationAddedEvent is triggered when a new application is added.
type ApplicationAddedEvent struct {
	events.GenericEvent
	ApplicationID string `json:"application_id"`
	DisplayName   string `json:"display_name"`
	HostName      string `json:"host_name"`
}

// ApplicationDeletedEvent is triggered when an application is deleted.
type ApplicationDeletedEvent struct {
	events.GenericEvent
	ApplicationID string `json:"application_id"`
}

// ApplicationHostNameUpdatedEvent is triggered when an application's host name is updated.
type ApplicationHostNameUpdatedEvent struct {
	events.GenericEvent
	ApplicationID string `json:"application_id"`
	HostName      string `json:"host_name"`
}

// -- DB Helpers --

// GetApplication retrieves a specific application by its ID.
func GetApplication(db *sqlx.DB, id string) (*Application, error) {
	var app Application
	err := db.Get(&app, "SELECT id, display_name, host_name FROM applications_v1 WHERE id = $1", id)
	return &app, err
}

// --- State Handler ---

// ApplicationStateHandler processes events related to application management.
func ApplicationStateHandler(tx *sqlx.Tx, event events.Event) (bool, error) {
	fmt.Printf("Application Handler received event ID: %d, Type: %s\n", event.GetId(), event.GetType())

	switch evt := event.(type) {
	case *events.DBInitEvent:
		// Create applications table
		_, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS applications_v1 (
				id TEXT PRIMARY KEY,
				display_name TEXT NOT NULL,
				host_name TEXT NOT NULL
			)`)
		if err != nil {
			return false, fmt.Errorf("failed to create applications table: %w", err)
		}

		_, err = tx.Exec(`
			INSERT INTO applications_v1 (id, display_name, host_name)
			SELECT '0001-0001', 'User Management', ''
			WHERE NOT EXISTS (
				SELECT 1 FROM applications_v1 WHERE id = '0001-0001'
			)
		`)
		if err != nil {
			return false, fmt.Errorf("failed to create admin application: %w", err)
		}
		fmt.Println("Application tables initialized (if not exists).")

	case *ApplicationAddedEvent:
		fmt.Printf("Adding application: %s (ID: %s)\n", evt.DisplayName, evt.ApplicationID)
		_, err := tx.Exec(`INSERT INTO applications_v1 (id, display_name, host_name) VALUES ($1, $2, $3)`,
			evt.ApplicationID, evt.DisplayName, evt.HostName)
		if err != nil {
			// Consider UNIQUE constraint violation etc.
			return false, fmt.Errorf("failed to insert application %s: %w", evt.ApplicationID, err)
		}
		return true, nil

	case *ApplicationDeletedEvent:
		fmt.Printf("Deleting application: (ID: %s)\n", evt.ApplicationID)
		_, err := tx.Exec(`DELETE FROM applications_v1 WHERE id = $1`,
			evt.ApplicationID)
		if err != nil {
			return false, fmt.Errorf("failed to delete application %s: %w", evt.ApplicationID, err)
		}
		return true, nil

	case *ApplicationHostNameUpdatedEvent:
		fmt.Printf("Updating application host name: (ID: %s, New HostName: %s)\n", evt.ApplicationID, evt.HostName)
		_, err := tx.Exec(`UPDATE applications_v1 SET host_name = $1 WHERE id = $2`,
			evt.HostName, evt.ApplicationID)
		if err != nil {
			return false, fmt.Errorf("failed to update host name for application %s: %w", evt.ApplicationID, err)
		}
		return true, nil
	}

	// Event not relevant to this handler
	return false, nil
}

// --- Getters ---

// GetAllApplications retrieves all applications from the database.
func GetAllApplications(db *sqlx.DB) ([]Application, error) {
	var apps []Application
	err := db.Select(&apps, "SELECT id, display_name, host_name FROM applications_v1")
	return apps, err
}

func InitApplicationHandlers(db *database.Database) {
	http.HandleFunc("/api/listapplications", middleware.ApplyDefault(func(w http.ResponseWriter, r *http.Request) {
		resp, err := GetAllApplications(db.GetDB())
		database.HandleAPIResponse(w, r, resp, err)
	}))
}
