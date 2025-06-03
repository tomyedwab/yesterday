package state

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database/events"
)

// -- Event handlers --

func UserProfilesHandleInitEvent(tx *sqlx.Tx, event *events.DBInitEvent) (bool, error) {
	// Create user_profiles table
	_, err := tx.Exec(`
		CREATE TABLE user_profiles_v1 (
			user_id INTEGER NOT NULL,
			application_id TEXT NOT NULL,
			profile_data TEXT NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users_v1(id) ON DELETE CASCADE,
			FOREIGN KEY(application_id) REFERENCES applications_v1(id) ON DELETE CASCADE,
			PRIMARY KEY (user_id, application_id)
		)`)
	if err != nil {
		return false, fmt.Errorf("failed to create user_profiles table: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO user_profiles_v1 (user_id, application_id, profile_data)
		SELECT 1, '18736e4f-93f9-4606-a7be-863c7986ea5b', '{}'
		`)
	if err != nil {
		return false, fmt.Errorf("failed to create admin user profile: %w", err)
	}

	fmt.Println("User profiles table initialized.")
	return true, nil
}

// --- Getter ---

// GetUserProfile retrieves the profile data for a given user and application.
func GetUserProfile(db *sqlx.DB, userID int, applicationID string) (*string, error) {
	var profileData string
	err := db.Get(&profileData, `
		SELECT profile_data
		FROM user_profiles_v1
		WHERE user_id = $1 AND application_id = $2
		`, userID, applicationID)
	if err != nil {
		// There is no need to error here, just return nil
		return nil, nil
	}
	return &profileData, nil
}
