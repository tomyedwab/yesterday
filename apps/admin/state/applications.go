package state

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/wasi/guest"
)

// Application represents an application in the system.
type Application struct {
	InstanceID  string `db:"instance_id" json:"instanceId"`
	AppID       string `db:"app_id" json:"appId"`
	DisplayName string `db:"display_name" json:"displayName"`
	HostName    string `db:"host_name" json:"hostName"`
	DBName      string `db:"db_name" json:"dbName"`
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
	DBName      string `json:"dbName"`
}

type UpdateApplicationEvent struct {
	events.GenericEvent
	InstanceID  string `json:"instanceId"`
	AppID       string `json:"appId"`
	DisplayName string `json:"displayName"`
	HostName    string `json:"hostName"`
	DBName      string `json:"dbName"`
}

type DeleteApplicationEvent struct {
	events.GenericEvent
	InstanceID string `json:"instanceId"`
}

// -- Event handlers --

func ApplicationsHandleInitEvent(tx *guest.Tx, event *events.DBInitEvent) (bool, error) {
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
		INSERT INTO applications_v1 (instance_id, app_id, display_name, host_name, db_name)
		SELECT '3bf3e3c0-6e51-482a-b180-00f6aa568ee9', '0001-0001', 'Login service', 'login.yesterday.localhost:8443', 'db/sessions.db'
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

func ApplicationsHandleCreateEvent(tx *guest.Tx, event *CreateApplicationEvent) (bool, error) {
	guest.WriteLog(fmt.Sprintf("Creating application: %s", event.DisplayName))

	// Generate unique instance ID
	instanceID := uuid.New().String()

	_, err := tx.Exec(`
		INSERT INTO applications_v1 (instance_id, app_id, display_name, host_name, db_name)
		VALUES ($1, $2, $3, $4, $5)`,
		instanceID, event.AppID, event.DisplayName, event.HostName, event.DBName)

	if err != nil {
		return false, fmt.Errorf("failed to create application %s: %w", event.DisplayName, err)
	}

	return true, nil
}

func ApplicationsHandleUpdateEvent(tx *guest.Tx, event *UpdateApplicationEvent) (bool, error) {
	guest.WriteLog(fmt.Sprintf("Updating application: %s", event.InstanceID))

	result, err := tx.Exec(`
		UPDATE applications_v1
		SET app_id = $1, display_name = $2, host_name = $3, db_name = $4
		WHERE instance_id = $5`,
		event.AppID, event.DisplayName, event.HostName, event.DBName, event.InstanceID)

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

func ApplicationsHandleDeleteEvent(tx *guest.Tx, event *DeleteApplicationEvent) (bool, error) {
	guest.WriteLog(fmt.Sprintf("Deleting application: %s", event.InstanceID))

	// Prevent deletion of core system applications
	if event.InstanceID == "3bf3e3c0-6e51-482a-b180-00f6aa568ee9" ||
		event.InstanceID == "18736e4f-93f9-4606-a7be-863c7986ea5b" {
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
func GetApplication(db *guest.DB, instanceId string) (*Application, error) {
	var app Application
	err := db.Get(&app, "SELECT instance_id, app_id, display_name, host_name, db_name FROM applications_v1 WHERE instance_id = $1", instanceId)
	return &app, err
}

// GetApplications retrieves all applications sorted by display name.
func GetApplications(db *guest.DB) ([]Application, error) {
	ret := []Application{}
	err := db.Select(&ret, "SELECT instance_id, app_id, display_name, host_name, db_name FROM applications_v1 ORDER BY display_name")
	if err != nil {
		return ret, fmt.Errorf("failed to select all applications: %v", err)
	}
	return ret, nil
}
