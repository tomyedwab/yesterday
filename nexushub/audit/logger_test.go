package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates a temporary test database
func setupTestDB(t *testing.T) *sqlx.DB {
	tmpDir := t.TempDir()
	dbPath := path.Join(tmpDir, "test_audit.db")
	db := sqlx.MustConnect("sqlite3", dbPath)
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})
	return db
}

func TestNewLogger(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)

	if err != nil {
		t.Fatalf("NewLogger returned error: %v", err)
	}

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	if logger.db == nil {
		t.Fatal("Logger's internal db is nil")
	}
}

func TestDBInit(t *testing.T) {
	db := setupTestDB(t)
	err := DBInit(db)

	if err != nil {
		t.Fatalf("DBInit returned error: %v", err)
	}

	// Verify table exists
	var tableName string
	err = db.Get(&tableName, "SELECT name FROM sqlite_master WHERE type='table' AND name='audit_events'")
	if err != nil {
		t.Fatalf("Table 'audit_events' does not exist: %v", err)
	}

	// Verify indexes exist
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND tbl_name='audit_events'")
	if err != nil {
		t.Fatalf("Failed to query indexes: %v", err)
	}
	if count < 3 {
		t.Errorf("Expected at least 3 indexes, got %d", count)
	}
}

func TestTokenFingerprint(t *testing.T) {
	testToken := "test-token-123"
	expected := sha256.Sum256([]byte(testToken))
	expectedHex := hex.EncodeToString(expected[:])

	result := tokenFingerprint(testToken)

	if result != expectedHex {
		t.Errorf("Expected fingerprint %s, got %s", expectedHex, result)
	}

	// Test that same token produces same fingerprint
	result2 := tokenFingerprint(testToken)
	if result != result2 {
		t.Error("Same token should produce same fingerprint")
	}

	// Test that different tokens produce different fingerprints
	result3 := tokenFingerprint("different-token")
	if result == result3 {
		t.Error("Different tokens should produce different fingerprints")
	}

	// Test empty token
	emptyResult := tokenFingerprint("")
	if emptyResult != "" {
		t.Errorf("Empty token should produce empty fingerprint, got %s", emptyResult)
	}
}

func TestLogLogin(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	userID := 123
	refreshToken := "refresh-token-xyz"

	err = logger.LogLogin(userID, refreshToken)
	if err != nil {
		t.Fatalf("LogLogin failed: %v", err)
	}

	// Verify event was stored
	var event AuditEvent
	err = db.Get(&event, "SELECT * FROM audit_events WHERE event_type = $1", string(EventLogin))
	if err != nil {
		t.Fatalf("Failed to retrieve event: %v", err)
	}

	if event.EventType != string(EventLogin) {
		t.Errorf("Expected event_type '%s', got '%s'", EventLogin, event.EventType)
	}

	if event.UserID == nil || *event.UserID != userID {
		t.Errorf("Expected user_id %d, got %v", userID, event.UserID)
	}

	expectedFingerprint := tokenFingerprint(refreshToken)
	if event.RefreshTokenFingerprint != expectedFingerprint {
		t.Errorf("Expected refresh_token_fingerprint '%s', got '%s'", expectedFingerprint, event.RefreshTokenFingerprint)
	}

	if event.Timestamp == 0 {
		t.Error("Expected timestamp to be set")
	}
}

func TestLogLogout(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	userID := 456
	refreshToken := "logout-token-abc"

	err = logger.LogLogout(userID, refreshToken)
	if err != nil {
		t.Fatalf("LogLogout failed: %v", err)
	}

	// Verify event was stored
	var event AuditEvent
	err = db.Get(&event, "SELECT * FROM audit_events WHERE event_type = $1", string(EventLogout))
	if err != nil {
		t.Fatalf("Failed to retrieve event: %v", err)
	}

	if event.EventType != string(EventLogout) {
		t.Errorf("Expected event_type '%s', got '%s'", EventLogout, event.EventType)
	}

	if event.UserID == nil || *event.UserID != userID {
		t.Errorf("Expected user_id %d, got %v", userID, event.UserID)
	}

	expectedFingerprint := tokenFingerprint(refreshToken)
	if event.RefreshTokenFingerprint != expectedFingerprint {
		t.Errorf("Expected refresh_token_fingerprint '%s', got '%s'", expectedFingerprint, event.RefreshTokenFingerprint)
	}
}

func TestLogAccessTokenRefresh(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	userID := 789
	oldRefreshToken := "old-refresh-token"
	newRefreshToken := "new-refresh-token"
	accessToken := "access-token-123"

	err = logger.LogAccessTokenRefresh(userID, oldRefreshToken, newRefreshToken, accessToken)
	if err != nil {
		t.Fatalf("LogAccessTokenRefresh failed: %v", err)
	}

	// Verify event was stored
	var event AuditEvent
	err = db.Get(&event, "SELECT * FROM audit_events WHERE event_type = $1", string(EventAccessTokenRefresh))
	if err != nil {
		t.Fatalf("Failed to retrieve event: %v", err)
	}

	if event.EventType != string(EventAccessTokenRefresh) {
		t.Errorf("Expected event_type '%s', got '%s'", EventAccessTokenRefresh, event.EventType)
	}

	if event.UserID == nil || *event.UserID != userID {
		t.Errorf("Expected user_id %d, got %v", userID, event.UserID)
	}

	expectedOldFingerprint := tokenFingerprint(oldRefreshToken)
	if event.OldRefreshTokenFingerprint != expectedOldFingerprint {
		t.Errorf("Expected old_refresh_token_fingerprint '%s', got '%s'", expectedOldFingerprint, event.OldRefreshTokenFingerprint)
	}

	expectedNewFingerprint := tokenFingerprint(newRefreshToken)
	if event.NewRefreshTokenFingerprint != expectedNewFingerprint {
		t.Errorf("Expected new_refresh_token_fingerprint '%s', got '%s'", expectedNewFingerprint, event.NewRefreshTokenFingerprint)
	}

	expectedAccessFingerprint := tokenFingerprint(accessToken)
	if event.AccessTokenFingerprint != expectedAccessFingerprint {
		t.Errorf("Expected access_token_fingerprint '%s', got '%s'", expectedAccessFingerprint, event.AccessTokenFingerprint)
	}
}

func TestLogAccessTokenExpiry(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	accessToken := "expired-access-token"

	err = logger.LogAccessTokenExpiry(accessToken)
	if err != nil {
		t.Fatalf("LogAccessTokenExpiry failed: %v", err)
	}

	// Verify event was stored
	var event AuditEvent
	err = db.Get(&event, "SELECT * FROM audit_events WHERE event_type = $1", string(EventAccessTokenExpiry))
	if err != nil {
		t.Fatalf("Failed to retrieve event: %v", err)
	}

	if event.EventType != string(EventAccessTokenExpiry) {
		t.Errorf("Expected event_type '%s', got '%s'", EventAccessTokenExpiry, event.EventType)
	}

	expectedFingerprint := tokenFingerprint(accessToken)
	if event.AccessTokenFingerprint != expectedFingerprint {
		t.Errorf("Expected access_token_fingerprint '%s', got '%s'", expectedFingerprint, event.AccessTokenFingerprint)
	}
}

func TestLogInvalidRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	refreshToken := "invalid-refresh-token"

	err = logger.LogInvalidRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("LogInvalidRefreshToken failed: %v", err)
	}

	// Verify event was stored
	var event AuditEvent
	err = db.Get(&event, "SELECT * FROM audit_events WHERE event_type = $1", string(EventInvalidRefreshToken))
	if err != nil {
		t.Fatalf("Failed to retrieve event: %v", err)
	}

	if event.EventType != string(EventInvalidRefreshToken) {
		t.Errorf("Expected event_type '%s', got '%s'", EventInvalidRefreshToken, event.EventType)
	}

	expectedFingerprint := tokenFingerprint(refreshToken)
	if event.RefreshTokenFingerprint != expectedFingerprint {
		t.Errorf("Expected refresh_token_fingerprint '%s', got '%s'", expectedFingerprint, event.RefreshTokenFingerprint)
	}
}

func TestGetEventsByUserID(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	userID := 100
	// Log multiple events for the user
	logger.LogLogin(userID, "token1")
	logger.LogLogout(userID, "token2")
	logger.LogLogin(999, "token3") // Different user

	events, err := logger.GetEventsByUserID(userID, 10)
	if err != nil {
		t.Fatalf("GetEventsByUserID failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	// Verify they're for the correct user
	for _, event := range events {
		if event.UserID == nil || *event.UserID != userID {
			t.Errorf("Event has wrong user_id: %v", event.UserID)
		}
	}
}

func TestGetEventsByType(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log different types of events
	logger.LogLogin(100, "token1")
	logger.LogLogin(200, "token2")
	logger.LogLogout(300, "token3")

	events, err := logger.GetEventsByType(EventLogin, 10)
	if err != nil {
		t.Fatalf("GetEventsByType failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 login events, got %d", len(events))
	}

	for _, event := range events {
		if event.EventType != string(EventLogin) {
			t.Errorf("Event has wrong type: %s", event.EventType)
		}
	}
}

func TestGetRecentEvents(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log multiple events
	logger.LogLogin(100, "token1")
	time.Sleep(10 * time.Millisecond)
	logger.LogLogout(200, "token2")
	time.Sleep(10 * time.Millisecond)
	logger.LogLogin(300, "token3")

	events, err := logger.GetRecentEvents(2)
	if err != nil {
		t.Fatalf("GetRecentEvents failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	// Verify they're in descending timestamp order (most recent first)
	if len(events) == 2 && events[0].Timestamp < events[1].Timestamp {
		t.Error("Events should be in descending timestamp order")
	}
}

func TestDeleteOldEvents(t *testing.T) {
	db := setupTestDB(t)
	logger, err := NewLogger(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Manually insert old events
	oldTimestamp := time.Now().UTC().Add(-2 * time.Hour).Unix()
	_, err = db.Exec(`
		INSERT INTO audit_events (id, event_type, timestamp, user_id, refresh_token_fingerprint)
		VALUES ($1, $2, $3, $4, $5)`,
		"old-event-1", string(EventLogin), oldTimestamp, 100, "fingerprint1")
	if err != nil {
		t.Fatalf("Failed to insert old event: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO audit_events (id, event_type, timestamp, user_id, refresh_token_fingerprint)
		VALUES ($1, $2, $3, $4, $5)`,
		"old-event-2", string(EventLogout), oldTimestamp, 200, "fingerprint2")
	if err != nil {
		t.Fatalf("Failed to insert old event: %v", err)
	}

	// Also insert a recent event that should not be deleted
	logger.LogLogin(300, "token3")

	// Delete events older than 1 hour (should delete the 2 old ones)
	deleted, err := logger.DeleteOldEvents(1 * time.Hour)
	if err != nil {
		t.Fatalf("DeleteOldEvents failed: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected to delete 2 events, deleted %d", deleted)
	}

	// Verify only 1 event remains (the recent one)
	events, err := logger.GetRecentEvents(10)
	if err != nil {
		t.Fatalf("GetRecentEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event after deletion, got %d", len(events))
	}
}

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		expected  string
	}{
		{"Login", EventLogin, "login"},
		{"Logout", EventLogout, "logout"},
		{"AccessTokenRefresh", EventAccessTokenRefresh, "access_token_refresh"},
		{"AccessTokenExpiry", EventAccessTokenExpiry, "access_token_expiry"},
		{"InvalidRefreshToken", EventInvalidRefreshToken, "invalid_refresh_token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.eventType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.eventType))
			}
		})
	}
}
