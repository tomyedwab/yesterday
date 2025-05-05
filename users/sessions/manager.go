package sessions

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tomyedwab/yesterday/database"
)

var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrTokenGeneration     = errors.New("failed to generate token")
	ErrSessionExpired      = errors.New("session expired")
)

// SessionManager handles the lifecycle of user sessions and refresh tokens.
type SessionManager struct {
	database      *database.Database
	accessExpiry  time.Duration // How long access tokens are valid
	sessionExpiry time.Duration // How long sessions are valid
	jwtSecretKey  []byte        // The secret key for JWT signing
}

func LoadJWTSecretKey(path string) ([]byte, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		// If the file doesn't exist, generate a new key
		if os.IsNotExist(err) {
			b := make([]byte, 32)
			_, err := rand.Read(b)
			if err != nil {
				return nil, fmt.Errorf("failed to generate random JWT secret key: %w", err)
			}
			if err := os.WriteFile(path, b, 0600); err != nil {
				return nil, fmt.Errorf("failed to write JWT secret key: %w", err)
			}
			key = b
		} else {
			return nil, fmt.Errorf("failed to read JWT secret key: %w", err)
		}
	}
	return key, nil
}

// NewManager creates and initializes a new SessionManager.
// It requires a SessionStore implementation and token durations.
func NewManager(database *database.Database, accessTokenExpiry, sessionExpiry time.Duration, jwtSecretKeyPath string) (*SessionManager, error) {
	err := DBInit(database.GetDB())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	jwtSecretKey, err := LoadJWTSecretKey(jwtSecretKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load JWT secret key: %w", err)
	}

	m := &SessionManager{
		database:      database,
		accessExpiry:  accessTokenExpiry,
		sessionExpiry: sessionExpiry,
		jwtSecretKey:  jwtSecretKey,
	}

	fmt.Println("SessionManager initialized")

	return m, nil
}

// CreateSession performs the following steps:
// 1. Creates a new Session object.
// 2. Generates a refresh token.
// 3. Stores the session details in the SessionStore (database).
// Returns the new session or an error.
func (m *SessionManager) CreateSession(userID int) (*Session, error) {
	session := NewSession(userID)

	// Store the session in the database
	if err := session.DBCreate(m.database.GetDB()); err != nil {
		// Handle DB error (e.g., constraints violation, connection issues)
		return nil, err
	}

	return session, nil
}

func (m *SessionManager) GetSession(sessionID string) (*Session, error) {
	session, err := DBGetSessionByID(m.database.GetDB(), sessionID)
	if err != nil {
		return nil, err
	}

	if time.Since(session.LastRefreshed) > m.sessionExpiry {
		session.DBDelete(m.database.GetDB())
		return nil, ErrSessionExpired
	}

	return session, nil
}

// Delete sessions that have been inactive for a while
func (m *SessionManager) DeleteExpiredSessions() error {
	return DBDeleteExpiredSessions(m.database.GetDB(), m.sessionExpiry)
}

// RefreshAccessToken creates a new JWT access token and updates the session
// with a new refresh token. It returns the access and refresh tokens, or an
// error.
func (m *SessionManager) RefreshAccessToken(session *Session, refreshToken, applicationName, profileData string) (string, string, error) {
	if session.RefreshToken != refreshToken {
		return "", "", ErrInvalidRefreshToken
	}

	if time.Since(session.LastRefreshed) > m.sessionExpiry {
		session.DBDelete(m.database.GetDB())
		return "", "", ErrSessionExpired
	}

	// Calculate expiry time
	expiresAt := time.Now().UTC().Add(m.accessExpiry)

	// Create the JWT claims
	claims := jwt.MapClaims{
		"session_id": session.ID,
		"exp":        expiresAt.Unix(),
		"iat":        time.Now().UTC().Unix(),
		"app":        applicationName,
		"pro":        profileData,
	}

	// Create the token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	tokenString, err := token.SignedString(m.jwtSecretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	// Update the session with the new refresh token
	refreshToken, err = session.DBUpdateRefreshToken(m.database.GetDB())
	if err != nil {
		return "", "", fmt.Errorf("failed to update session with new refresh token: %w", err)
	}

	return tokenString, refreshToken, nil
}
