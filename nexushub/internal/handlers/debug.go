// Package handlers implements the NexusHub debug API endpoints for debug application management.
//
// This module provides REST API endpoints for creating, managing, and monitoring debug applications
// as part of the NexusDebug development workflow.
//
// Reference: spec/nexushub.md - Task nexushub-debug-application
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	httpsproxy_types "github.com/tomyedwab/yesterday/nexushub/httpsproxy/types"
)

// DebugApplicationRequest represents the request payload for creating debug applications
type DebugApplicationRequest struct {
	AppID            string `json:"appId"`
	DisplayName      string `json:"displayName"`
	HostName         string `json:"hostName"`
	DbName           string `json:"dbName"`
	StaticServiceURL string `json:"staticServiceUrl,omitempty"`
}

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

// DebugHandler handles debug application lifecycle management
type DebugHandler struct {
	processManager httpsproxy_types.ProcessManagerInterface
	logger         *slog.Logger
	debugApps      map[string]*DebugApplication // In-memory storage for debug apps
	internalSecret string
}

// NewDebugHandler creates a new debug handler instance
func NewDebugHandler(processManager httpsproxy_types.ProcessManagerInterface, logger *slog.Logger, internalSecret string) *DebugHandler {
	return &DebugHandler{
		processManager: processManager,
		logger:         logger,
		debugApps:      make(map[string]*DebugApplication),
		internalSecret: internalSecret,
	}
}

// HandleCreateApplication handles POST /debug/application for creating debug applications
func (h *DebugHandler) HandleCreateApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req DebugApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse debug application request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.AppID == "" || req.DisplayName == "" || req.HostName == "" || req.DbName == "" {
		h.logger.Error("Missing required fields in debug application request", 
			"appId", req.AppID, "displayName", req.DisplayName, "hostName", req.HostName, "dbName", req.DbName)
		http.Error(w, "Missing required fields: appId, displayName, hostName, dbName", http.StatusBadRequest)
		return
	}

	// Check for existing debug application with same AppID and clean it up
	if err := h.cleanupExistingApplication(req.AppID); err != nil {
		h.logger.Warn("Failed to cleanup existing debug application", "appId", req.AppID, "error", err)
		// Continue with creation - don't fail on cleanup errors
	}

	// Generate unique ID for the debug application
	appID := uuid.New().String()

	// Create debug application
	debugApp := &DebugApplication{
		ID:               appID,
		AppID:            req.AppID,
		DisplayName:      req.DisplayName,
		HostName:         req.HostName,
		DbName:           req.DbName,
		StaticServiceURL: req.StaticServiceURL,
		Status:           "pending", // Applications start in pending state until installed
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}

	// Store in memory (in production, this would be stored in a database)
	h.debugApps[appID] = debugApp

	h.logger.Info("Debug application created", 
		"id", appID, 
		"appId", req.AppID, 
		"displayName", req.DisplayName, 
		"hostName", req.HostName,
		"staticServiceUrl", req.StaticServiceURL)

	// Return the created application
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(debugApp); err != nil {
		h.logger.Error("Failed to encode debug application response", "error", err)
	}
}

// HandleDeleteApplication handles DELETE /debug/application/{id} for removing debug applications
func (h *DebugHandler) HandleDeleteApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract application ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/debug/application/")
	appID := path

	if appID == "" {
		http.Error(w, "Missing application ID", http.StatusBadRequest)
		return
	}

	// Find the debug application
	debugApp, exists := h.debugApps[appID]
	if !exists {
		http.Error(w, "Debug application not found", http.StatusNotFound)
		return
	}

	// Stop the application if it's running (through process manager)
	if debugApp.Status == "running" {
		if err := h.stopDebugApplication(debugApp); err != nil {
			h.logger.Warn("Failed to stop debug application during deletion", "id", appID, "error", err)
			// Continue with deletion even if stop fails
		}
	}

	// Remove from storage
	delete(h.debugApps, appID)

	h.logger.Info("Debug application deleted", "id", appID, "appId", debugApp.AppID)

	w.WriteHeader(http.StatusNoContent)
}

// cleanupExistingApplication removes existing debug applications with the same AppID
func (h *DebugHandler) cleanupExistingApplication(appID string) error {
	for id, app := range h.debugApps {
		if app.AppID == appID {
			h.logger.Info("Cleaning up existing debug application", "id", id, "appId", appID)
			
			// Stop if running
			if app.Status == "running" {
				if err := h.stopDebugApplication(app); err != nil {
					h.logger.Warn("Failed to stop existing debug application", "id", id, "error", err)
				}
			}
			
			// Remove from storage
			delete(h.debugApps, id)
		}
	}
	return nil
}

// stopDebugApplication stops a running debug application via the process manager
func (h *DebugHandler) stopDebugApplication(app *DebugApplication) error {
	// In a real implementation, this would interact with the process manager
	// to stop the running instance. For now, we'll just update the status.
	
	h.logger.Info("Stopping debug application", "id", app.ID, "appId", app.AppID)
	
	// Update status to stopped
	app.Status = "stopped"
	
	return nil
}

// GetDebugApplication retrieves a debug application by ID
func (h *DebugHandler) GetDebugApplication(appID string) (*DebugApplication, bool) {
	app, exists := h.debugApps[appID]
	return app, exists
}

// ListDebugApplications returns all debug applications
func (h *DebugHandler) ListDebugApplications() []*DebugApplication {
	apps := make([]*DebugApplication, 0, len(h.debugApps))
	for _, app := range h.debugApps {
		apps = append(apps, app)
	}
	return apps
}

// ValidateDebugApplicationRequest validates the debug application request
func ValidateDebugApplicationRequest(req *DebugApplicationRequest) error {
	if req.AppID == "" {
		return fmt.Errorf("appId is required")
	}
	if req.DisplayName == "" {
		return fmt.Errorf("displayName is required")
	}
	if req.HostName == "" {
		return fmt.Errorf("hostName is required")
	}
	if req.DbName == "" {
		return fmt.Errorf("dbName is required")
	}
	
	// Validate hostname format
	if !strings.Contains(req.HostName, ".") {
		return fmt.Errorf("hostName must be a valid hostname format")
	}
	
	// Validate static service URL if provided
	if req.StaticServiceURL != "" {
		if !strings.HasPrefix(req.StaticServiceURL, "http://") && !strings.HasPrefix(req.StaticServiceURL, "https://") {
			return fmt.Errorf("staticServiceUrl must be a valid HTTP(S) URL")
		}
	}
	
	return nil
}
