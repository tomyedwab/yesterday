// Package handlers implements the NexusHub debug API endpoints for debug application management.
//
// This module provides REST API endpoints for creating, managing, and monitoring debug applications
// as part of the NexusDebug development workflow.
//
// Reference: spec/nexushub.md - Task nexushub-debug-application
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	PackagePath      string `json:"-"` // Path to uploaded package (not exposed via JSON)
}

// UploadChunk represents a single chunk of an uploaded file
type UploadChunk struct {
	ChunkIndex int    `json:"chunkIndex"`
	Data       []byte `json:"-"`
	Received   bool   `json:"received"`
}

// UploadSession represents an ongoing upload session
type UploadSession struct {
	ApplicationID string                  `json:"applicationId"`
	TotalChunks   int                    `json:"totalChunks"`
	Chunks        map[int]*UploadChunk   `json:"chunks"`
	FileHash      string                 `json:"fileHash"`
	Completed     bool                   `json:"completed"`
	CreatedAt     time.Time              `json:"createdAt"`
	mu            sync.RWMutex           `json:"-"`
}

// UploadStatus represents the status of an upload session
type UploadStatus struct {
	ApplicationID   string  `json:"applicationId"`
	TotalChunks     int     `json:"totalChunks"`
	ReceivedChunks  int     `json:"receivedChunks"`
	Progress        float64 `json:"progress"`
	Completed       bool    `json:"completed"`
	FileHash        string  `json:"fileHash,omitempty"`
	Error           string  `json:"error,omitempty"`
}

// DebugHandler handles debug application lifecycle management
type DebugHandler struct {
	processManager   httpsproxy_types.ProcessManagerInterface
	logger           *slog.Logger
	debugApps        map[string]*DebugApplication // In-memory storage for debug apps
	uploadSessions   map[string]*UploadSession    // In-memory storage for upload sessions
	cleanupCancels   map[string]context.CancelFunc // Cleanup timer cancellation functions
	uploadDir        string                       // Directory for storing uploaded packages
	internalSecret   string
	mu               sync.RWMutex                 // Protects debugApps, uploadSessions, and cleanupCancels
}

// NewDebugHandler creates a new debug handler instance
func NewDebugHandler(processManager httpsproxy_types.ProcessManagerInterface, logger *slog.Logger, internalSecret string) *DebugHandler {
	// Create upload directory
	uploadDir := filepath.Join(os.TempDir(), "nexushub-debug-uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		logger.Warn("Failed to create upload directory", "dir", uploadDir, "error", err)
		uploadDir = os.TempDir() // Fallback to temp dir
	}

	return &DebugHandler{
		processManager:   processManager,
		logger:           logger,
		debugApps:        make(map[string]*DebugApplication),
		uploadSessions:   make(map[string]*UploadSession),
		cleanupCancels:   make(map[string]context.CancelFunc),
		uploadDir:        uploadDir,
		internalSecret:   internalSecret,
	}
}

// HandleUpload handles POST /debug/application/{id}/upload for chunked file uploads
func (h *DebugHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract application ID from URL path: /debug/application/{id}/upload
	path := strings.TrimPrefix(r.URL.Path, "/debug/application/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "upload" {
		http.Error(w, "Invalid upload URL format", http.StatusBadRequest)
		return
	}
	appID := parts[0]

	if appID == "" {
		http.Error(w, "Missing application ID", http.StatusBadRequest)
		return
	}

	// Check if debug application exists
	h.mu.RLock()
	debugApp, exists := h.debugApps[appID]
	h.mu.RUnlock()

	if !exists {
		h.logger.Error("Debug application not found for upload", "id", appID)
		http.Error(w, "Debug application not found", http.StatusNotFound)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		h.logger.Error("Failed to parse multipart form", "error", err)
		http.Error(w, "Failed to parse upload form", http.StatusBadRequest)
		return
	}

	// Extract chunk metadata
	chunkIndexStr := r.FormValue("chunkIndex")
	totalChunksStr := r.FormValue("totalChunks")
	fileHash := r.FormValue("fileHash")

	if chunkIndexStr == "" || totalChunksStr == "" || fileHash == "" {
		h.logger.Error("Missing required upload parameters", 
			"chunkIndex", chunkIndexStr, "totalChunks", totalChunksStr, "fileHash", fileHash)
		http.Error(w, "Missing required parameters: chunkIndex, totalChunks, fileHash", http.StatusBadRequest)
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		h.logger.Error("Invalid chunk index", "chunkIndex", chunkIndexStr, "error", err)
		http.Error(w, "Invalid chunk index", http.StatusBadRequest)
		return
	}

	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil {
		h.logger.Error("Invalid total chunks", "totalChunks", totalChunksStr, "error", err)
		http.Error(w, "Invalid total chunks", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, _, err := r.FormFile("chunk")
	if err != nil {
		h.logger.Error("Failed to get chunk file from form", "error", err)
		http.Error(w, "Failed to get chunk file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read chunk data
	chunkData, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read chunk data", "error", err)
		http.Error(w, "Failed to read chunk data", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Received chunk upload", 
		"appId", appID, "chunkIndex", chunkIndex, "totalChunks", totalChunks, 
		"chunkSize", len(chunkData), "fileHash", fileHash)

	// Process the chunk
	if err := h.processUploadChunk(appID, chunkIndex, totalChunks, fileHash, chunkData); err != nil {
		h.logger.Error("Failed to process upload chunk", "error", err)
		http.Error(w, "Failed to process chunk", http.StatusInternalServerError)
		return
	}

	// Check if upload is complete
	h.mu.RLock()
	uploadSession, exists := h.uploadSessions[appID]
	h.mu.RUnlock()

	if exists && uploadSession.Completed {
		h.logger.Info("Upload completed", "appId", appID, "packagePath", debugApp.PackagePath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "completed",
			"message": "Package upload completed successfully",
		})
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "received",
			"message": "Chunk received successfully",
		})
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
	h.mu.Lock()
	h.debugApps[appID] = debugApp
	h.mu.Unlock()

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

	// Cancel any pending cleanup timer
	if cancel, exists := h.cleanupCancels[appID]; exists {
		cancel()
		delete(h.cleanupCancels, appID)
	}

	// Remove from storage
	delete(h.debugApps, appID)
	delete(h.uploadSessions, appID)

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
			
			// Cancel any pending cleanup timer
			if cancel, exists := h.cleanupCancels[id]; exists {
				cancel()
				delete(h.cleanupCancels, id)
			}
			
			// Remove from storage
			delete(h.debugApps, id)
			delete(h.uploadSessions, id)
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
