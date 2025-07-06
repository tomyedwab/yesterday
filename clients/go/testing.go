package yesterdaygo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// MockRequest represents a recorded HTTP request for testing
type MockRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    interface{}
}

// MockResponse represents a configured mock response
type MockResponse struct {
	StatusCode int
	Body       interface{}
	Headers    map[string]string
	Error      error
}

// MockClient implements the same interface as Client for testing purposes
type MockClient struct {
	baseURL          string
	authenticated    bool
	responses        map[string]*MockResponse
	requestHistory   []MockRequest
	mu               sync.RWMutex
	eventPoller      *MockEventPoller
	eventPublisher   *MockEventPublisher
}

// NewMockClient creates a new mock client instance for testing
func NewMockClient() *MockClient {
	client := &MockClient{
		baseURL:        "https://mock.yesterday.localhost",
		responses:      make(map[string]*MockResponse),
		requestHistory: make([]MockRequest, 0),
	}
	
	client.eventPoller = NewMockEventPoller(client)
	client.eventPublisher = NewMockEventPublisher(client)
	
	return client
}

// SetMockResponse configures a mock response for a specific URI pattern
func (m *MockClient) SetMockResponse(uri string, statusCode int, response interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.responses[uri] = &MockResponse{
		StatusCode: statusCode,
		Body:       response,
		Headers:    make(map[string]string),
	}
}

// SetMockError configures a mock error for a specific URI pattern
func (m *MockClient) SetMockError(uri string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.responses[uri] = &MockResponse{
		Error: err,
	}
}

// SetMockHeaders configures mock response headers for a specific URI pattern
func (m *MockClient) SetMockHeaders(uri string, headers map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if resp, exists := m.responses[uri]; exists {
		resp.Headers = headers
	}
}

// GetRequestHistory returns all recorded requests for verification
func (m *MockClient) GetRequestHistory() []MockRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent external modification
	history := make([]MockRequest, len(m.requestHistory))
	copy(history, m.requestHistory)
	return history
}

// ClearRequestHistory clears the recorded request history
func (m *MockClient) ClearRequestHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.requestHistory = make([]MockRequest, 0)
}

// SetAuthenticated sets the mock authentication state
func (m *MockClient) SetAuthenticated(authenticated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.authenticated = authenticated
}

// recordRequest records a request for later verification
func (m *MockClient) recordRequest(method, path string, body interface{}, headers map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.requestHistory = append(m.requestHistory, MockRequest{
		Method:  method,
		Path:    path,
		Headers: headers,
		Body:    body,
	})
}

// getMockResponse retrieves the configured mock response for a URI
func (m *MockClient) getMockResponse(uri string) *MockResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.responses[uri]
}

// Mock implementations of Client interface methods

// GetBaseURL returns the mock client's base URL
func (m *MockClient) GetBaseURL() string {
	return m.baseURL
}

// GetHTTPClient returns a mock HTTP client
func (m *MockClient) GetHTTPClient() *http.Client {
	return &http.Client{}
}

// GetRefreshTokenPath returns a mock refresh token path
func (m *MockClient) GetRefreshTokenPath() string {
	return "/tmp/mock_refresh_token"
}

// makeRequest simulates an HTTP request and returns the configured mock response
func (m *MockClient) makeRequest(ctx context.Context, method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	m.recordRequest(method, path, body, headers)
	
	mockResp := m.getMockResponse(path)
	if mockResp == nil {
		// Default response if no mock configured
		recorder := httptest.NewRecorder()
		recorder.WriteHeader(200)
		return recorder.Result(), nil
	}
	
	if mockResp.Error != nil {
		return nil, mockResp.Error
	}
	
	// Create a mock HTTP response
	recorder := httptest.NewRecorder()
	
	// Set headers
	for key, value := range mockResp.Headers {
		recorder.Header().Set(key, value)
	}
	
	// Set status code and body
	recorder.WriteHeader(mockResp.StatusCode)
	if mockResp.Body != nil {
		if bodyBytes, err := json.Marshal(mockResp.Body); err == nil {
			recorder.Write(bodyBytes)
		}
	}
	
	return recorder.Result(), nil
}

// Get performs a mock GET request
func (m *MockClient) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return m.makeRequest(ctx, "GET", path, nil, headers)
}

// Post performs a mock POST request
func (m *MockClient) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return m.makeRequest(ctx, "POST", path, body, headers)
}

// Put performs a mock PUT request
func (m *MockClient) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return m.makeRequest(ctx, "PUT", path, body, headers)
}

// Delete performs a mock DELETE request
func (m *MockClient) Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return m.makeRequest(ctx, "DELETE", path, nil, headers)
}

// Login simulates user authentication
func (m *MockClient) Login(ctx context.Context, username, password string) error {
	m.recordRequest("POST", "/public/login", LoginRequest{Username: username, Password: password}, nil)
	
	mockResp := m.getMockResponse("/public/login")
	if mockResp != nil && mockResp.Error != nil {
		return mockResp.Error
	}
	
	m.SetAuthenticated(true)
	return nil
}

// Logout simulates session termination
func (m *MockClient) Logout(ctx context.Context) error {
	m.recordRequest("POST", "/public/logout", nil, nil)
	
	mockResp := m.getMockResponse("/public/logout")
	if mockResp != nil && mockResp.Error != nil {
		return mockResp.Error
	}
	
	m.SetAuthenticated(false)
	return nil
}

// RefreshAccessToken simulates access token refresh
func (m *MockClient) RefreshAccessToken(ctx context.Context) error {
	m.recordRequest("POST", "/api/access_token", nil, nil)
	
	mockResp := m.getMockResponse("/api/access_token")
	if mockResp != nil && mockResp.Error != nil {
		return mockResp.Error
	}
	
	return nil
}

// IsAuthenticated returns the mock authentication state
func (m *MockClient) IsAuthenticated() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.authenticated
}

// Initialize performs mock initialization
func (m *MockClient) Initialize(ctx context.Context) error {
	return nil
}

// GetEventPoller returns the mock event poller
func (m *MockClient) GetEventPoller() *EventPoller {
	// Return nil since we use MockEventPoller separately
	return nil
}

// GetMockEventPoller returns the mock event poller for testing
func (m *MockClient) GetMockEventPoller() *MockEventPoller {
	return m.eventPoller
}

// GetEventPublisher returns the mock event publisher
func (m *MockClient) GetEventPublisher() *EventPublisher {
	// Return nil since we use MockEventPublisher separately
	return nil
}

// GetMockEventPublisher returns the mock event publisher for testing
func (m *MockClient) GetMockEventPublisher() *MockEventPublisher {
	return m.eventPublisher
}

// MockEventPoller provides controllable event polling for testing
type MockEventPoller struct {
	client              interface{} // MockClient reference
	currentEventNumber  int64
	subscribers         []chan int64
	running             bool
	mu                  sync.RWMutex
}

// NewMockEventPoller creates a new mock event poller
func NewMockEventPoller(client interface{}) *MockEventPoller {
	return &MockEventPoller{
		client:      client,
		subscribers: make([]chan int64, 0),
	}
}

// StartEventPolling simulates starting event polling
func (m *MockEventPoller) StartEventPolling(interval time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.running = true
	return nil
}

// StopEventPolling simulates stopping event polling
func (m *MockEventPoller) StopEventPolling() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.running = false
	
	// Close all subscriber channels
	for _, ch := range m.subscribers {
		close(ch)
	}
	m.subscribers = make([]chan int64, 0)
}

// SubscribeToEvents returns a channel for event number notifications
func (m *MockEventPoller) SubscribeToEvents() <-chan int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	ch := make(chan int64, 10)
	m.subscribers = append(m.subscribers, ch)
	return ch
}

// TriggerEvent manually triggers an event with the specified event number
func (m *MockEventPoller) TriggerEvent(eventNumber int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.currentEventNumber = eventNumber
	
	// Notify all subscribers
	for _, ch := range m.subscribers {
		select {
		case ch <- eventNumber:
		default:
			// Skip if channel is full
		}
	}
}

// GetCurrentEventNumber returns the current event number
func (m *MockEventPoller) GetCurrentEventNumber() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentEventNumber
}

// IsRunning returns whether polling is active
func (m *MockEventPoller) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// WaitForEvent simulates waiting for the next event
func (m *MockEventPoller) WaitForEvent(ctx context.Context) (int64, error) {
	ch := m.SubscribeToEvents()
	select {
	case eventNum := <-ch:
		return eventNum, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// SetPollInterval simulates setting the polling interval
func (m *MockEventPoller) SetPollInterval(interval time.Duration) {
	// No-op for mock
}

// MockEventPublisher provides controllable event publishing for testing
type MockEventPublisher struct {
	client         interface{} // MockClient reference
	publishedEvents []MockPublishedEvent
	running        bool
	mu             sync.RWMutex
}

// MockPublishedEvent represents an event that was published during testing
type MockPublishedEvent struct {
	EventType string
	Payload   interface{}
	Timestamp time.Time
}

// NewMockEventPublisher creates a new mock event publisher
func NewMockEventPublisher(client interface{}) *MockEventPublisher {
	return &MockEventPublisher{
		client:          client,
		publishedEvents: make([]MockPublishedEvent, 0),
		running:         true,
	}
}

// PublishEvent simulates publishing an event
func (m *MockEventPublisher) PublishEvent(eventType string, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.publishedEvents = append(m.publishedEvents, MockPublishedEvent{
		EventType: eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	})
	
	return nil
}

// FlushEvents simulates flushing pending events
func (m *MockEventPublisher) FlushEvents(timeout time.Duration) error {
	// Immediate return for mock - all events are "published" immediately
	return nil
}

// Stop simulates stopping the event publisher
func (m *MockEventPublisher) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.running = false
}

// IsRunning returns whether the publisher is running
func (m *MockEventPublisher) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetQueueLength simulates returning queue length (always 0 for mock)
func (m *MockEventPublisher) GetQueueLength() int {
	return 0
}

// GetPublishedEvents returns all events published during testing
func (m *MockEventPublisher) GetPublishedEvents() []MockPublishedEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent external modification
	events := make([]MockPublishedEvent, len(m.publishedEvents))
	copy(events, m.publishedEvents)
	return events
}

// ClearPublishedEvents clears the published events history
func (m *MockEventPublisher) ClearPublishedEvents() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.publishedEvents = make([]MockPublishedEvent, 0)
}

// Test Helper Functions and Assertions

// AssertRequestMade verifies that a request was made to the specified URI
func AssertRequestMade(t *testing.T, client *MockClient, method, uri string) {
	t.Helper()
	
	history := client.GetRequestHistory()
	for _, req := range history {
		if req.Method == method && req.Path == uri {
			return // Request found
		}
	}
	
	t.Errorf("Expected %s request to %s was not made. Request history: %+v", method, uri, history)
}

// AssertRequestCount verifies the total number of requests made
func AssertRequestCount(t *testing.T, client *MockClient, expectedCount int) {
	t.Helper()
	
	history := client.GetRequestHistory()
	if len(history) != expectedCount {
		t.Errorf("Expected %d requests, but got %d. Request history: %+v", expectedCount, len(history), history)
	}
}

// AssertAuthenticationCalled verifies that login was attempted
func AssertAuthenticationCalled(t *testing.T, client *MockClient) {
	t.Helper()
	AssertRequestMade(t, client, "POST", "/public/login")
}

// AssertEventPollingStarted verifies that event polling was started
func AssertEventPollingStarted(t *testing.T, poller *MockEventPoller) {
	t.Helper()
	
	if !poller.IsRunning() {
		t.Error("Expected event polling to be started, but it is not running")
	}
}

// AssertEventPublished verifies that an event was published
func AssertEventPublished(t *testing.T, publisher *MockEventPublisher, eventType string) {
	t.Helper()
	
	events := publisher.GetPublishedEvents()
	for _, event := range events {
		if event.EventType == eventType {
			return // Event found
		}
	}
	
	t.Errorf("Expected event type '%s' to be published, but it was not found in: %+v", eventType, events)
}

// AssertEventPublishedWithPayload verifies that an event was published with specific payload
func AssertEventPublishedWithPayload(t *testing.T, publisher *MockEventPublisher, eventType string, expectedPayload interface{}) {
	t.Helper()
	
	events := publisher.GetPublishedEvents()
	for _, event := range events {
		if event.EventType == eventType {
			if fmt.Sprintf("%v", event.Payload) == fmt.Sprintf("%v", expectedPayload) {
				return // Event with payload found
			}
		}
	}
	
	t.Errorf("Expected event type '%s' with payload '%v' to be published, but it was not found in: %+v", eventType, expectedPayload, events)
}

// AssertNoErrors is a helper to check that no errors occurred
func AssertNoErrors(t *testing.T, errs []error) {
	t.Helper()
	
	for _, err := range errs {
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

// Test Data Builders

// LoginResponseBuilder helps build mock login responses
type LoginResponseBuilder struct {
	response *MockResponse
}

// NewLoginResponse creates a new login response builder
func NewLoginResponse() *LoginResponseBuilder {
	return &LoginResponseBuilder{
		response: &MockResponse{
			StatusCode: http.StatusOK,
			Headers:    make(map[string]string),
		},
	}
}

// WithStatus sets the HTTP status code
func (b *LoginResponseBuilder) WithStatus(status int) *LoginResponseBuilder {
	b.response.StatusCode = status
	return b
}

// WithRefreshToken sets the refresh token cookie
func (b *LoginResponseBuilder) WithRefreshToken(token string) *LoginResponseBuilder {
	b.response.Headers["Set-Cookie"] = fmt.Sprintf("YRT=%s; Path=/; HttpOnly", token)
	return b
}

// WithError sets an error for the response
func (b *LoginResponseBuilder) WithError(err error) *LoginResponseBuilder {
	b.response.Error = err
	return b
}

// Build returns the constructed mock response
func (b *LoginResponseBuilder) Build() *MockResponse {
	return b.response
}

// AccessTokenResponseBuilder helps build mock access token responses
type AccessTokenResponseBuilder struct {
	response *MockResponse
}

// NewAccessTokenResponse creates a new access token response builder
func NewAccessTokenResponse() *AccessTokenResponseBuilder {
	return &AccessTokenResponseBuilder{
		response: &MockResponse{
			StatusCode: http.StatusOK,
			Headers:    make(map[string]string),
		},
	}
}

// WithStatus sets the HTTP status code
func (b *AccessTokenResponseBuilder) WithStatus(status int) *AccessTokenResponseBuilder {
	b.response.StatusCode = status
	return b
}

// WithAccessToken sets the access token in the response body
func (b *AccessTokenResponseBuilder) WithAccessToken(token string) *AccessTokenResponseBuilder {
	b.response.Body = AccessTokenResponse{AccessToken: token}
	return b
}

// WithError sets an error for the response
func (b *AccessTokenResponseBuilder) WithError(err error) *AccessTokenResponseBuilder {
	b.response.Error = err
	return b
}

// Build returns the constructed mock response
func (b *AccessTokenResponseBuilder) Build() *MockResponse {
	return b.response
}

// APIResponseBuilder helps build generic API responses
type APIResponseBuilder struct {
	response *MockResponse
}

// NewAPIResponse creates a new API response builder
func NewAPIResponse() *APIResponseBuilder {
	return &APIResponseBuilder{
		response: &MockResponse{
			StatusCode: http.StatusOK,
			Headers:    make(map[string]string),
		},
	}
}

// WithStatus sets the HTTP status code
func (b *APIResponseBuilder) WithStatus(status int) *APIResponseBuilder {
	b.response.StatusCode = status
	return b
}

// WithBody sets the response body
func (b *APIResponseBuilder) WithBody(body interface{}) *APIResponseBuilder {
	b.response.Body = body
	return b
}

// WithHeader adds a response header
func (b *APIResponseBuilder) WithHeader(key, value string) *APIResponseBuilder {
	b.response.Headers[key] = value
	return b
}

// WithError sets an error for the response
func (b *APIResponseBuilder) WithError(err error) *APIResponseBuilder {
	b.response.Error = err
	return b
}

// Build returns the constructed mock response
func (b *APIResponseBuilder) Build() *MockResponse {
	return b.response
}

// Integration Test Utilities

// IntegrationTestConfig holds configuration for integration tests
type IntegrationTestConfig struct {
	BaseURL    string
	Username   string
	Password   string
	SkipLogin  bool
	Timeout    time.Duration
}

// NewIntegrationTestConfig creates a new integration test configuration
func NewIntegrationTestConfig() *IntegrationTestConfig {
	return &IntegrationTestConfig{
		BaseURL: "http://localhost:8080",
		Timeout: 30 * time.Second,
	}
}

// CreateIntegrationTestClient creates a real client for integration testing
func CreateIntegrationTestClient(config *IntegrationTestConfig) (*Client, error) {
	client := NewClient(config.BaseURL)
	
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	
	if err := client.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}
	
	if !config.SkipLogin && config.Username != "" && config.Password != "" {
		if err := client.Login(ctx, config.Username, config.Password); err != nil {
			return nil, fmt.Errorf("failed to login: %w", err)
		}
	}
	
	return client, nil
}

// WaitForEventCondition waits for a specific condition on event numbers
func WaitForEventCondition(t *testing.T, poller *MockEventPoller, condition func(int64) bool, timeout time.Duration) {
	t.Helper()
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			t.Error("Timeout waiting for event condition")
			return
		case <-ticker.C:
			if condition(poller.GetCurrentEventNumber()) {
				return
			}
		}
	}
}
