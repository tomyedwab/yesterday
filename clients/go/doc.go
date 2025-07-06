// Package yesterdaygo provides a Go client library for interacting with Yesterday's API.
//
// The client provides authentication, asynchronous event polling, and generic data providers
// that automatically refresh when data changes on the server. It abstracts the complexity
// of Yesterday's event-driven architecture and provides a simple, idiomatic Go interface
// for API interaction.
//
// # Basic Usage
//
//	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
//	
//	// Initialize client (attempts to refresh existing tokens)
//	if err := client.Initialize(context.Background()); err != nil {
//		log.Printf("Warning: %v", err)
//	}
//	
//	// Login with credentials
//	if err := client.Login(context.Background(), "username", "password"); err != nil {
//		log.Fatal(err)
//	}
//	
//	// Check authentication status
//	if client.IsAuthenticated() {
//		log.Println("Successfully authenticated")
//	}
//	
//	// Logout when done
//	defer client.Logout(context.Background())
//
// # Configuration Options
//
// The client can be configured with various options:
//
//	client := yesterdaygo.NewClient("https://api.yesterday.localhost",
//		yesterdaygo.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
//		yesterdaygo.WithRefreshTokenPath("/custom/path/token"),
//	)
//
// # Error Handling
//
// The client provides structured error types for different categories of errors:
//
//	if err := client.Login(ctx, username, password); err != nil {
//		if yesterdaygo.IsAuthenticationError(err) {
//			log.Println("Invalid credentials")
//		} else if yesterdaygo.IsNetworkError(err) {
//			log.Println("Network connectivity issue")
//		} else {
//			log.Printf("Other error: %v", err)
//		}
//	}
//
// # Thread Safety
//
// The client is designed to be thread-safe and can be used concurrently from multiple
// goroutines. Access tokens are protected by internal synchronization.
package yesterdaygo
