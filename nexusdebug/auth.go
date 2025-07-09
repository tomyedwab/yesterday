// Package main implements authentication functionality for the NexusDebug CLI tool.
//
// This module integrates with the Go client library for authentication workflow,
// providing interactive username/password prompts and handling login flow against
// the Admin app's /public/login endpoint.
//
// Reference: spec/nexusdebug.md - Task nexusdebug-authentication
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
	"golang.org/x/term"
)

// AuthManager handles authentication operations for the NexusDebug CLI
type AuthManager struct {
	adminURL string
	client   *yesterdaygo.Client
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(adminURL string) *AuthManager {
	client := yesterdaygo.NewClient(adminURL)
	return &AuthManager{
		adminURL: adminURL,
		client:   client,
	}
}

// PromptCredentials interactively prompts the user for username and password
func (am *AuthManager) PromptCredentials() (username, password string, err error) {
	reader := bufio.NewReader(os.Stdin)

	// Prompt for username
	fmt.Print("Username: ")
	username, err = reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	if username == "" {
		return "", "", fmt.Errorf("username cannot be empty")
	}

	// Prompt for password (hidden input)
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // Add newline after password input

	password = strings.TrimSpace(string(passwordBytes))
	if password == "" {
		return "", "", fmt.Errorf("password cannot be empty")
	}

	return username, password, nil
}

// Login performs the authentication flow against the Admin app
func (am *AuthManager) Login(ctx context.Context) error {
	log.Printf("Starting authentication flow...")

	// Check if we already have a valid token
	if am.IsAuthenticated(ctx) {
		log.Printf("Already authenticated")
		return nil
	}

	// Prompt for credentials
	username, password, err := am.PromptCredentials()
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	log.Printf("Authenticating user: %s", username)

	// Perform login request
	if err := am.performLogin(ctx, username, password); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	log.Printf("Authentication successful")
	return nil
}

// performLogin executes the actual login request
func (am *AuthManager) performLogin(ctx context.Context, username, password string) error {
	log.Printf("Sending login request to %s/public/login", am.adminURL)

	// Use the actual client library login method
	if err := am.client.Login(ctx, username, password); err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}

	return nil
}

// IsAuthenticated checks if the client has a valid access token
func (am *AuthManager) IsAuthenticated(ctx context.Context) bool {
	return am.client.IsAuthenticated()
}

// RefreshAccessToken refreshes the access token using the stored refresh token
func (am *AuthManager) RefreshAccessToken(ctx context.Context) error {
	log.Printf("Refreshing access token...")

	// Use the actual client library refresh method
	if err := am.client.RefreshAccessToken(ctx); err != nil {
		return fmt.Errorf("failed to refresh access token: %w", err)
	}

	log.Printf("Access token refreshed successfully")
	return nil
}

// Logout terminates the current session and clears stored tokens
func (am *AuthManager) Logout(ctx context.Context) error {
	log.Printf("Logging out...")

	// Use the actual client library logout method
	if err := am.client.Logout(ctx); err != nil {
		log.Printf("Warning: logout request failed: %v", err)
		// Continue with cleanup even if logout request fails
	}

	log.Printf("Logout completed")
	return nil
}

// GetAuthenticationStatus returns a human-readable authentication status
func (am *AuthManager) GetAuthenticationStatus(ctx context.Context) string {
	if am.IsAuthenticated(ctx) {
		return "Authenticated"
	}
	return "Not authenticated"
}
