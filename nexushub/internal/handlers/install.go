// Package handlers implements the NexusHub debug install API endpoints for debug application installation.
//
// This module provides REST API endpoints for installing uploaded debug applications using the
// existing package manager and integrating with the process manager for application lifecycle.
//
// Reference: spec/nexushub.md - Task nexushub-debug-install
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tomyedwab/yesterday/nexushub/packages"
	"github.com/tomyedwab/yesterday/nexushub/processes"
)

// InstallRequest represents the request payload for installing debug applications
type InstallRequest struct {
	// No additional parameters needed for basic installation
}

// InstallResponse represents the response from the install endpoint
type InstallResponse struct {
	ApplicationID string `json:"applicationId"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	InstalledAt   string `json:"installedAt"`
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

// HandleInstallApplication handles POST /debug/application/{id}/install-dev for application deployment
func (h *DebugHandler) HandleInstallDevApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract application ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/debug/application/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "install" {
		http.Error(w, "Invalid install URL format", http.StatusBadRequest)
		return
	}
	appID := parts[0]

	if appID == "" {
		http.Error(w, "Missing application ID", http.StatusBadRequest)
		return
	}

	h.logger.Info("Installing debug application", "appId", appID)

	// Find the debug application
	h.mu.RLock()
	debugApp, exists := h.debugApps[appID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Debug application not found", http.StatusNotFound)
		return
	}

	// Check if package has been uploaded
	if debugApp.PackagePath == "" {
		http.Error(w, "No package uploaded for this application", http.StatusBadRequest)
		return
	}

	// Verify package file exists
	if _, err := os.Stat(debugApp.PackagePath); os.IsNotExist(err) {
		http.Error(w, "Package file not found", http.StatusNotFound)
		return
	}

	// Stop existing instance if running
	if debugApp.Status == "running" || debugApp.Status == "installed" {
		h.logger.Info("Stopping existing instance before reinstall", "appId", appID)
		if err := h.stopDebugApplicationInstance(debugApp); err != nil {
			h.logger.Warn("Failed to stop existing instance", "appId", appID, "error", err)
			// Continue with installation even if stop fails
		}
	}

	// Install the package using the package manager
	packageManager := packages.NewPackageManager()

	// Copy package to package manager directory with expected naming
	if err := h.preparePackageForInstallation(debugApp, packageManager); err != nil {
		h.logger.Error("Failed to prepare package for installation", "appId", appID, "error", err)
		http.Error(w, fmt.Sprintf("Package preparation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Install the package
	if err := packageManager.InstallPackage(debugApp.AppID, appID); err != nil {
		h.logger.Error("Package installation failed", "appId", appID, "error", err)
		http.Error(w, fmt.Sprintf("Installation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Update application status
	h.mu.Lock()
	debugApp.Status = "installed"
	h.mu.Unlock()

	h.logger.Info("Debug application installed successfully", "appId", appID)

	// Start the application via process manager
	if err := h.startDebugApplicationInstance(debugApp, packageManager); err != nil {
		h.logger.Error("Failed to start debug application", "appId", appID, "error", err)

		// Update status to failed but still return success for the installation
		h.mu.Lock()
		debugApp.Status = "failed"
		h.mu.Unlock()

		// Return partial success - installed but not started
		response := InstallResponse{
			ApplicationID: appID,
			Status:        "installed_not_started",
			Message:       fmt.Sprintf("Application installed but failed to start: %v", err),
			InstalledAt:   time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update status to running
	h.mu.Lock()
	debugApp.Status = "running"
	h.mu.Unlock()

	// Schedule cleanup after 1 hour of inactivity
	go h.scheduleApplicationCleanup(appID)

	// Return success response
	response := InstallResponse{
		ApplicationID: appID,
		Status:        "running",
		Message:       "Application installed and started successfully",
		InstalledAt:   time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode install response", "error", err)
	}
}

// HandleApplicationStatus handles GET /debug/application/{id}/status for application health monitoring
func (h *DebugHandler) HandleApplicationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract application ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/debug/application/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "status" {
		http.Error(w, "Invalid status URL format", http.StatusBadRequest)
		return
	}
	appID := parts[0]

	if appID == "" {
		http.Error(w, "Missing application ID", http.StatusBadRequest)
		return
	}

	// Find the debug application
	h.mu.RLock()
	debugApp, exists := h.debugApps[appID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Debug application not found", http.StatusNotFound)
		return
	}

	// Reset cleanup timer since status was checked
	go h.scheduleApplicationCleanup(appID)

	// Get status from process manager if running
	status := ApplicationStatus{
		ApplicationID: appID,
		Status:        debugApp.Status,
		LastUpdated:   time.Now().Format(time.RFC3339),
		Metadata: map[string]interface{}{
			"appId":            debugApp.AppID,
			"displayName":      debugApp.DisplayName,
			"hostName":         debugApp.HostName,
			"staticServiceUrl": debugApp.StaticServiceURL,
		},
	}

	// If application is supposed to be running, check with process manager
	if debugApp.Status == "running" {
		if appInstance, port, err := h.processManager.GetAppInstanceByID(appID, nil); err == nil && appInstance != nil {
			// Note: GetAppInstanceByID returns the AppInstance config, but we need ManagedProcess for runtime info
			// For now, we'll use the returned port and indicate the app is running
			status.Port = port
			status.HealthCheck = "healthy" // Process manager only returns running instances
			status.ProcessID = 0           // ProcessID not available from this interface
		} else {
			// Application should be running but not found in process manager
			status.Status = "pending"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Error("Failed to encode status response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// preparePackageForInstallation copies the uploaded package to the expected location for package manager
func (h *DebugHandler) preparePackageForInstallation(debugApp *DebugApplication, packageManager *packages.PackageManager) error {
	// Get PKG_DIR from package manager (via environment)
	pkgDir := os.Getenv("PKG_DIR")
	if pkgDir == "" {
		pkgDir = "/usr/local/etc/nexushub/packages"
	}

	// Ensure PKG_DIR exists
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return fmt.Errorf("failed to create PKG_DIR: %w", err)
	}

	// Expected package filename format
	expectedPackageName := fmt.Sprintf("%s.zip", debugApp.AppID)
	expectedPackagePath := filepath.Join(pkgDir, expectedPackageName)

	// Copy the uploaded package to the expected location
	sourceFile, err := os.Open(debugApp.PackagePath)
	if err != nil {
		return fmt.Errorf("failed to open source package: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(expectedPackagePath)
	if err != nil {
		return fmt.Errorf("failed to create destination package: %w", err)
	}
	defer destFile.Close()

	// Copy file contents
	_, err = destFile.ReadFrom(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy package: %w", err)
	}

	h.logger.Info("Package prepared for installation",
		"appId", debugApp.AppID,
		"source", debugApp.PackagePath,
		"destination", expectedPackagePath)

	return nil
}

// startDebugApplicationInstance starts the debug application via process manager
func (h *DebugHandler) startDebugApplicationInstance(debugApp *DebugApplication, packageManager *packages.PackageManager) error {
	// Create app instance configuration for the process manager
	installDir := packageManager.GetInstallDir()
	if installDir == "" {
		installDir = "/usr/local/etc/nexushub/install"
	}

	appInstancePath := filepath.Join(installDir, debugApp.ID)
	appBinaryPath := filepath.Join(appInstancePath, "app", "bin", "app")

	// Verify the binary exists
	if _, err := os.Stat(appBinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("application binary not found at %s", appBinaryPath)
	}

	// The process manager integration would happen here
	// For now, we'll simulate starting the application
	h.logger.Info("Starting debug application instance",
		"appId", debugApp.ID,
		"binaryPath", appBinaryPath,
		"installPath", appInstancePath)

	// Create an AppInstance for the process manager
	appInstance := processes.AppInstance{
		InstanceID: debugApp.ID,
		HostName:   debugApp.HostName,
		PkgPath:    appInstancePath,
	}

	// Add the instance to the provider
	h.instanceProvider.AddDebugInstance(appInstance)

	return nil
}

// stopDebugApplicationInstance stops a running debug application via process manager
func (h *DebugHandler) stopDebugApplicationInstance(debugApp *DebugApplication) error {
	h.logger.Info("Stopping debug application instance", "appId", debugApp.ID)

	// Get the app instance from process manager
	if appInstance, _, err := h.processManager.GetAppInstanceByID(debugApp.ID, nil); err == nil && appInstance != nil {
		// Stop the instance (this would be the actual implementation)
		h.logger.Info("Found running instance, stopping", "appId", debugApp.ID)
		// The process manager would handle stopping the process
		h.instanceProvider.RemoveDebugInstance(debugApp.ID)
	}

	// Update status
	debugApp.Status = "stopped"

	return nil
}

// scheduleApplicationCleanup schedules automatic cleanup after 1 hour of inactivity
// This function cancels any existing cleanup timer for the application and starts a new one
func (h *DebugHandler) scheduleApplicationCleanup(appID string) {
	h.mu.Lock()

	// Cancel existing cleanup timer if one exists
	if existingCancel, exists := h.cleanupCancels[appID]; exists {
		existingCancel()
		h.logger.Debug("Cancelled existing cleanup timer", "appId", appID)
	}

	// Create new context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	h.cleanupCancels[appID] = cancel

	h.mu.Unlock()

	// Start cleanup timer in a goroutine
	go func() {
		// Wait for 1 hour or cancellation
		select {
		case <-time.After(1 * time.Hour):
			// Timer expired, proceed with cleanup
			h.performApplicationCleanup(appID)
		case <-ctx.Done():
			// Timer was cancelled, do nothing
			h.logger.Debug("Cleanup timer cancelled", "appId", appID)
			return
		}
	}()
}

// performApplicationCleanup performs the actual cleanup of an application
func (h *DebugHandler) performApplicationCleanup(appID string) {
	h.logger.Info("Auto-cleaning up inactive debug application", "appId", appID)

	// Check if application still exists
	h.mu.RLock()
	debugApp, exists := h.debugApps[appID]
	h.mu.RUnlock()

	if !exists {
		h.logger.Debug("Application already cleaned up", "appId", appID)
		return
	}

	// Stop the application
	if debugApp.Status == "running" {
		h.stopDebugApplicationInstance(debugApp)
	}

	// Remove from debug apps and cleanup timers
	h.mu.Lock()
	delete(h.debugApps, appID)
	delete(h.cleanupCancels, appID)
	delete(h.uploadSessions, appID)
	h.mu.Unlock()

	// Clean up uploaded package file
	if debugApp.PackagePath != "" {
		if err := os.Remove(debugApp.PackagePath); err != nil {
			h.logger.Warn("Failed to remove package file during cleanup", "path", debugApp.PackagePath, "error", err)
		}
	}

	h.logger.Info("Debug application cleaned up", "appId", appID)
}
