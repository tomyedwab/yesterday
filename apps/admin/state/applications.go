package state

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/applib/events"
)

// Application represents an application in the system.
type Application struct {
	InstanceID  string `db:"instance_id" json:"instanceId"`
	AppID       string `db:"app_id" json:"appId"`
	DisplayName string `db:"display_name" json:"displayName"`
	HostName    string `db:"host_name" json:"hostName"`
}

// Event types for application management
const CreateApplicationEventType string = "CreateApplication"
const UpdateApplicationEventType string = "UpdateApplication"
const DeleteApplicationEventType string = "DeleteApplication"

// Event structures for application management
type CreateApplicationEvent struct {
	events.GenericEvent
	AppID       string `json:"appId"`
	DisplayName string `json:"displayName"`
	HostName    string `json:"hostName"`
}

type UpdateApplicationEvent struct {
	events.GenericEvent
	InstanceID  string `json:"instanceId"`
	AppID       string `json:"appId"`
	DisplayName string `json:"displayName"`
	HostName    string `json:"hostName"`
}

type DeleteApplicationEvent struct {
	events.GenericEvent
	InstanceID string `json:"instanceId"`
}

// -- Event handlers --

func ApplicationsHandleInitEvent(tx *sqlx.Tx, event *events.DBInitEvent) (bool, error) {
	// Create applications table
	_, err := tx.Exec(`
		CREATE TABLE applications_v1 (
			instance_id TEXT PRIMARY KEY,
			app_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			host_name TEXT NOT NULL
		)`)
	if err != nil {
		return false, fmt.Errorf("failed to create applications table: %w", err)
	}

	// Create indexes for better performance
	_, err = tx.Exec(`CREATE INDEX idx_applications_app_id ON applications_v1(app_id)`)
	if err != nil {
		return false, fmt.Errorf("failed to create applications app_id index: %w", err)
	}

	_, err = tx.Exec(`CREATE INDEX idx_applications_display_name ON applications_v1(display_name)`)
	if err != nil {
		return false, fmt.Errorf("failed to create applications display_name index: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO applications_v1 (instance_id, app_id, display_name, host_name)
		SELECT '3bf3e3c0-6e51-482a-b180-00f6aa568ee9', 'github.com/tomyedwab/yesterday/apps/login', 'Login service', 'login'
	`)
	if err != nil {
		return false, fmt.Errorf("failed to create login application: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO applications_v1 (instance_id, app_id, display_name, host_name)
		SELECT 'MBtskI6D', 'github.com/tomyedwab/yesterday/apps/admin', 'Admin service', 'admin'
	`)
	if err != nil {
		return false, fmt.Errorf("failed to create admin application: %w", err)
	}

	fmt.Println("Application tables initialized (if not exists).")
	return true, nil
}

func ApplicationsHandleCreateEvent(tx *sqlx.Tx, event *CreateApplicationEvent) (bool, error) {
	fmt.Printf("Creating application: %s\n", event.DisplayName)

	// Generate unique instance ID
	instanceID := uuid.New().String()

	_, err := tx.Exec(`
		INSERT INTO applications_v1 (instance_id, app_id, display_name, host_name)
		VALUES ($1, $2, $3, $4)`,
		instanceID, event.AppID, event.DisplayName, event.HostName)

	if err != nil {
		return false, fmt.Errorf("failed to create application %s: %w", event.DisplayName, err)
	}

	return true, nil
}

func ApplicationsHandleUpdateEvent(tx *sqlx.Tx, event *UpdateApplicationEvent) (bool, error) {
	fmt.Printf("Updating application: %s\n", event.InstanceID)

	result, err := tx.Exec(`
		UPDATE applications_v1
		SET app_id = $1, display_name = $2, host_name = $3
		WHERE instance_id = $4`,
		event.AppID, event.DisplayName, event.HostName, event.InstanceID)

	if err != nil {
		return false, fmt.Errorf("failed to update application %s: %w", event.InstanceID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return false, fmt.Errorf("no application found with instance ID %s", event.InstanceID)
	}

	return true, nil
}

func ApplicationsHandleDeleteEvent(tx *sqlx.Tx, event *DeleteApplicationEvent) (bool, error) {
	fmt.Printf("Deleting application: %s\n", event.InstanceID)

	// Prevent deletion of core system applications
	if event.InstanceID == "MBtskI6D" {
		return false, fmt.Errorf("cannot delete core system applications")
	}

	// Delete associated access rules first
	_, err := tx.Exec(`DELETE FROM user_access_rules_v1 WHERE application_id = $1`, event.InstanceID)
	if err != nil {
		return false, fmt.Errorf("failed to delete access rules for application %s: %w", event.InstanceID, err)
	}

	// Delete the application
	result, err := tx.Exec(`DELETE FROM applications_v1 WHERE instance_id = $1`, event.InstanceID)
	if err != nil {
		return false, fmt.Errorf("failed to delete application %s: %w", event.InstanceID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return false, fmt.Errorf("no application found with instance ID %s", event.InstanceID)
	}

	return true, nil
}

// -- DB Helpers --

// GetApplication retrieves a specific application by its ID.
func GetApplication(db *sqlx.DB, instanceId string) (*Application, error) {
	var app Application
	err := db.Get(&app, "SELECT instance_id, app_id, display_name, host_name FROM applications_v1 WHERE instance_id = $1", instanceId)
	return &app, err
}

// GetApplications retrieves all applications sorted by display name.
func GetApplications(db *sqlx.DB) ([]Application, error) {
	ret := []Application{}
	err := db.Select(&ret, "SELECT instance_id, app_id, display_name, host_name FROM applications_v1 ORDER BY instance_id")
	if err != nil {
		return ret, fmt.Errorf("failed to select all applications: %v", err)
	}
	return ret, nil
}
