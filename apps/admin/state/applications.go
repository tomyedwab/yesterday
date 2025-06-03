package state

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database/events"
)

// Application represents an application in the system.
type Application struct {
	InstanceID  string `db:"instance_id"`
	AppID       string `db:"app_id"`
	DisplayName string `db:"display_name"`
	HostName    string `db:"host_name"`
	DBName      string `db:"db_name"`
}

// -- Event handlers --

func ApplicationsHandleInitEvent(tx *sqlx.Tx, event *events.DBInitEvent) (bool, error) {
	// Create applications table
	_, err := tx.Exec(`
		CREATE TABLE applications_v1 (
			instance_id TEXT PRIMARY KEY,
			app_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			host_name TEXT NOT NULL,
			db_name TEXT NOT NULL
		)`)
	if err != nil {
		return false, fmt.Errorf("failed to create applications table: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO applications_v1 (instance_id, app_id, display_name, host_name, db_name)
		SELECT '3bf3e3c0-6e51-482a-b180-00f6aa568ee9', '0001-0001', 'Login service', 'login.yesterday.localhost:8443', 'db/admin.db'
	`)
	if err != nil {
		return false, fmt.Errorf("failed to create login application: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO applications_v1 (instance_id, app_id, display_name, host_name, db_name)
		SELECT '18736e4f-93f9-4606-a7be-863c7986ea5b', '0001-0002', 'Admin service', 'admin.yesterday.localhost:8443', 'db/admin.db'
	`)
	if err != nil {
		return false, fmt.Errorf("failed to create admin application: %w", err)
	}

	fmt.Println("Application tables initialized (if not exists).")
	return true, nil
}

// -- DB Helpers --

// GetApplication retrieves a specific application by its ID.
func GetApplication(db *sqlx.DB, instanceId string) (*Application, error) {
	var app Application
	err := db.Get(&app, "SELECT instance_id, app_id, display_name, host_name, db_name FROM applications_v1 WHERE instance_id = $1", instanceId)
	return &app, err
}
