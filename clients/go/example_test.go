package yesterdaygo_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

func ExampleClient_basic() {
	// Create a new client
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")

	// Initialize client (attempts to refresh existing tokens)
	ctx := context.Background()
	if err := client.Initialize(ctx); err != nil {
		log.Printf("Warning during initialization: %v", err)
	}

	// Login with credentials
	if err := client.Login(ctx, "myusername", "mypassword"); err != nil {
		log.Fatal(err)
	}

	// Check authentication status
	if client.IsAuthenticated() {
		fmt.Println("Successfully authenticated")
	}

	// Logout when done
	defer func() {
		if err := client.Logout(ctx); err != nil {
			log.Printf("Error during logout: %v", err)
		}
	}()

	// Output: Successfully authenticated
}

func ExampleClient_withOptions() {
	// Create client with custom options
	customHTTPClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	client := yesterdaygo.NewClient("https://api.yesterday.localhost",
		yesterdaygo.WithHTTPClient(customHTTPClient),
		yesterdaygo.WithRefreshTokenPath("/custom/path/refresh_token"),
	)

	fmt.Printf("Base URL: %s\n", client.GetBaseURL())
	fmt.Printf("HTTP Timeout: %v\n", client.GetHTTPClient().Timeout)
	fmt.Printf("Refresh Token Path: %s\n", client.GetRefreshTokenPath())

	// Output:
	// Base URL: https://api.yesterday.localhost
	// HTTP Timeout: 1m0s
	// Refresh Token Path: /custom/path/refresh_token
}

func ExampleClient_errorHandling() {
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	ctx := context.Background()

	// Attempt login with invalid credentials
	if err := client.Login(ctx, "invalid", "credentials"); err != nil {
		if yesterdaygo.IsAuthenticationError(err) {
			fmt.Println("Authentication failed: Invalid credentials")
		} else if yesterdaygo.IsNetworkError(err) {
			fmt.Println("Network error occurred")
		} else if yesterdaygo.IsValidationError(err) {
			fmt.Println("Validation error")
		} else {
			fmt.Printf("Other error: %v\n", err)
		}
	}

	// This would output something like:
	// Authentication failed: Invalid credentials
}

func ExampleNewError() {
	// Create different types of errors
	networkErr := yesterdaygo.NewNetworkError("Connection timeout", fmt.Errorf("dial tcp: timeout"))
	authErr := yesterdaygo.NewAuthenticationError("Invalid token")
	apiErr := yesterdaygo.NewAPIError("Server error", 500)
	validationErr := yesterdaygo.NewValidationError("Missing required field")

	fmt.Printf("Network error: %v\n", networkErr)
	fmt.Printf("Auth error: %v\n", authErr)
	fmt.Printf("API error: %v\n", apiErr)
	fmt.Printf("Validation error: %v\n", validationErr)

	// Check error types
	fmt.Printf("Is network error: %v\n", yesterdaygo.IsNetworkError(networkErr))
	fmt.Printf("Is auth error: %v\n", yesterdaygo.IsAuthenticationError(authErr))

	// Output:
	// Network error: Connection timeout: dial tcp: timeout
	// Auth error: Invalid token
	// API error: Server error
	// Validation error: Missing required field
	// Is network error: true
	// Is auth error: true
}
