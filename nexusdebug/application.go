// Package main implements debug application lifecycle management for the NexusDebug CLI tool.
//
// This module handles the complete lifecycle of debug applications including creation,
// configuration, installation, monitoring, and cleanup via NexusHub's debug API endpoints.
//
// Reference: spec/nexusdebug.md - Task nexusdebug-application-management
package nexusdebug

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

// DebugApplication represents a debug application instance
type DebugApplication struct {
	ID               string `json:"id"`
	AppID            string `json:"appId"`
	DisplayName      string `json:"displayName"`
	HostName         string `json:"hostName"`
	DbName           string `json:"dbName"`
	StaticServiceURL string `json:"staticServiceUrl,omitempty"`
	Status           string `json:"status"`
	CreatedAt        string `json:"createdAt"`
}

// ApplicationStatus represents the current status of a debug application
type ApplicationStatus struct {
	ApplicationID string                 `json:"applicationId"`
	Status        string                 `json:"status"`
	ProcessID     int                    `json:"processId,omitempty"`
	Port          int                    `json:"port,omitempty"`
	HealthCheck   string                 `json:"healthCheck,omitempty"`
	Error         string                 `json:"error,omitempty"`
	LastUpdated   string                 `json:"lastUpdated"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ApplicationManager handles debug application lifecycle operations
type ApplicationManager struct {
	client           *yesterdaygo.Client
	currentApp       *DebugApplication
	appName          string
	staticServiceURL string
}

// NewApplicationManager creates a new application manager
func NewApplicationManager(client *yesterdaygo.Client, appName, staticServiceURL string) *ApplicationManager {
	return &ApplicationManager{
		client:           client,
		appName:          appName,
		staticServiceURL: staticServiceURL,
	}
}

// generateAppIdentifiers generates unique identifiers from the application name
func (am *ApplicationManager) generateAppIdentifiers() (appID, displayName, hostName, dbName string) {
	// Clean the app name to create valid identifiers
	cleanName := strings.ToLower(strings.ReplaceAll(am.appName, " ", "-"))
	cleanName = strings.ReplaceAll(cleanName, "_", "-")

	// Generate identifiers
	appID = fmt.Sprintf("debug-%s", cleanName)
	displayName = fmt.Sprintf("Debug: %s", am.appName)
	hostName = fmt.Sprintf("%s.debug", cleanName)
	dbName = fmt.Sprintf("debug_%s.db", strings.ReplaceAll(cleanName, "-", "_"))

	return appID, displayName, hostName, dbName
}

// CreateApplication creates a new debug application via the NexusHub API
func (am *ApplicationManager) CreateApplication(ctx context.Context) (*DebugApplication, error) {
	log.Printf("Creating debug application for: %s", am.appName)

	// Generate application identifiers
	appID, displayName, hostName, dbName := am.generateAppIdentifiers()

	// Prepare the request payload
	createRequest := map[string]interface{}{
		"appId":       appID,
		"displayName": displayName,
		"hostName":    hostName,
		"dbName":      dbName,
	}

	// Add static service URL if provided
	if am.staticServiceURL != "" {
		createRequest["staticServiceUrl"] = am.staticServiceURL
		log.Printf("Configuring static service URL: %s", am.staticServiceURL)
	}

	log.Printf("Application identifiers:")
	log.Printf("  App ID: %s", appID)
	log.Printf("  Display Name: %s", displayName)
	log.Printf("  Host Name: %s", hostName)
	log.Printf("  Database Name: %s", dbName)

	// Make the API request
	response, err := am.client.Post(ctx, "/debug/application", createRequest, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create debug application: %w", err)
	}
	defer response.Body.Close()

	// Parse the response
	var app DebugApplication
	if err := json.NewDecoder(response.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to parse application creation response: %w", err)
	}

	am.currentApp = &app
	log.Printf("Debug application created successfully with ID: %s", app.ID)
	log.Printf("Application status: %s", app.Status)

	return &app, nil
}

// CleanupExistingApplication removes any existing debug application with the same identifiers
func (am *ApplicationManager) CleanupExistingApplication(ctx context.Context) error {
	if am.currentApp == nil {
		return nil
	}

	log.Printf("Cleaning up existing debug application: %s", am.currentApp.ID)

	// Stop the application if it's running
	if err := am.StopApplication(ctx); err != nil {
		log.Printf("Warning: failed to stop application during cleanup: %v", err)
		// Continue with cleanup even if stop fails
	}

	// Delete the application
	_, err := am.client.Delete(ctx, fmt.Sprintf("/debug/application/%s", am.currentApp.ID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete debug application: %w", err)
	}

	log.Printf("Debug application cleaned up successfully")
	am.currentApp = nil
	return nil
}

// InstallApplication installs and starts the debug application
func (am *ApplicationManager) InstallApplication(ctx context.Context) error {
	if am.currentApp == nil {
		return fmt.Errorf("no debug application to install")
	}

	log.Printf("Installing debug application: %s", am.currentApp.ID)

	// Make the install request
	_, err := am.client.Post(ctx, fmt.Sprintf("/debug/application/%s/install-dev", am.currentApp.ID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to install debug application: %w", err)
	}

	log.Printf("Debug application installation initiated")

	// Wait for application to start and become healthy
	return am.waitForApplicationReady(ctx)
}

// waitForApplicationReady waits for the application to become ready
func (am *ApplicationManager) waitForApplicationReady(ctx context.Context) error {
	log.Printf("Waiting for application to become ready...")

	timeout := time.NewTimer(60 * time.Second)
	defer timeout.Stop()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for application to become ready")
		case <-ticker.C:
			status, err := am.GetApplicationStatus(ctx)
			if err != nil {
				log.Printf("Error checking application status: %v", err)
				continue
			}

			log.Printf("Application status: %s", status.Status)
			if status.HealthCheck != "" {
				log.Printf("Health check: %s", status.HealthCheck)
			}
			if status.Error != "" {
				log.Printf("Error: %s", status.Error)
			}

			// Check if application is ready
			if status.Status == "running" && status.HealthCheck == "healthy" {
				log.Printf("Application is ready!")
				if status.Port > 0 {
					log.Printf("Application listening on port: %d", status.Port)
				}
				return nil
			}

			// Check for failure states
			if status.Status == "stopped" {
				return fmt.Errorf("application failed to start: %s", status.Error)
			}
		}
	}
}

// GetApplicationStatus retrieves the current status of the debug application
func (am *ApplicationManager) GetApplicationStatus(ctx context.Context) (*ApplicationStatus, error) {
	if am.currentApp == nil {
		return nil, fmt.Errorf("no debug application to check status")
	}

	response, err := am.client.Get(ctx, fmt.Sprintf("/debug/application/%s/status", am.currentApp.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get application status: %w", err)
	}
	defer response.Body.Close()

	var status ApplicationStatus
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %w", err)
	}

	return &status, nil
}

// StopApplication stops the running debug application
func (am *ApplicationManager) StopApplication(ctx context.Context) error {
	if am.currentApp == nil {
		return nil
	}

	log.Printf("Stopping debug application: %s", am.currentApp.ID)

	_, err := am.client.Post(ctx, fmt.Sprintf("/debug/application/%s/stop", am.currentApp.ID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to stop debug application: %w", err)
	}

	log.Printf("Debug application stop request sent")

	// Wait for application to stop
	return am.waitForApplicationStopped(ctx)
}

// waitForApplicationStopped waits for the application to stop
func (am *ApplicationManager) waitForApplicationStopped(ctx context.Context) error {
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			log.Printf("Warning: timeout waiting for application to stop")
			return nil // Don't fail on stop timeout
		case <-ticker.C:
			status, err := am.GetApplicationStatus(ctx)
			if err != nil {
				// If we can't get status, assume it's stopped
				log.Printf("Application appears to be stopped (status check failed)")
				return nil
			}

			if status.Status == "stopped" || status.Status == "failed" {
				log.Printf("Application stopped successfully")
				return nil
			}
		}
	}
}

// GetCurrentApplication returns the current debug application
func (am *ApplicationManager) GetCurrentApplication() *DebugApplication {
	return am.currentApp
}

// IsApplicationRunning checks if the current application is running
func (am *ApplicationManager) IsApplicationRunning(ctx context.Context) bool {
	if am.currentApp == nil {
		return false
	}

	status, err := am.GetApplicationStatus(ctx)
	if err != nil {
		return false
	}

	return status.Status == "running"
}

// RestartApplication restarts the debug application (used for hot-reload)
func (am *ApplicationManager) RestartApplication(ctx context.Context) error {
	if am.currentApp == nil {
		return fmt.Errorf("no debug application to restart")
	}

	log.Printf("Restarting debug application for hot-reload...")

	// Stop the current instance
	if err := am.StopApplication(ctx); err != nil {
		log.Printf("Warning: failed to stop application during restart: %v", err)
	}

	// Reinstall with new package
	return am.InstallApplication(ctx)
}

// Cleanup performs final cleanup when exiting the CLI
func (am *ApplicationManager) Cleanup(ctx context.Context) error {
	if am.currentApp == nil {
		return nil
	}

	log.Printf("Performing final cleanup of debug application...")

	// Stop the application
	if err := am.StopApplication(ctx); err != nil {
		log.Printf("Warning: failed to stop application during cleanup: %v", err)
	}

	// Remove the application
	return am.CleanupExistingApplication(ctx)
}
