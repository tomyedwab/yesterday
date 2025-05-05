package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	// Consider using a dedicated UUID library in a real application
	// "github.com/google/uuid"
)

// Session represents an active user session for a specific application.
// Sessions are stored in the database and referenced by their ID.
type Session struct {
	ID            string    `json:"id" db:"id"` // Unique identifier for the session (e.g., UUID)
	UserID        int       `json:"user_id" db:"user_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	LastRefreshed time.Time `json:"last_refreshed" db:"last_refreshed"` // Timestamp of the last refresh token issuance
	RefreshToken  string    `json:"refresh_token" db:"refresh_token"`   // The refresh token
}

// NewSession creates a new session instance with a unique ID.
func NewSession(userID int) (*Session, error) {
	sessionID, err := generateRandomID(16) // Generate a 16-byte random ID
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRandomID(32)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Session{
		ID:            sessionID,
		UserID:        userID,
		CreatedAt:     now,
		LastRefreshed: now,
		RefreshToken:  refreshToken,
	}, nil
}

// generateRandomID generates a cryptographically secure random string encoded in base64.
func generateRandomID(length int) (string, error) {
	if length <= 0 {
		// Default length if invalid input is provided
		length = 16
	}
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err // Propagate the error
	}
	// Use URL-safe base64 encoding without padding
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// --- Database Methods ---

func DBInit(db *sqlx.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		refresh_token TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_refreshed TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users_v1(id) ON DELETE CASCADE
	)
	`)
	return err
}

func DBGetSessionByID(db *sqlx.DB, id string) (*Session, error) {
	var s Session
	err := db.Get(&s, "SELECT * FROM sessions WHERE id = $1", id)
	return &s, err
}

func DBGetSessionByRefreshToken(db *sqlx.DB, refreshToken string) (*Session, error) {
	var s Session
	err := db.Get(&s, "SELECT * FROM sessions WHERE refresh_token = $1", refreshToken)
	return &s, err
}

func (s *Session) DBCreate(db *sqlx.DB) error {
	fmt.Printf("Creating session %s\n", s.ID)
	_, err := db.Exec("INSERT INTO sessions (id, user_id, refresh_token, last_refreshed) VALUES ($1, $2, $3, $4)", s.ID, s.UserID, s.RefreshToken, s.LastRefreshed)
	return err
}

func (s *Session) DBUpdateRefreshToken(db *sqlx.DB) (string, error) {
	fmt.Printf("Updating refresh token for session %s\n", s.ID)
	refreshToken, err := generateRandomID(32)
	if err != nil {
		return "", err
	}
	s.RefreshToken = refreshToken
	s.LastRefreshed = time.Now().UTC()
	_, err = db.Exec("UPDATE sessions SET refresh_token = $1, last_refreshed = $2 WHERE id = $3", s.RefreshToken, s.LastRefreshed, s.ID)
	return refreshToken, err
}

func (s *Session) DBDelete(db *sqlx.DB) error {
	fmt.Printf("Deleting session %s\n", s.ID)
	_, err := db.Exec("DELETE FROM sessions WHERE id = $1", s.ID)
	return err
}

func DBDeleteExpiredSessions(db *sqlx.DB, sessionExpiry time.Duration) error {
	// Get session IDs that have expired
	var sessionIDs []string
	err := db.Select(&sessionIDs, "SELECT id FROM sessions WHERE last_refreshed < $1", time.Now().UTC().Add(-sessionExpiry))
	if err != nil {
		return err
	}

	for _, sessionID := range sessionIDs {
		fmt.Printf("Automatically deleting expired session %s\n", sessionID)
		_, err = db.Exec("DELETE FROM sessions WHERE id = $1", sessionID)
		if err != nil {
			return err
		}
	}
	return nil
}
