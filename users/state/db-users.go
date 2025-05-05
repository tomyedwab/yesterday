package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/database/events"
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

func ChangePassword(db *database.Database, username string, password string) error {
	user, err := GetUser(db.GetDB(), username)
	if err != nil {
		return fmt.Errorf("failed to get user %s: %w", username, err)
	}

	hasher := sha256.New()
	hasher.Write([]byte(user.Salt + password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	return db.PublishEventCB(&UserChangedPasswordEvent{
		GenericEvent: events.GenericEvent{
			Id:   0,
			Type: "users:CHANGE_PASSWORD",
		},
		Username:     username,
		PasswordHash: passwordHash,
	})
}
