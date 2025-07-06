# Technical Product Specification: Go Yesterday Client

**Reference Design Document:** [design/clients/go.md](../../design/clients/go.md)

## Introduction

This specification defines a Go client library that provides utilities for connecting any Go application (CLI, server, etc.) to Yesterday's API. The client provides authentication, asynchronous event polling, and generic data providers that automatically refresh when data changes on the server. The library abstracts the complexity of Yesterday's event-driven architecture and provides a simple, idiomatic Go interface for API interaction.

The implementation will reside in `clients/go/` and provide both synchronous and asynchronous patterns for API interaction, with automatic data freshness management through Yesterday's event numbering system.

**Related Components:**
- Yesterday API Server - Provides the REST endpoints and event numbering system
- Authentication Service - Handles username/password validation and session management

## Task `go-client-core-client`: Core Client Structure
**Reference:** design/clients/go.md  
**Implementation status:** Completed (2025-07-05)  
**Files:** `clients/go/client.go`, `clients/go/auth.go`, `clients/go/errors.go`, `clients/go/doc.go`

**Details:**
- Define `Client` struct with base configuration:
  - `baseURL string`: API server base URL (e.g., "https://api.yesterday.localhost")
  - `httpClient *http.Client`: Customizable HTTP client for requests
  - `refreshTokenPath string`: File in which to persist the refresh token,
    defaults to ~/.yesterday/refresh_token
- Implement `NewClient(baseURL string, options ...ClientOption) *Client` constructor
- Add structured error types for API errors, network errors, and authentication failures

## Task `go-client-authentication`: Authentication and Session Management
**Reference:** design/clients/go.md  
**Implementation status:** Completed (2025-07-05)  
**Files:** `clients/go/auth.go`, `clients/go/client.go`

**Details:**
- Implement `Login(username, password string) error` method for credential-based authentication:
  - POST request to `/public/login` endpoint
  - Extract the refresh token from the response cookie named "YRT" and store in `refreshTokenPath`
  - Handle authentication errors with specific error types
- Implement `Logout() error` method for session termination:
  - POST request to `/public/logout` endpoint
  - Clear stored authentication token
- Implement `RefreshAccessToken() error` helper method
  - Called at client initialization
  - Uses the refresh token stored in `refreshTokenPath` to get a new access token
  - Calls /api/access_token with the refresh token in the YRT cookie header
  - Stores the new access token from the JSON response's 'access_token' field in memory
  - Falls back on username/password login if anything goes wrong
- Implement `IsAuthenticated() bool` helper method
- Add middleware for automatic authentication header injection in all authenticated requests using Bearer <access_token>

## Task `go-client-event-polling`: Event Number Polling System  
**Reference:** design/clients/go.md  
**Implementation status:** Completed (2025-07-05)  
**Files:** `clients/go/events.go`, `clients/go/events_example_test.go`

**Details:**
- Define `EventPoller` struct for managing event number polling:
  - `client *Client`: Reference to parent client
  - `currentEventNumber int64`: Last known event number
  - `pollInterval time.Duration`: Polling frequency (default 5 seconds)
  - `subscribers []chan int64`: List of event number subscribers
- Implement `StartEventPolling(interval time.Duration) error` method:
  - Start background goroutine for periodic polling
  - GET request to `/api/poll?e=<event_number>` endpoint, where event_number is the last seen event number
  - A 304 response indicates no new events have been saved since the last poll
  - Compare received event number with cached value
  - Notify all subscribers when event number changes
- Implement `StopEventPolling()` method for graceful shutdown
- Implement `SubscribeToEvents() <-chan int64` for event notifications
- Implement `GetCurrentEventNumber() int64` for synchronous access
- Add exponential backoff for polling failures with maximum retry limit
- Ensure thread-safe access to event number and subscriber list with mutex protection

## Task `go-client-data-provider`: Generic Data Provider
**Reference:** design/clients/go.md  
**Implementation status:** Completed (2025-01-05)  
**Files:** `clients/go/provider.go`, `clients/go/provider_example_test.go`

**Details:**
- Define `DataProvider[T any]` generic struct for type-safe data management:
  - `client *Client`: Reference to parent client for API requests
  - `uri string`: API endpoint URI for data fetching
  - `params map[string]interface{}`: Query parameters for requests
  - `data T`: Cached data of generic type
  - `lastEventNumber int64`: Event number when data was last fetched
  - `refreshCallback func(T)`: Optional callback for data change notifications
- Implement `NewDataProvider[T](client *Client, uri string, params map[string]interface{}) *DataProvider[T]` constructor
- Implement `Get() (T, error)` method for synchronous data access:
  - Return cached data if event number hasn't changed
  - Fetch fresh data if event number is newer
  - Update cache and last event number after successful fetch
- Implement `Subscribe(callback func(T))` method for asynchronous updates:
  - Register callback for data change notifications
  - Automatically refresh data when event number changes
  - Call callback with new data after successful refresh
- Implement `Refresh() error` method for manual data refresh
- Add generic JSON unmarshaling with proper error handling for type safety
- Ensure thread-safe access to cached data and metadata with RWMutex

## Task `go-client-event-publisher`: Generic Event Publishing Utility
**Reference:** design/clients/go.md  
**Implementation status:** Completed (2025-01-05)  
**Files:** `clients/go/publisher.go`, `clients/go/publisher_example_test.go`

**Details:**
- Define `EventPublisher` struct for reliable event publishing:
  - `client *Client`: Reference to parent client for API requests
  - `queue []PendingEvent`: Queue of events awaiting publication
  - `retryBackoff time.Duration`: Base backoff duration for retry attempts
  - `maxRetries int`: Maximum retry attempts per event (default 10)
  - `batchSize int`: Number of events to send in a single request (default 1)
- Define `PendingEvent` struct with event metadata:
  - `id string`: Unique identifier for the event
  - `eventType string`: Type of event being published
  - `payload interface{}`: Event data payload
  - `attempts int`: Number of retry attempts made
  - `lastAttempt time.Time`: Timestamp of last publish attempt
- Implement `NewEventPublisher(client *Client, options ...PublisherOption) *EventPublisher` constructor
- Implement `PublishEvent(eventType string, payload interface{}) error` method:
  - Generate unique event ID and add to publish queue
  - Trigger immediate publish attempt if queue was empty
  - Return immediately without waiting for publish confirmation
- Implement background publishing with exponential backoff:
  - Continuous goroutine processing queued events
  - POST requests to `/api/publish?cid=<clientId>` endpoint with one event at a
    time and a randomly-generated client ID per event
  - Exponential backoff: 1s, 2s, 4s, 8s, up to 5 minutes maximum
  - Remove successfully published events from queue
  - Persistent retry for failed events until max attempts reached
- Implement `FlushEvents(timeout time.Duration) error` for graceful shutdown:
  - Block until all queued events are published or timeout reached
  - Return error if events remain in queue after timeout
- Add event queue persistence for reliability across application restarts
- Ensure thread-safe queue operations with mutex protection

## Task `go-client-testing-utilities`: Testing Support and Mocks
**Reference:** design/clients/go.md  
**Implementation status:** Not started  
**Files:** `clients/go/testing.go`

**Details:**
- Define `MockClient` struct implementing the same interface as `Client`:
  - Configurable responses for different API endpoints
  - Request recording for verification in tests
  - Controllable authentication state
- Implement test helpers:
  - `NewMockClient() *MockClient`: Create mock client instance
  - `SetMockResponse(uri string, response interface{})`: Configure endpoint responses
  - `SetMockError(uri string, err error)`: Configure endpoint errors
  - `GetRequestHistory() []MockRequest`: Retrieve recorded requests
- Implement `MockEventPoller` for testing event-driven functionality:
  - Controllable event number changes
  - Synchronous event triggering for deterministic tests
- Add assertion helpers for common testing scenarios:
  - `AssertRequestMade(t *testing.T, uri string)`
  - `AssertAuthenticationCalled(t *testing.T)`
  - `AssertEventPollingStarted(t *testing.T)`
- Implement test data builders for common API response structures
- Add integration test utilities for testing against real API endpoints

## Implementation Summary

The Go Yesterday client provides a comprehensive, idiomatic Go interface to Yesterday's API with the following key features:

**Core Architecture:**
- Clean separation between authentication, data access, event management, and publishing
- Generic data providers with type safety and automatic refresh capabilities
- Event-driven data synchronization using Yesterday's event numbering system
- Reliable event publishing with queuing, retry logic, and exponential backoff
- Comprehensive error handling with structured error types

**Key Components:**
- `Client`: Core HTTP client with authentication and configuration management
- `EventPoller`: Background service for monitoring data changes
- `DataProvider[T]`: Generic, type-safe data access with automatic refresh
- `EventPublisher`: Reliable event publishing with queuing and retry logic
- Configuration system with functional options and environment variable support
- Comprehensive testing utilities including mocks and test helpers

**Advanced Features:**
- Automatic retry with exponential backoff for resilience
- Thread-safe concurrent access to cached data and event state
- Configurable polling intervals and request timeouts
- Debug logging and request/response tracing capabilities
- Production-ready error handling and recovery mechanisms

The implementation provides both synchronous and asynchronous access patterns, allowing developers to choose the most appropriate approach for their use case while maintaining data consistency through Yesterday's event system.
