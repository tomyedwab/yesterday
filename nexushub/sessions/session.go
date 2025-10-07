package sessions

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FlexibleInt64 can handle both int64 and float64 from database
type FlexibleInt64 int64

func (f *FlexibleInt64) Scan(value interface{}) error {
	if value == nil {
		*f = 0
		return nil
	}

	switch v := value.(type) {
	case int64:
		*f = FlexibleInt64(v)
	case float64:
		*f = FlexibleInt64(int64(v))
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return err
		}
		*f = FlexibleInt64(i)
	default:
		return fmt.Errorf("cannot scan %T into FlexibleInt64", value)
	}
	return nil
}

func (f FlexibleInt64) Value() (driver.Value, error) {
	return int64(f), nil
}

func (f FlexibleInt64) Int64() int64 {
	return int64(f)
}

// Session represents an active user session.
// Sessions are stored in the database and referenced by their ID.
type Session struct {
	ID            string        `json:"id" db:"id"` // Unique identifier for the session (e.g., UUID)
	UserID        int           `json:"user_id" db:"user_id"`
	RefreshToken  string        `json:"refresh_token" db:"refresh_token"` // The refresh token
	CreatedAt     FlexibleInt64 `json:"created_at" db:"created_at"`
	LastRefreshed FlexibleInt64 `json:"last_refreshed" db:"last_refreshed"` // Timestamp of the last refresh token issuance
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

// NewSession creates a new session instance with a unique ID.
func NewSession(userID int) (*Session, error) {
	sessionID := uuid.New().String()
	refreshToken, err := generateRandomID(32)
	if err != nil {
		return nil, err
	}
	now := FlexibleInt64(time.Now().UTC().Unix())
	return &Session{
		ID:            sessionID,
		UserID:        userID,
		RefreshToken:  refreshToken,
		CreatedAt:     now,
		LastRefreshed: now,
	}, nil
}

// --- Database Methods ---

func DBInit(db *sqlx.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		refresh_token TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		last_refreshed INTEGER NOT NULL
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
	_, err := db.Exec("INSERT INTO sessions (id, user_id, refresh_token, created_at, last_refreshed) VALUES ($1, $2, $3, $4, $5)", s.ID, s.UserID, s.RefreshToken, s.CreatedAt, s.LastRefreshed)
	return err
}

func (s *Session) DBUpdateRefreshToken(db *sqlx.DB) (string, error) {
	fmt.Printf("Updating refresh token for session %s\n", s.ID)
	refreshToken, err := generateRandomID(32)
	if err != nil {
		return "", err
	}
	s.RefreshToken = refreshToken
	s.LastRefreshed = FlexibleInt64(time.Now().UTC().Unix())
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
	err := db.Select(&sessionIDs, "SELECT id FROM sessions WHERE last_refreshed < $1", FlexibleInt64(time.Now().UTC().Add(-sessionExpiry).Unix()))
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

func DBDeleteSessionsForUser(db *sqlx.DB, userID int) error {
	fmt.Printf("Deleting sessions for user %d\n", userID)
	_, err := db.Exec("DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}
