package sessions

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/wasi/guest"
)

var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrTokenGeneration     = errors.New("failed to generate token")
	ErrSessionExpired      = errors.New("session expired")
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
	guest.WriteLog("Initializing session manager")
	err := DBInit(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	guest.WriteLog("Database initialized")

	m := &SessionManager{
		db:            db,
		accessExpiry:  accessTokenExpiry,
		sessionExpiry: sessionExpiry,
	}

	guest.WriteLog("SessionManager initialized")

	return m, nil
}

// CreateSession performs the following steps:
// 1. Creates a new Session object.
// 2. Generates a refresh token.
// 3. Stores the session details in the SessionStore (database).
// Returns the new session or an error.
func (m *SessionManager) CreateSession(userID int, applicationID string) (*Session, error) {
	session, err := NewSession(userID, applicationID)
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

func (m *SessionManager) GetSession(sessionID string) (*Session, error) {
	session, err := DBGetSessionByID(m.db, sessionID)
	if err != nil {
		return nil, err
	}

	if time.Since(session.LastRefreshed) > m.sessionExpiry {
		session.DBDelete(m.db)
		return nil, ErrSessionExpired
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

// RefreshAccessToken creates a new JWT access token and updates the session
// with a new refresh token. It returns the access and refresh tokens, or an
// error.
/*
func (m *SessionManager) RefreshAccessToken(session *Session, refreshToken, applicationID, profileData string) (string, string, error) {
	if session.RefreshToken != refreshToken {
		return "", "", ErrInvalidRefreshToken
	}

	if time.Since(session.LastRefreshed) > m.sessionExpiry {
		session.DBDelete(m.db)
		return "", "", ErrSessionExpired
	}

	// Calculate expiry time
	expiresAt := time.Now().UTC().Add(m.accessExpiry)

	// Create the JWT claims
	claims := util.YesterdayUserClaims{
		SessionID:   session.ID,
		Expiry:      expiresAt.Unix(),
		IssuedAt:    time.Now().UTC().Unix(),
		Application: applicationID,
		Profile:     profileData,
	}

	// Create the token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	tokenString, err := token.SignedString(m.jwtSecretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	// Update the session with the new refresh token
	refreshToken, err = session.DBUpdateRefreshToken(m.db)
	if err != nil {
		return "", "", fmt.Errorf("failed to update session with new refresh token: %w", err)
	}

	return tokenString, refreshToken, nil
}
*/
