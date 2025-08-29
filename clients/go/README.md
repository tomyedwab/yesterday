# Yesterday Go Client

A Go client library for interacting with Yesterday's API. This client provides authentication, asynchronous event polling, and generic data providers that automatically refresh when data changes on the server.

## Installation

```bash
go get github.com/tomyedwab/yesterday/clients/go
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

func main() {
    // Create client
    client := yesterdaygo.NewClient("https://api.yesterday.localhost")

    // Initialize (attempt to refresh existing tokens)
    ctx := context.Background()
    if err := client.Initialize(ctx); err != nil {
        log.Printf("Warning: %v", err)
    }

    // Login
    if err := client.Login(ctx, "username", "password"); err != nil {
        log.Fatal(err)
    }
    defer client.Logout(ctx)

    // Client is now authenticated and ready to use
    log.Println("Successfully authenticated!")
}
```

## Features

### ✅ Implemented
- **Core Client Structure**: HTTP client with functional options and configuration
- **Authentication System**: Username/password login with token management
- **Event Polling**: Asynchronous background polling for data change notifications
- **Request Utilities**: Generic HTTP methods with error handling
- **Error Handling**: Structured error types for different categories
- **Thread Safety**: Concurrent access protection with mutex synchronization
- **Generic Data Provider**: Type-safe data access with automatic refresh

### Coming Soon
- **Event Publishing**: Reliable event publishing with queuing and retry logic
- **Testing Support**: Mock client and testing utilities
- **Advanced Configuration**: Environment variable support and logging

## Configuration Options

The client supports various configuration options:

```go
client := yesterdaygo.NewClient("https://api.yesterday.localhost",
    // Custom HTTP client with timeout
    yesterdaygo.WithHTTPClient(&http.Client{
        Timeout: 60 * time.Second,
    }),
    // Custom refresh token storage path
    yesterdaygo.WithRefreshTokenPath("/path/to/refresh_token"),
)
```

## Error Handling

The client provides structured error types:

```go
if err := client.Login(ctx, username, password); err != nil {
    switch {
    case yesterdaygo.IsAuthenticationError(err):
        log.Println("Invalid credentials")
    case yesterdaygo.IsNetworkError(err):
        log.Println("Network connectivity issue")
    case yesterdaygo.IsValidationError(err):
        log.Println("Invalid input")
    case yesterdaygo.IsAPIError(err):
        log.Println("Server error")
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

## Available Error Types

- `ErrorTypeAuthentication`: Invalid credentials or unauthorized access
- `ErrorTypeNetwork`: Network connectivity issues
- `ErrorTypeValidation`: Invalid input or missing required fields
- `ErrorTypeAPI`: Server-side errors with HTTP status codes
- `ErrorTypeUnknown`: Unexpected errors

## Authentication Flow

1. **Login**: Authenticate with username/password
   - Sends POST to `/public/login`
   - Extracts refresh token from `YRT` cookie
   - Stores refresh token securely

2. **Token Refresh**: Automatic access token management
   - Uses stored refresh token to get access tokens
   - Sends POST to `/public/access_token` with YRT cookie
   - Stores access token in memory

3. **Authenticated Requests**: Automatic authentication headers
   - Adds `Authorization: Bearer <token>` to API requests
   - Thread-safe token access

4. **Logout**: Clean session termination
   - Sends POST to `/public/logout`
   - Clears all stored tokens

## Event Polling

The client provides asynchronous event polling to detect data changes on the server:

```go
// Get the event poller
poller := client.GetEventPoller()

// Subscribe to event notifications
eventCh := poller.SubscribeToEvents()

// Start polling with custom interval
if err := poller.StartEventPolling(3 * time.Second); err != nil {
    log.Fatal(err)
}
defer poller.StopEventPolling()

// Listen for events
go func() {
    for eventNumber := range eventCh {
        fmt.Printf("New event: %d\n", eventNumber)
        // Data has changed - refresh your data providers
    }
}()
```

### Event Polling Features

- **Background Polling**: Runs in a separate goroutine
- **Multiple Subscribers**: Support for multiple event listeners
- **Configurable Intervals**: Customize polling frequency
- **Thread Safe**: Concurrent access to event state
- **Graceful Shutdown**: Clean resource cleanup

### Event Polling Methods

```go
// Start/stop polling
poller.StartEventPolling(interval time.Duration) error
poller.StopEventPolling()

// Event subscription
eventCh := poller.SubscribeToEvents()

// Wait for next event with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
eventNumber, err := poller.WaitForEvent(ctx)

// Status and configuration
poller.IsRunning() bool
poller.GetCurrentEventNumber() int64
poller.SetPollInterval(interval time.Duration)
poller.GetSubscriberCount() int
```

## Thread Safety

The client is designed for concurrent use:

- Access tokens are protected by read-write mutexes
- Event polling state is thread-safe with proper synchronization
- Multiple goroutines can safely use the same client instance
- Authentication state is consistently managed across threads

## Generic Data Provider

The Generic Data Provider offers type-safe data access with automatic refresh capabilities based on event changes. It uses Go generics to provide compile-time type safety while integrating seamlessly with the event polling system.

### Basic Usage

```go
// Define your data structure
type User struct {
    ID       int    `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
}

// Create a data provider
userProvider := yesterdaygo.NewDataProvider[User](client, "/api/users/123", nil)
defer userProvider.Close()

// Get data (fetches from API on first call, returns cached on subsequent calls)
user, err := userProvider.Get()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User: %s (%s)\n", user.Username, user.Email)
```

### Data Provider with Parameters

```go
// Create provider with query parameters
params := map[string]interface{}{
    "page":     1,
    "per_page": 10,
    "active":   true,
}

usersProvider := yesterdaygo.NewDataProvider[UserList](client, "/api/users", params)
defer usersProvider.Close()

// Get data with parameters
userList, err := usersProvider.Get()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Retrieved %d users\n", len(userList.Users))

// Update parameters
newParams := map[string]interface{}{"page": 2, "per_page": 20}
if err := usersProvider.SetParams(newParams); err != nil {
    log.Fatal(err)
}
```

### Automatic Refresh with Subscriptions

```go
// Start event polling first
poller := client.GetEventPoller()
if err := poller.StartEventPolling(5 * time.Second); err != nil {
    log.Fatal(err)
}
defer poller.StopEventPolling()

// Create data provider and subscribe to automatic updates
userProvider := yesterdaygo.NewDataProvider[User](client, "/api/users/123", nil)
defer userProvider.Close()

// Subscribe to automatic refresh when events change
err := userProvider.Subscribe(func(user User) {
    fmt.Printf("User data updated: %s (%s)\n", user.Username, user.Email)
})
if err != nil {
    log.Fatal(err)
}

// Get initial data
user, err := userProvider.Get()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Initial user: %s\n", user.Username)

// The callback will be called automatically when server data changes
```

### Manual Refresh

```go
// Manually refresh data from the API
if err := userProvider.Refresh(); err != nil {
    log.Printf("Failed to refresh: %v", err)
}

// Get the refreshed data
user, err := userProvider.Get()
if err != nil {
    log.Fatal(err)
}
```

### Data Provider API Methods

```go
// Core methods
NewDataProvider[T](client, uri, params) *DataProvider[T]
provider.Get() (T, error)
provider.Refresh() error

// Subscription methods
provider.Subscribe(callback func(T)) error
provider.Unsubscribe()
provider.IsSubscribed() bool

// Parameter management
provider.SetParams(params map[string]interface{}) error
provider.GetParams() map[string]interface{}

// Status and metadata
provider.GetURI() string
provider.GetLastEventNumber() int64
provider.Close()
```

### Data Provider Features

- **Type Safety**: Uses Go generics for compile-time type checking
- **Automatic Refresh**: Integrates with event polling for automatic data updates
- **Smart Caching**: Caches data and only refetches when server events indicate changes
- **Thread Safety**: All operations are safe for concurrent use
- **Flexible Parameters**: Supports dynamic query parameters
- **Resource Management**: Proper cleanup with Close() method
- **Event Integration**: Seamlessly works with the EventPoller system

## Event Publishing

The `EventPublisher` provides reliable event publishing with automatic queuing, retry logic, and exponential backoff.

### Basic Usage

```go
// Get the event publisher (automatically created with client)
publisher := client.GetEventPublisher()

// Publish an event
err := publisher.PublishEvent("user.created", map[string]interface{}{
    "userId": "user123",
    "email":  "user@example.com",
})
if err != nil {
    log.Printf("Failed to publish event: %v", err)
}
```

### Custom Configuration

```go
// Create publisher with custom settings
publisher := NewEventPublisher(client,
    WithRetryBackoff(2*time.Second),
    WithMaxRetries(5),
    WithBatchSize(10),
)
```

### Graceful Shutdown

```go
// Wait for all events to be published
err := publisher.FlushEvents(30 * time.Second)
if err != nil {
    log.Printf("Some events may not have been published: %v", err)
}

// Stop the publisher
publisher.Stop()
```

### Event Publisher API Methods

```go
// Core methods
NewEventPublisher(client, options...) *EventPublisher
publisher.PublishEvent(eventType string, payload interface{}) error
publisher.FlushEvents(timeout time.Duration) error
publisher.Stop()

// Monitoring methods
publisher.IsRunning() bool
publisher.GetQueueLength() int

// Configuration options
WithRetryBackoff(backoff time.Duration) PublisherOption
WithMaxRetries(maxRetries int) PublisherOption
WithBatchSize(batchSize int) PublisherOption
```

### Event Publisher Features

- **Reliable Delivery**: Automatic queuing with persistent retry until success
- **Exponential Backoff**: 1s, 2s, 4s, 8s progression up to 5 minutes maximum
- **Thread Safety**: All operations are safe for concurrent use
- **Graceful Shutdown**: FlushEvents() waits for pending events before shutdown
- **Flexible Payloads**: Supports any JSON-serializable data as event payload
- **Random Client IDs**: Each event gets a unique client ID for API tracking
- **Error Classification**: Distinguishes between retryable and non-retryable errors

## Development Status

This implementation covers the **Core Client Structure**, **Event Polling**, **Generic Data Provider**, and **Event Publishing** tasks from the technical specification.

### Completed Features
- ✅ Core client with HTTP operations and configuration
- ✅ Authentication system with login/logout and token management
- ✅ Event polling system with background goroutine and subscriber pattern
- ✅ Generic data provider with type-safe automatic refresh
- ✅ Event publishing with queuing, retry logic, and exponential backoff
- ✅ Structured error handling with type checking
- ✅ Thread-safe concurrent access
- ✅ Comprehensive testing utilities with mock client and assertion helpers

## Testing Utilities

The Yesterday Go client includes comprehensive testing utilities to help you write robust tests for applications using the client library. The mock implementations provide controllable behavior for all client operations.

### MockClient Usage

```go
import (
    "testing"
    yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

func TestYourApplication(t *testing.T) {
    // Create mock client
    client := yesterdaygo.NewMockClient()

    // Configure mock responses
    client.SetMockResponse("/public/login", 200, nil)
    client.SetMockResponse("/api/users", 200, map[string]interface{}{
        "users": []map[string]interface{}{
            {"id": 1, "name": "Alice"},
            {"id": 2, "name": "Bob"},
        },
    })

    // Test your application logic
    err := client.Login(ctx, "testuser", "password")
    if err != nil {
        t.Errorf("Login failed: %v", err)
    }

    // Verify requests were made
    yesterdaygo.AssertAuthenticationCalled(t, client)
    yesterdaygo.AssertRequestCount(t, client, 1)
}
```

### MockEventPoller for Event Testing

```go
func TestEventHandling(t *testing.T) {
    client := yesterdaygo.NewMockClient()
    poller := client.GetMockEventPoller()

    // Start polling
    err := poller.StartEventPolling(100 * time.Millisecond)
    if err != nil {
        t.Fatalf("Failed to start polling: %v", err)
    }

    // Subscribe to events
    eventCh := poller.SubscribeToEvents()

    // Trigger test events
    go func() {
        time.Sleep(10 * time.Millisecond)
        poller.TriggerEvent(100)
        poller.TriggerEvent(101)
    }()

    // Verify events are received
    select {
    case eventNum := <-eventCh:
        if eventNum != 100 {
            t.Errorf("Expected event 100, got %d", eventNum)
        }
    case <-time.After(200 * time.Millisecond):
        t.Error("Timeout waiting for event")
    }

    yesterdaygo.AssertEventPollingStarted(t, poller)
}
```

### MockEventPublisher for Publishing Tests

```go
func TestEventPublishing(t *testing.T) {
    client := yesterdaygo.NewMockClient()
    publisher := client.GetMockEventPublisher()

    // Publish test events
    err := publisher.PublishEvent("user.created", map[string]string{"id": "123"})
    if err != nil {
        t.Fatalf("Failed to publish event: %v", err)
    }

    // Verify events were published
    yesterdaygo.AssertEventPublished(t, publisher, "user.created")

    // Check published events
    events := publisher.GetPublishedEvents()
    if len(events) != 1 {
        t.Errorf("Expected 1 event, got %d", len(events))
    }
}
```

### Test Data Builders

Use the built-in test data builders to create consistent mock responses:

```go
func TestWithBuilders(t *testing.T) {
    client := yesterdaygo.NewMockClient()

    // Build login response with refresh token
    loginResp := yesterdaygo.NewLoginResponse().
        WithStatus(200).
        WithRefreshToken("test-refresh-token").
        Build()

    client.SetMockResponse("/public/login", loginResp.StatusCode, loginResp.Body)
    client.SetMockHeaders("/public/login", loginResp.Headers)

    // Build access token response
    tokenResp := yesterdaygo.NewAccessTokenResponse().
        WithAccessToken("test-access-token").
        Build()

    client.SetMockResponse("/public/access_token", tokenResp.StatusCode, tokenResp.Body)

    // Test authentication flow
    ctx := context.Background()
    err := client.Login(ctx, "testuser", "password")
    if err != nil {
        t.Errorf("Login failed: %v", err)
    }
}
```

### Assertion Helpers

The library includes several assertion helpers for common testing scenarios:

```go
// Request verification
yesterdaygo.AssertRequestMade(t, client, "POST", "/public/login")
yesterdaygo.AssertRequestCount(t, client, 2)

// Authentication verification
yesterdaygo.AssertAuthenticationCalled(t, client)

// Event verification
yesterdaygo.AssertEventPollingStarted(t, poller)
yesterdaygo.AssertEventPublished(t, publisher, "user.created")
yesterdaygo.AssertEventPublishedWithPayload(t, publisher, "user.created", payload)

// Error handling
yesterdaygo.AssertNoErrors(t, []error{err1, err2})
```

### Integration Testing

For integration tests against real API endpoints:

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    config := yesterdaygo.NewIntegrationTestConfig()
    config.BaseURL = "http://localhost:8080"
    config.Username = "testuser"
    config.Password = "testpass"

    client, err := yesterdaygo.CreateIntegrationTestClient(config)
    if err != nil {
        t.Fatalf("Failed to create integration test client: %v", err)
    }

    // Test against real API
    ctx := context.Background()
    resp, err := client.Get(ctx, "/api/status", nil)
    if err != nil {
        t.Errorf("Status check failed: %v", err)
    }

    if resp.StatusCode != 200 {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}
```

### Testing Utilities API

#### MockClient Methods
```go
// Configuration
NewMockClient() *MockClient
client.SetMockResponse(uri string, statusCode int, response interface{})
client.SetMockError(uri string, err error)
client.SetMockHeaders(uri string, headers map[string]string)
client.SetAuthenticated(authenticated bool)

// Verification
client.GetRequestHistory() []MockRequest
client.ClearRequestHistory()
```

#### MockEventPoller Methods
```go
// Control
poller.StartEventPolling(interval time.Duration) error
poller.StopEventPolling()
poller.TriggerEvent(eventNumber int64)

// Status
poller.GetCurrentEventNumber() int64
poller.IsRunning() bool
poller.SubscribeToEvents() <-chan int64
```

#### MockEventPublisher Methods
```go
// Publishing
publisher.PublishEvent(eventType string, payload interface{}) error
publisher.FlushEvents(timeout time.Duration) error

// Verification
publisher.GetPublishedEvents() []MockPublishedEvent
publisher.ClearPublishedEvents()
publisher.IsRunning() bool
```

### Upcoming Features
- 🔲 Advanced configuration and logging

## License

See the main Yesterday project for license information.
