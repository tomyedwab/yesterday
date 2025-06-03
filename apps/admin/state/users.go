package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database/events"
)

type User struct {
	ID           int    `db:"id"`
	Username     string `db:"username"`
	Salt         string `db:"salt"`
	PasswordHash string `db:"password_hash"`
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
