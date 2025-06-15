package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/applib/database/events"
)

type User struct {
	ID           int    `db:"id" json:"id"`
	Username     string `db:"username" json:"username"`
	Salt         string `db:"salt" json:"-"`
	PasswordHash string `db:"password_hash" json:"-"`
}

const UserAddedEventType string = "AddUser"
const UpdateUserPasswordEventType string = "UpdateUserPassword"
const DeleteUserEventType string = "DeleteUser"
const UpdateUserEventType string = "UpdateUser"

type UserAddedEvent struct {
	events.GenericEvent
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
}

type UpdateUserPasswordEvent struct {
	events.GenericEvent
	UserID      int    `json:"user_id"`
	NewPassword string `json:"new_password"`
}

type DeleteUserEvent struct {
	events.GenericEvent
	UserID int `json:"user_id"`
}

type UpdateUserEvent struct {
	events.GenericEvent
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
}

// -- DB Helpers --

func GetUser(db *sqlx.DB, username string) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT id, username, salt, password_hash FROM users_v1 WHERE username = $1", username)
	return &user, err
}

// -- Event handlers --

func UsersHandleInitEvent(tx *sqlx.Tx, event *events.DBInitEvent) (bool, error) {
	// Generate a random salt for the admin user
	salt := uuid.New().String()
	hasher := sha256.New()
	hasher.Write([]byte(salt + "admin"))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	// Create users table
	_, err := tx.Exec(`
		CREATE TABLE users_v1 (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			salt TEXT NOT NULL,
			password_hash TEXT NOT NULL
		)`)
	if err != nil {
		return false, fmt.Errorf("failed to create users table: %w", err)
	}

	// Create indexes for better performance
	_, err = tx.Exec(`CREATE INDEX idx_users_username ON users_v1(username)`)
	if err != nil {
		return false, fmt.Errorf("failed to create users username index: %w", err)
	}

	// Create admin user
	_, err = tx.Exec(`
		INSERT INTO users_v1 (username, salt, password_hash)
		SELECT 'admin', $1, $2
		`, salt, passwordHash)
	if err != nil {
		return false, fmt.Errorf("failed to create admin user: %w", err)
	}

	fmt.Println("User tables initialized.")
	return true, nil
}

func UsersHandleAddedEvent(tx *sqlx.Tx, event *UserAddedEvent) (bool, error) {
	fmt.Printf("Adding user: %s\n", event.Username)
	// Create random salt
	salt := uuid.New().String()

	var passwordHash string
	if event.Password != "" {
		// Hash the provided password
		hasher := sha256.New()
		hasher.Write([]byte(salt + event.Password))
		passwordHash = hex.EncodeToString(hasher.Sum(nil))
	}

	_, err := tx.Exec(`INSERT INTO users_v1 (username, salt, password_hash) VALUES ($1, $2, $3)`,
		event.Username, salt, passwordHash)
	if err != nil {
		// Consider UNIQUE constraint violation etc.
		return false, fmt.Errorf("failed to insert user %s: %w", event.Username, err)
	}
	return true, nil
}

func UsersHandleUpdatePasswordEvent(tx *sqlx.Tx, event *UpdateUserPasswordEvent) (bool, error) {
	fmt.Printf("Updating password for user ID: %d\n", event.UserID)

	// Generate new salt and hash password
	salt := uuid.New().String()
	hasher := sha256.New()
	hasher.Write([]byte(salt + event.NewPassword))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	result, err := tx.Exec(`UPDATE users_v1 SET salt = $1, password_hash = $2 WHERE id = $3`,
		salt, passwordHash, event.UserID)
	if err != nil {
		return false, fmt.Errorf("failed to update password for user %d: %w", event.UserID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return false, fmt.Errorf("no user found with ID %d", event.UserID)
	}

	return true, nil
}

func UsersHandleDeleteEvent(tx *sqlx.Tx, event *DeleteUserEvent) (bool, error) {
	fmt.Printf("Deleting user ID: %d\n", event.UserID)

	// Prevent deletion of admin user (ID = 1)
	if event.UserID == 1 {
		return false, fmt.Errorf("cannot delete admin user")
	}

	// Delete user access rules first (cascade delete)
	_, err := tx.Exec(`DELETE FROM user_access_rules_v1 WHERE subject_type = 'USER' AND subject_id = $1`,
		fmt.Sprintf("%d", event.UserID))
	if err != nil {
		return false, fmt.Errorf("failed to delete user access rules for user %d: %w", event.UserID, err)
	}

	// Delete the user
	result, err := tx.Exec(`DELETE FROM users_v1 WHERE id = $1`, event.UserID)
	if err != nil {
		return false, fmt.Errorf("failed to delete user %d: %w", event.UserID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return false, fmt.Errorf("no user found with ID %d", event.UserID)
	}

	return true, nil
}

func UsersHandleUpdateEvent(tx *sqlx.Tx, event *UpdateUserEvent) (bool, error) {
	fmt.Printf("Updating user ID: %d with username: %s\n", event.UserID, event.Username)

	// Prevent updating admin user ID (ID = 1) to different username
	if event.UserID == 1 && event.Username != "admin" {
		return false, fmt.Errorf("cannot change username of admin user")
	}

	result, err := tx.Exec(`UPDATE users_v1 SET username = $1 WHERE id = $2`,
		event.Username, event.UserID)
	if err != nil {
		return false, fmt.Errorf("failed to update user %d: %w", event.UserID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return false, fmt.Errorf("no user found with ID %d", event.UserID)
	}

	return true, nil
}

// -- Getters --

func GetUsers(db *sqlx.DB) ([]User, error) {
	ret := []User{}
	err := db.Select(&ret, "SELECT id, username FROM users_v1")
	if err != nil {
		return ret, fmt.Errorf("failed to select all users: %v", err)
	}

	return ret, nil
}
