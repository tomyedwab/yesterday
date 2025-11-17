package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	// AuditLoggerKey is the key used to store the AuditLogger in the context.
	AuditLoggerKey = "auditLogger"
)

// EventType represents the type of audit event
type EventType string

const (
	EventLogin                EventType = "login"
	EventLogout               EventType = "logout"
	EventAccessTokenRefresh   EventType = "access_token_refresh"
	EventAccessTokenExpiry    EventType = "access_token_expiry"
	EventInvalidRefreshToken  EventType = "invalid_refresh_token"
)

// AuditEvent represents an audit log entry in the database
type AuditEvent struct {
	ID                         string `db:"id"`
	EventType                  string `db:"event_type"`
	Timestamp                  int64  `db:"timestamp"`
	UserID                     *int   `db:"user_id"`                        // Nullable for events without user context
	RefreshTokenFingerprint    string `db:"refresh_token_fingerprint"`
	OldRefreshTokenFingerprint string `db:"old_refresh_token_fingerprint"`
	NewRefreshTokenFingerprint string `db:"new_refresh_token_fingerprint"`
	AccessTokenFingerprint     string `db:"access_token_fingerprint"`
}

// Logger handles audit logging for authentication and authorization events
type Logger struct {
	db *sqlx.DB
}

// NewLogger creates a new audit logger instance
func NewLogger(db *sqlx.DB) (*Logger, error) {
	if err := DBInit(db); err != nil {
		return nil, err
	}
	return &Logger{
		db: db,
	}, nil
}

// DBInit initializes the audit events database table
func DBInit(db *sqlx.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS audit_events (
		id TEXT PRIMARY KEY,
		event_type TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		user_id INTEGER,
		refresh_token_fingerprint TEXT,
		old_refresh_token_fingerprint TEXT,
		new_refresh_token_fingerprint TEXT,
		access_token_fingerprint TEXT
	)
	`)
	if err != nil {
		return err
	}

	// Create indexes for common queries
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON audit_events(user_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type)`)
	return err
}

// tokenFingerprint creates a SHA-256 hash of a token for audit logging
// This allows us to track token usage without storing the actual token value
func tokenFingerprint(token string) string {
	if token == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// insertEvent is a helper method to insert an audit event into the database
func (l *Logger) insertEvent(event *AuditEvent) error {
	_, err := l.db.Exec(`
		INSERT INTO audit_events (
			id, event_type, timestamp, user_id,
			refresh_token_fingerprint, old_refresh_token_fingerprint,
			new_refresh_token_fingerprint, access_token_fingerprint
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		event.ID,
		event.EventType,
		event.Timestamp,
		event.UserID,
		event.RefreshTokenFingerprint,
		event.OldRefreshTokenFingerprint,
		event.NewRefreshTokenFingerprint,
		event.AccessTokenFingerprint,
	)
	return err
}

// LogLogin logs a successful login event
func (l *Logger) LogLogin(userID int, refreshToken string) error {
	event := &AuditEvent{
		ID:                      uuid.New().String(),
		EventType:               string(EventLogin),
		Timestamp:               time.Now().UTC().Unix(),
		UserID:                  &userID,
		RefreshTokenFingerprint: tokenFingerprint(refreshToken),
	}
	return l.insertEvent(event)
}

// LogLogout logs a logout event
func (l *Logger) LogLogout(userID int, refreshToken string) error {
	event := &AuditEvent{
		ID:                      uuid.New().String(),
		EventType:               string(EventLogout),
		Timestamp:               time.Now().UTC().Unix(),
		UserID:                  &userID,
		RefreshTokenFingerprint: tokenFingerprint(refreshToken),
	}
	return l.insertEvent(event)
}

// LogAccessTokenRefresh logs an access token refresh event
func (l *Logger) LogAccessTokenRefresh(userID int, oldRefreshToken string, newRefreshToken string, accessToken string) error {
	event := &AuditEvent{
		ID:                         uuid.New().String(),
		EventType:                  string(EventAccessTokenRefresh),
		Timestamp:                  time.Now().UTC().Unix(),
		UserID:                     &userID,
		OldRefreshTokenFingerprint: tokenFingerprint(oldRefreshToken),
		NewRefreshTokenFingerprint: tokenFingerprint(newRefreshToken),
		AccessTokenFingerprint:     tokenFingerprint(accessToken),
	}
	return l.insertEvent(event)
}

// LogAccessTokenExpiry logs when an access token expires
func (l *Logger) LogAccessTokenExpiry(accessToken string) error {
	event := &AuditEvent{
		ID:                     uuid.New().String(),
		EventType:              string(EventAccessTokenExpiry),
		Timestamp:              time.Now().UTC().Unix(),
		AccessTokenFingerprint: tokenFingerprint(accessToken),
	}
	return l.insertEvent(event)
}

// LogInvalidRefreshToken logs when an expired or invalid refresh token is used
func (l *Logger) LogInvalidRefreshToken(refreshToken string) error {
	event := &AuditEvent{
		ID:                      uuid.New().String(),
		EventType:               string(EventInvalidRefreshToken),
		Timestamp:               time.Now().UTC().Unix(),
		RefreshTokenFingerprint: tokenFingerprint(refreshToken),
	}
	return l.insertEvent(event)
}

// GetEventsByUserID retrieves audit events for a specific user
func (l *Logger) GetEventsByUserID(userID int, limit int) ([]AuditEvent, error) {
	var events []AuditEvent
	err := l.db.Select(&events,
		"SELECT * FROM audit_events WHERE user_id = $1 ORDER BY timestamp DESC LIMIT $2",
		userID, limit)
	return events, err
}

// GetEventsByType retrieves audit events of a specific type
func (l *Logger) GetEventsByType(eventType EventType, limit int) ([]AuditEvent, error) {
	var events []AuditEvent
	err := l.db.Select(&events,
		"SELECT * FROM audit_events WHERE event_type = $1 ORDER BY timestamp DESC LIMIT $2",
		string(eventType), limit)
	return events, err
}

// GetRecentEvents retrieves the most recent audit events
func (l *Logger) GetRecentEvents(limit int) ([]AuditEvent, error) {
	var events []AuditEvent
	err := l.db.Select(&events,
		"SELECT * FROM audit_events ORDER BY timestamp DESC LIMIT $1",
		limit)
	return events, err
}

// DeleteOldEvents deletes audit events older than the specified duration
func (l *Logger) DeleteOldEvents(olderThan time.Duration) (int64, error) {
	threshold := time.Now().UTC().Add(-olderThan).Unix()
	result, err := l.db.Exec("DELETE FROM audit_events WHERE timestamp < $1", threshold)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
