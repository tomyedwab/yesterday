package state

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/database/middleware"
)

// --- Event Types ---

// UserProfileUpdatedEvent represents an event where a user's profile for a specific application is updated.
type UserProfileUpdatedEvent struct {
	events.GenericEvent
	UserID        int    `json:"user_id"`
	ApplicationID string `json:"application_id"`
	ProfileData   string `json:"profile_data"` // JSON blob
}

// --- State Handler ---

// UserProfileStateHandler processes events related to user profiles.
func UserProfileStateHandler(tx *sqlx.Tx, event events.Event) (bool, error) {
	switch evt := event.(type) {
	case *events.DBInitEvent:
		// Create user_profiles table
		_, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS user_profiles_v1 (
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
			SELECT 1, '0001-0001', '{}'
			WHERE NOT EXISTS (
				SELECT 1 FROM user_profiles_v1 WHERE user_id = 1 AND application_id = '0001-0001'
			)`)
		if err != nil {
			return false, fmt.Errorf("failed to create admin user profile: %w", err)
		}
		fmt.Println("User profiles table initialized (if not exists).")

	case *UserProfileUpdatedEvent:
		fmt.Printf("Updating profile for user ID %d, application %s\n", evt.UserID, evt.ApplicationID)
		_, err := tx.Exec(`
			INSERT INTO user_profiles_v1 (user_id, application_id, profile_data)
			VALUES ($1, $2, $3)
			ON CONFLICT(user_id, application_id) DO UPDATE SET
				profile_data = excluded.profile_data
			`, evt.UserID, evt.ApplicationID, evt.ProfileData)
		if err != nil {
			return false, fmt.Errorf("failed to upsert user profile for user ID %d, app %s: %w", evt.UserID, evt.ApplicationID, err)
		}
		return true, nil
	}

	// Event not relevant to this handler
	return false, nil
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

func InitUserProfileHandlers(db *database.Database) {
	http.HandleFunc("/api/getuserprofile", middleware.ApplyDefault(func(w http.ResponseWriter, r *http.Request) {
		userId, err := strconv.Atoi(r.URL.Query().Get("userId"))
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		applicationID := r.URL.Query().Get("applicationId")
		profile, err := GetUserProfile(db.GetDB(), userId, applicationID)
		if err != nil {
			http.Error(w, "Failed to get profile", http.StatusInternalServerError)
			return
		}
		if profile == nil {
			database.HandleAPIResponse(w, r, nil, nil)
			return
		}

		var profileParsed map[string]interface{}
		err = json.Unmarshal([]byte(*profile), &profileParsed)
		if err != nil {
			profileParsed = map[string]interface{}{
				"_value": profile,
			}
		}
		database.HandleAPIResponse(w, r, map[string]interface{}{"userId": userId, "applicationId": applicationID, "profile": profileParsed}, err)
	}))
}
