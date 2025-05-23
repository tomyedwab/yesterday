package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/database/middleware"
)

type User struct {
	ID           int    `db:"id"`
	Username     string `db:"username"`
	Salt         string `db:"salt"`
	PasswordHash string `db:"password_hash"`
}

// --- Event Types ---

type UserAddedEvent struct {
	events.GenericEvent
	Username string `json:"username"`
}

type UserChangedPasswordEvent struct {
	events.GenericEvent
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

// -- DB Helpers --

func GetUser(db *sqlx.DB, username string) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT id, username, salt, password_hash FROM users_v1 WHERE username = $1", username)
	return &user, err
}

// --- State Handler  ---

// UserStateHandler processes events related to user management.
func UserStateHandler(tx *sqlx.Tx, event events.Event) (bool, error) {
	fmt.Printf("User Handler received event ID: %d, Type: %s\n", event.GetId(), event.GetType())

	switch evt := event.(type) {
	case *events.DBInitEvent:
		// Generate a random salt for the admin user
		salt := uuid.New().String()
		hasher := sha256.New()
		hasher.Write([]byte(salt + "admin"))
		passwordHash := hex.EncodeToString(hasher.Sum(nil))

		// Create users table
		_, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS users_v1 (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT UNIQUE NOT NULL,
				salt TEXT NOT NULL,
				password_hash TEXT NOT NULL
			)`)
		if err != nil {
			return false, fmt.Errorf("failed to create users table: %w", err)
		}

		// Create admin user if it doesn't exist
		_, err = tx.Exec(`
			INSERT INTO users_v1 (username, salt, password_hash)
			SELECT 'admin', $1, $2
			WHERE NOT EXISTS (
				SELECT 1 FROM users_v1 WHERE username = 'admin'
			)`, salt, passwordHash)
		if err != nil {
			return false, fmt.Errorf("failed to create admin user: %w", err)
		}
		fmt.Println("User tables initialized (if not exists).")
		return true, nil

	case *UserAddedEvent:
		fmt.Printf("Adding user: %s\n", evt.Username)
		// Create random salt
		salt := uuid.New().String()
		_, err := tx.Exec(`INSERT INTO users_v1 (username, salt, password_hash) VALUES ($1, $2, $3)`,
			evt.Username, salt, "")
		if err != nil {
			// Consider UNIQUE constraint violation etc.
			return false, fmt.Errorf("failed to insert user %s: %w", evt.Username, err)
		}
		return true, nil

	case *UserChangedPasswordEvent:
		fmt.Printf("Changing password for user: %s\n", evt.Username)
		_, err := tx.Exec(`UPDATE users_v1 SET password_hash = $1 WHERE username = $2`, evt.PasswordHash, evt.Username)
		if err != nil {
			return false, fmt.Errorf("failed to update user %s: %w", evt.Username, err)
		}
		return true, nil
	}

	// Event not relevant to this handler
	return false, nil
}

// --- Login ---

func AttemptLogin(db *database.Database, username string, password string) (bool, int, error) {
	fmt.Printf("Attempting login for user: %s\n", username)
	user, err := GetUser(db.GetDB(), username)
	if err != nil {
		fmt.Printf("Failed to get user %s: %v\n", username, err)
		return false, 0, fmt.Errorf("failed to get user %s: %w", username, err)
	}

	hasher := sha256.New()
	hasher.Write([]byte(user.Salt + password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	return user.PasswordHash == passwordHash, user.ID, nil
}

func ChangePassword(db *database.Database, username string, password string) (int, error) {
	user, err := GetUser(db.GetDB(), username)
	if err != nil {
		return 0, fmt.Errorf("failed to get user %s: %w", username, err)
	}

	hasher := sha256.New()
	hasher.Write([]byte(user.Salt + password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	return user.ID, db.PublishEventCB(&UserChangedPasswordEvent{
		GenericEvent: events.GenericEvent{
			Id:   0,
			Type: "users:CHANGE_PASSWORD",
		},
		Username:     username,
		PasswordHash: passwordHash,
	})
}

// --- Getters ---

func GetAllUsers(db *sqlx.DB) ([]struct {
	ID       int
	Username string
}, error) {
	var users []struct {
		ID       int
		Username string
	}
	err := db.Select(&users, "SELECT id, username FROM users_v1")
	return users, err
}

func InitUserHandlers(db *database.Database) {
	http.HandleFunc("/api/listusers", middleware.ApplyDefault(func(w http.ResponseWriter, r *http.Request) {
		resp, err := GetAllUsers(db.GetDB())
		database.HandleAPIResponse(w, r, resp, err)
	}))
}
