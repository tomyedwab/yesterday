package sessions

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

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
	RefreshToken            string        `json:"refresh_token" db:"refresh_token"`                         // The refresh token
	RefreshTokenFingerprint string        `json:"refresh_token_fingerprint" db:"refresh_token_fingerprint"` // The fingerprint of the refresh token
	UserID                  int           `json:"user_id" db:"user_id"`
	CreatedAt               FlexibleInt64 `json:"created_at" db:"created_at"`
	ExpiresAt               FlexibleInt64 `json:"expires_at" db:"expires_at"` // Timestamp of the last refresh token issuance
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
func NewSession(userID int, sessionExpiry time.Duration) (*Session, error) {
	refreshToken, err := generateRandomID(32)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &Session{
		RefreshToken: refreshToken,
		UserID:       userID,
		CreatedAt:    FlexibleInt64(now.Unix()),
		ExpiresAt:    FlexibleInt64(now.Add(sessionExpiry).Unix()),
	}, nil
}

// --- Database Methods ---

func DBInit(db *sqlx.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS sessions (
		refresh_token TEXT PRIMARY KEY,
		refresh_token_fingerprint TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL
	)
	`)
	return err
}

func DBGetSessionByRefreshToken(db *sqlx.DB, refreshToken string) (*Session, error) {
	var s Session
	err := db.Get(&s, "SELECT * FROM sessions WHERE refresh_token = $1", refreshToken)
	return &s, err
}

func (s *Session) DBCreate(db *sqlx.DB) error {
	fmt.Printf("Creating session for user %d\n", s.UserID)
	hash := sha256.Sum256([]byte(s.RefreshToken))
	fingerprint := hex.EncodeToString(hash[:])
	_, err := db.Exec("INSERT INTO sessions (refresh_token, refresh_token_fingerprint, user_id, created_at, expires_at) VALUES ($1, $2, $3, $4, $5)", s.RefreshToken, fingerprint, s.UserID, s.CreatedAt, s.ExpiresAt)
	return err
}

func (s *Session) DBUpdateRefreshToken(db *sqlx.DB, oldSessionExpiry, newSessionExpiry time.Duration) (string, error) {
	fmt.Printf("Updating refresh token for user %d\n", s.UserID)
	newSession, err := NewSession(s.UserID, newSessionExpiry)
	if err != nil {
		return "", err
	}
	err = newSession.DBCreate(db)
	if err != nil {
		return "", err
	}

	s.ExpiresAt = FlexibleInt64(time.Now().UTC().Add(oldSessionExpiry).Unix())
	_, err = db.Exec("UPDATE sessions SET expires_at = $1 WHERE refresh_token = $2", s.ExpiresAt, s.RefreshToken)

	return newSession.RefreshToken, err
}

func (s *Session) DBDelete(db *sqlx.DB) error {
	_, err := db.Exec("DELETE FROM sessions WHERE refresh_token = $1", s.RefreshToken)
	return err
}

func DBDeleteExpiredSessions(db *sqlx.DB, sessionExpiry time.Duration) error {
	// Delete sessions that have expired
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at < UNIXEPOCH()")
	if err != nil {
		return err
	}
	return nil
}

func DBDeleteSessionsForUser(db *sqlx.DB, userID int) error {
	fmt.Printf("Deleting sessions for user %d\n", userID)
	_, err := db.Exec("DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}
