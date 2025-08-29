package sessions

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/nexushub/types"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrTokenGeneration     = errors.New("failed to generate token")
	ErrSessionExpired      = errors.New("session expired")
)

const (
	// SessionManagerKey is the key used to store the SessionManager in the context.
	SessionManagerKey = "sessionManager"
)

// SessionManager handles the lifecycle of user sessions and refresh tokens.
type SessionManager struct {
	db            *sqlx.DB
	accessExpiry  time.Duration // How long access tokens are valid
	sessionExpiry time.Duration // How long sessions are valid
}

// NewManager creates and initializes a new SessionManager.
// It requires a SessionStore implementation and token durations.
func NewManager(db *sqlx.DB, accessTokenExpiry, sessionExpiry time.Duration) (*SessionManager, error) {
	log.Printf("Initializing session manager")
	err := DBInit(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	log.Printf("Database initialized")

	m := &SessionManager{
		db:            db,
		accessExpiry:  accessTokenExpiry,
		sessionExpiry: sessionExpiry,
	}

	log.Printf("SessionManager initialized")

	return m, nil
}

func (m *SessionManager) CreateSession(userID int) (*Session, error) {
	session, err := NewSession(userID)
	if err != nil {
		return nil, err
	}

	// Store the session in the database
	if err := session.DBCreate(m.db); err != nil {
		// Handle DB error (e.g., constraints violation, connection issues)
		return nil, err
	}

	return session, nil
}

func (m *SessionManager) GetSessionByRefreshToken(refreshToken string) (*Session, error) {
	session, err := DBGetSessionByRefreshToken(m.db, refreshToken)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (m *SessionManager) DeleteSessionsForUser(userID int) error {
	return DBDeleteSessionsForUser(m.db, userID)
}

// Delete sessions that have been inactive for a while
func (m *SessionManager) DeleteExpiredSessions() error {
	return DBDeleteExpiredSessions(m.db, m.sessionExpiry)
}

// GetAccessToken creates a new access token which is stored in-memory in
// NexusHub, and rotates the refresh token in the database.
func (m *SessionManager) CreateAccessToken(session *Session) (*types.AccessTokenResponse, error) {
	if time.Since(time.Unix(int64(session.LastRefreshed), 0)) > m.sessionExpiry {
		session.DBDelete(m.db)
		return nil, ErrSessionExpired
	}

	// Calculate expiry time
	expiresAt := time.Now().UTC().Add(m.accessExpiry).Unix()

	// Update the session with the new refresh token
	refreshToken, err := session.DBUpdateRefreshToken(m.db)
	if err != nil {
		return nil, fmt.Errorf("failed to update session with new refresh token: %w", err)
	}

	return &types.AccessTokenResponse{
		Expiry:       expiresAt,
		RefreshToken: refreshToken,
		AccessToken:  uuid.New().String(),
	}, nil
}
