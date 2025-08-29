package yesterdaygo_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

// Example test demonstrating basic mock client usage
func ExampleMockClient_basic() {
	// Create a mock client
	client := yesterdaygo.NewMockClient()

	// Configure mock responses
	client.SetMockResponse("/public/login", http.StatusOK, nil)
	client.SetMockResponse("/public/access_token", http.StatusOK,
		yesterdaygo.AccessTokenResponse{AccessToken: "mock-token"})

	// Perform operations
	ctx := context.Background()
	err := client.Login(ctx, "testuser", "testpass")
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}

	err = client.RefreshAccessToken(ctx)
	if err != nil {
		fmt.Printf("Token refresh failed: %v\n", err)
		return
	}

	// Verify requests were made
	history := client.GetRequestHistory()
	fmt.Printf("Requests made: %d\n", len(history))
	fmt.Printf("Authentication state: %v\n", client.IsAuthenticated())

	// Output:
	// Requests made: 2
	// Authentication state: true
}

// Example test demonstrating mock event polling
func ExampleMockEventPoller() {
	client := yesterdaygo.NewMockClient()
	poller := client.GetMockEventPoller()

	// Start polling
	err := poller.StartEventPolling(1 * time.Second)
	if err != nil {
		fmt.Printf("Failed to start polling: %v\n", err)
		return
	}

	// Subscribe to events
	eventCh := poller.SubscribeToEvents()

	// Manually trigger events for testing
	go func() {
		time.Sleep(10 * time.Millisecond)
		poller.TriggerEvent(100)
		time.Sleep(10 * time.Millisecond)
		poller.TriggerEvent(101)
	}()

	// Receive events
	select {
	case eventNum := <-eventCh:
		fmt.Printf("Received event: %d\n", eventNum)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("Timeout waiting for event")
	}

	fmt.Printf("Current event number: %d\n", poller.GetCurrentEventNumber())
	fmt.Printf("Polling active: %v\n", poller.IsRunning())

	// Output:
	// Received event: 100
	// Current event number: 100
	// Polling active: true
}

// Example test demonstrating mock event publishing
func ExampleMockEventPublisher() {
	client := yesterdaygo.NewMockClient()
	publisher := client.GetMockEventPublisher()

	// Publish some events
	err := publisher.PublishEvent("user.created", map[string]string{"id": "123"})
	if err != nil {
		fmt.Printf("Failed to publish event: %v\n", err)
		return
	}

	err = publisher.PublishEvent("user.updated", map[string]string{"id": "123", "name": "John"})
	if err != nil {
		fmt.Printf("Failed to publish event: %v\n", err)
		return
	}

	// Verify published events
	events := publisher.GetPublishedEvents()
	fmt.Printf("Published events: %d\n", len(events))
	for _, event := range events {
		fmt.Printf("Event: %s\n", event.EventType)
	}

	// Output:
	// Published events: 2
	// Event: user.created
	// Event: user.updated
}

// TestMockClientBasicOperations demonstrates testing authentication flows
func TestMockClientBasicOperations(t *testing.T) {
	client := yesterdaygo.NewMockClient()

	// Test successful login
	client.SetMockResponse("/public/login", http.StatusOK, nil)

	ctx := context.Background()
	err := client.Login(ctx, "testuser", "password")
	if err != nil {
		t.Errorf("Expected successful login, got error: %v", err)
	}

	// Verify authentication state
	if !client.IsAuthenticated() {
		t.Error("Expected client to be authenticated after login")
	}

	// Use assertion helpers
	yesterdaygo.AssertAuthenticationCalled(t, client)
	yesterdaygo.AssertRequestCount(t, client, 1)
}

// TestMockClientErrorHandling demonstrates testing error scenarios
func TestMockClientErrorHandling(t *testing.T) {
	client := yesterdaygo.NewMockClient()

	// Configure error response
	expectedError := errors.New("invalid credentials")
	client.SetMockError("/public/login", expectedError)

	ctx := context.Background()
	err := client.Login(ctx, "testuser", "wrongpassword")
	if err == nil {
		t.Error("Expected login error, but got nil")
	}

	if err != expectedError {
		t.Errorf("Expected specific error %v, got %v", expectedError, err)
	}

	// Verify client is not authenticated
	if client.IsAuthenticated() {
		t.Error("Expected client to not be authenticated after failed login")
	}
}

// TestMockEventPollerControlledEvents demonstrates controlled event testing
func TestMockEventPollerControlledEvents(t *testing.T) {
	client := yesterdaygo.NewMockClient()
	poller := client.GetMockEventPoller()

	// Start polling
	err := poller.StartEventPolling(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to start event polling: %v", err)
	}

	yesterdaygo.AssertEventPollingStarted(t, poller)

	// Subscribe to events
	eventCh := poller.SubscribeToEvents()

	// Trigger specific events
	poller.TriggerEvent(42)
	poller.TriggerEvent(43)

	// Verify we receive the events
	select {
	case eventNum := <-eventCh:
		if eventNum != 42 {
			t.Errorf("Expected event number 42, got %d", eventNum)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for first event")
	}

	select {
	case eventNum := <-eventCh:
		if eventNum != 43 {
			t.Errorf("Expected event number 43, got %d", eventNum)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for second event")
	}

	// Verify current event number
	if poller.GetCurrentEventNumber() != 43 {
		t.Errorf("Expected current event number 43, got %d", poller.GetCurrentEventNumber())
	}
}

// TestMockEventPublisherVerification demonstrates event publishing verification
func TestMockEventPublisherVerification(t *testing.T) {
	client := yesterdaygo.NewMockClient()
	publisher := client.GetMockEventPublisher()

	// Publish test events
	testPayload := map[string]interface{}{
		"userId": 123,
		"action": "create",
	}

	err := publisher.PublishEvent("user.created", testPayload)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	err = publisher.PublishEvent("user.updated", map[string]string{"userId": "123"})
	if err != nil {
		t.Fatalf("Failed to publish second event: %v", err)
	}

	// Use assertion helpers
	yesterdaygo.AssertEventPublished(t, publisher, "user.created")
	yesterdaygo.AssertEventPublished(t, publisher, "user.updated")
	yesterdaygo.AssertEventPublishedWithPayload(t, publisher, "user.created", testPayload)

	// Verify event count
	events := publisher.GetPublishedEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 published events, got %d", len(events))
	}

	// Test clearing events
	publisher.ClearPublishedEvents()
	events = publisher.GetPublishedEvents()
	if len(events) != 0 {
		t.Errorf("Expected 0 events after clearing, got %d", len(events))
	}
}

// TestDataBuilders demonstrates using test data builders
func TestDataBuilders(t *testing.T) {
	client := yesterdaygo.NewMockClient()

	// Use login response builder
	loginResp := yesterdaygo.NewLoginResponse().
		WithStatus(http.StatusOK).
		WithRefreshToken("test-refresh-token").
		Build()

	client.SetMockResponse("/public/login", loginResp.StatusCode, loginResp.Body)
	client.SetMockHeaders("/public/login", loginResp.Headers)

	// Use access token response builder
	tokenResp := yesterdaygo.NewAccessTokenResponse().
		WithStatus(http.StatusOK).
		WithAccessToken("test-access-token").
		Build()

	client.SetMockResponse("/public/access_token", tokenResp.StatusCode, tokenResp.Body)

	// Test the configured responses
	ctx := context.Background()
	err := client.Login(ctx, "testuser", "password")
	if err != nil {
		t.Errorf("Login failed: %v", err)
	}

	err = client.RefreshAccessToken(ctx)
	if err != nil {
		t.Errorf("Token refresh failed: %v", err)
	}

	// Verify authentication flow worked
	yesterdaygo.AssertRequestMade(t, client, "POST", "/public/login")
	yesterdaygo.AssertRequestMade(t, client, "POST", "/public/access_token")
}

// TestAPIResponseBuilder demonstrates generic API response building
func TestAPIResponseBuilder(t *testing.T) {
	client := yesterdaygo.NewMockClient()

	// Build a custom API response
	apiResp := yesterdaygo.NewAPIResponse().
		WithStatus(http.StatusOK).
		WithBody(map[string]interface{}{
			"users": []map[string]interface{}{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			},
		}).
		WithHeader("X-Total-Count", "2").
		Build()

	client.SetMockResponse("/api/users", apiResp.StatusCode, apiResp.Body)
	client.SetMockHeaders("/api/users", apiResp.Headers)

	// Make the request
	ctx := context.Background()
	resp, err := client.Get(ctx, "/api/users", nil)
	if err != nil {
		t.Errorf("API request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify request was made
	yesterdaygo.AssertRequestMade(t, client, "GET", "/api/users")
}

// TestErrorScenarios demonstrates testing various error conditions
func TestErrorScenarios(t *testing.T) {
	client := yesterdaygo.NewMockClient()

	// Test network error
	networkErr := errors.New("network timeout")
	client.SetMockError("/api/data", networkErr)

	ctx := context.Background()
	_, err := client.Get(ctx, "/api/data", nil)
	if err != networkErr {
		t.Errorf("Expected network error, got: %v", err)
	}

	// Test unauthorized access
	client.SetMockResponse("/api/protected", 401, map[string]interface{}{"error": "unauthorized"})
	resp, err := client.Get(context.Background(), "/api/protected", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

// TestWaitForEventCondition demonstrates waiting for specific event conditions
func TestWaitForEventCondition(t *testing.T) {
	client := yesterdaygo.NewMockClient()
	poller := client.GetMockEventPoller()

	// Start polling
	err := poller.StartEventPolling(10 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to start polling: %v", err)
	}

	// Trigger event in background
	go func() {
		time.Sleep(50 * time.Millisecond)
		poller.TriggerEvent(100)
	}()

	// Wait for specific condition
	yesterdaygo.WaitForEventCondition(t, poller, func(eventNum int64) bool {
		return eventNum >= 100
	}, 200*time.Millisecond)

	// Verify the condition was met
	if poller.GetCurrentEventNumber() < 100 {
		t.Errorf("Expected event number >= 100, got %d", poller.GetCurrentEventNumber())
	}
}

// BenchmarkMockClientOperations benchmarks mock client performance
func BenchmarkMockClientOperations(b *testing.B) {
	client := yesterdaygo.NewMockClient()
	client.SetMockResponse("/api/test", http.StatusOK, map[string]string{"result": "success"})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Get(ctx, "/api/test", nil)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkMockEventPublisher benchmarks event publishing performance
func BenchmarkMockEventPublisher(b *testing.B) {
	client := yesterdaygo.NewMockClient()
	publisher := client.GetMockEventPublisher()

	testPayload := map[string]interface{}{
		"id":   123,
		"data": "test data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := publisher.PublishEvent("test.event", testPayload)
		if err != nil {
			b.Fatalf("Event publishing failed: %v", err)
		}
	}
}
