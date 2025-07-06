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
- **Request Utilities**: Generic HTTP methods with error handling
- **Error Handling**: Structured error types for different categories
- **Thread Safety**: Concurrent access protection with mutex synchronization

### 🚧 Coming Soon
- **Event Polling**: Asynchronous background polling for data change notifications
- **Generic Data Provider**: Type-safe data access with automatic refresh
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
   - Sends POST to `/api/access_token` with YRT cookie
   - Stores access token in memory

3. **Authenticated Requests**: Automatic authentication headers
   - Adds `Authorization: Bearer <token>` to API requests
   - Thread-safe token access

4. **Logout**: Clean session termination
   - Sends POST to `/public/logout`
   - Clears all stored tokens

## Thread Safety

The client is designed for concurrent use:

- Access tokens are protected by read-write mutexes
- Multiple goroutines can safely use the same client instance
- Authentication state is consistently managed across threads

## Development Status

This implementation covers the **Core Client Structure** task from the technical specification. Additional features like event polling, data providers, and event publishing are planned for future releases.

## License

See the main Yesterday project for license information.
