// Package handlers implements the NexusHub debug log streaming API endpoints for debug application log monitoring.
//
// This module provides REST API endpoints for streaming real-time logs from debug applications
// through WebSocket connections, allowing developers to monitor application output in real-time.
//
// Reference: spec/nexushub.md - Task nexushub-debug-logs
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	httpsproxy_types "github.com/tomyedwab/yesterday/nexushub/httpsproxy/types"
	"github.com/tomyedwab/yesterday/nexushub/processes"
)

// LogEntry represents a single log entry from a debug application
type LogEntry struct {
	ID        int64  `json:"id"`
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Source    string `json:"source"` // "stdout" or "stderr"
	Message   string `json:"message"`
	PID       int    `json:"pid,omitempty"`
}

// LogStreamClient represents a connected log streaming client using Server-Sent Events
type LogStreamClient struct {
	writer        http.ResponseWriter
	flusher       http.Flusher
	send          chan LogEntry
	done          chan struct{}
	applicationID string
	mu            sync.RWMutex
}

// LogStreamer manages log streaming for multiple applications
type LogStreamer struct {
	clients map[string]map[*LogStreamClient]bool // applicationID -> clients
	mu      sync.RWMutex
	logger  interface {
		Info(msg string, args ...interface{})
		Error(msg string, args ...interface{})
		Debug(msg string, args ...interface{})
	}
}

// NewLogStreamer creates a new log streaming manager
func NewLogStreamer(logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}) *LogStreamer {
	return &LogStreamer{
		clients: make(map[string]map[*LogStreamClient]bool),
		logger:  logger,
	}
}

// HandleLogStream handles GET /debug/application/{id}/logs for real-time log streaming
func (h *DebugHandler) HandleLogStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract application ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/debug/application/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "logs" {
		http.Error(w, "Invalid logs URL format", http.StatusBadRequest)
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

	// Check if application is running
	if debugApp.Status != "running" {
		http.Error(w, "Application is not running", http.StatusBadRequest)
		return
	}

	// Get the managed process for this application
	_, port, err := h.processManager.GetAppInstanceByID(appID)
	if err != nil {
		h.logger.Error("Failed to get app instance for log streaming", "appId", appID, "error", err)
		http.Error(w, "Application not found in process manager", http.StatusNotFound)
		return
	}

	h.logger.Info("Starting log stream for debug application", "appId", appID, "port", port)

	// Initialize log streamer if not already done
	if h.logStreamer == nil {
		h.logStreamer = NewLogStreamer(h.logger)
	}

	// Set up Server-Sent Events headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Check if the response writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Error("Response writer does not support flushing")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create log stream client
	client := &LogStreamClient{
		writer:        w,
		flusher:       flusher,
		send:          make(chan LogEntry, 256), // Buffer for log entries
		done:          make(chan struct{}),
		applicationID: appID,
	}

	// Register client with log streamer
	h.logStreamer.addClient(appID, client)

	// Start streaming logs from the ProcessManager
	go h.logStreamer.streamProcessLogs(client, appID, h.processManager)

	// Handle client connection (this will block until the client disconnects)
	h.logStreamer.handleClient(client, r.Context())
}

// addClient registers a new client for log streaming
func (ls *LogStreamer) addClient(appID string, client *LogStreamClient) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if ls.clients[appID] == nil {
		ls.clients[appID] = make(map[*LogStreamClient]bool)
	}
	ls.clients[appID][client] = true

	ls.logger.Info("Log stream client connected", "appId", appID, "totalClients", len(ls.clients[appID]))
}

// removeClient unregisters a client from log streaming
func (ls *LogStreamer) removeClient(appID string, client *LogStreamClient) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if clients, exists := ls.clients[appID]; exists {
		delete(clients, client)
		if len(clients) == 0 {
			delete(ls.clients, appID)
		}
	}

	ls.logger.Info("Log stream client disconnected", "appId", appID)
}

// streamProcessLogs streams logs from the ProcessManager to the client
func (ls *LogStreamer) streamProcessLogs(client *LogStreamClient, appID string, processManager interface{}) {
	defer func() {
		ls.removeClient(client.applicationID, client)
		close(client.send)
		close(client.done)
	}()

	// Cast to ProcessManagerInterface
	pm, ok := processManager.(httpsproxy_types.ProcessManagerInterface)
	if !ok {
		ls.logger.Error("ProcessManager does not support log streaming interface", "type", fmt.Sprintf("%T", processManager))
		return
	}

	ls.logger.Info("Starting log stream for application", "appId", client.applicationID)

	// Send initial connection message
	select {
	case client.send <- LogEntry{
		ID:        0,
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "info",
		Source:    "system",
		Message:   fmt.Sprintf("Log stream started for application %s", client.applicationID),
	}:
	case <-client.done:
		return
	}

	// Get recent logs and send them first
	recentLogs, err := pm.GetLatestProcessLogs(appID, 50) // Get last 50 log entries
	if err != nil {
		ls.logger.Error("Failed to get recent logs", "appId", appID, "error", err)
	} else {
		// Send recent logs
		for _, logEntry := range recentLogs {
			select {
			case client.send <- LogEntry{
				ID:        logEntry.ID,
				Timestamp: logEntry.Timestamp.Format(time.RFC3339),
				Level:     logEntry.Level,
				Source:    logEntry.Source,
				Message:   logEntry.Message,
				PID:       logEntry.PID,
			}:
			case <-client.done:
				return
			}
		}
	}

	// Get the latest log ID to poll from
	lastID, err := pm.GetProcessLogLatestID(appID)
	if err != nil {
		ls.logger.Error("Failed to get latest log ID", "appId", appID, "error", err)
		lastID = 0
	}

	// Set up a callback to receive new log entries
	pm.AddLogCallback(func(instanceID string, logEntry processes.ProcessLogEntry) {
		if instanceID != appID {
			return
		}

		// Send the new log entry to the client
		select {
		case client.send <- LogEntry{
			ID:        logEntry.ID,
			Timestamp: logEntry.Timestamp.Format(time.RFC3339),
			Level:     logEntry.Level,
			Source:    logEntry.Source,
			Message:   logEntry.Message,
			PID:       logEntry.PID,
		}:
		case <-client.done:
			return
		}
	})

	// Poll for new logs periodically as backup
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Poll for new logs since the last ID
			newLogs, err := pm.GetProcessLogs(appID, lastID)
			if err != nil {
				ls.logger.Error("Failed to poll for new logs", "appId", appID, "error", err)
				continue
			}

			// Send new logs
			for _, logEntry := range newLogs {
				select {
				case client.send <- LogEntry{
					ID:        logEntry.ID,
					Timestamp: logEntry.Timestamp.Format(time.RFC3339),
					Level:     logEntry.Level,
					Source:    logEntry.Source,
					Message:   logEntry.Message,
					PID:       logEntry.PID,
				}:
					lastID = logEntry.ID
				case <-client.done:
					return
				}
			}

		case <-client.done:
			ls.logger.Info("Log stream stopped", "appId", client.applicationID)
			return
		}
	}
}

// handleClient handles Server-Sent Events communication with a log streaming client
func (ls *LogStreamer) handleClient(client *LogStreamClient, ctx context.Context) {
	defer func() {
		ls.removeClient(client.applicationID, client)
	}()

	// Send keepalive messages to detect client disconnection
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case logEntry, ok := <-client.send:
			if !ok {
				// Channel closed, client is done
				return
			}

			// Convert log entry to JSON
			logData, err := json.Marshal(logEntry)
			if err != nil {
				ls.logger.Error("Failed to marshal log entry to JSON", "error", err)
				continue
			}

			// Send SSE event
			if err := ls.writeSSEEvent(client, "log", string(logData)); err != nil {
				ls.logger.Error("Failed to write SSE event", "error", err)
				return
			}

		case <-ticker.C:
			// Send keepalive event
			if err := ls.writeSSEEvent(client, "keepalive", ""); err != nil {
				ls.logger.Error("Failed to write keepalive event", "error", err)
				return
			}

		case <-ctx.Done():
			// Client disconnected
			ls.logger.Info("Client disconnected", "appId", client.applicationID)
			return

		case <-client.done:
			return
		}
	}
}

// writeSSEEvent writes a Server-Sent Event to the client
func (ls *LogStreamer) writeSSEEvent(client *LogStreamClient, eventType, data string) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	// Write event type
	if eventType != "" {
		if _, err := fmt.Fprintf(client.writer, "event: %s\n", eventType); err != nil {
			return err
		}
	}

	// Write data
	if data != "" {
		if _, err := fmt.Fprintf(client.writer, "data: %s\n", data); err != nil {
			return err
		}
	}

	// Write empty line to complete the event
	if _, err := fmt.Fprintf(client.writer, "\n"); err != nil {
		return err
	}

	// Flush the data
	client.flusher.Flush()
	return nil
}

// BroadcastLog sends a log entry to all clients streaming logs for the given application
func (ls *LogStreamer) BroadcastLog(appID string, logEntry LogEntry) {
	ls.mu.RLock()
	clients, exists := ls.clients[appID]
	ls.mu.RUnlock()

	if !exists {
		return
	}

	for client := range clients {
		select {
		case client.send <- logEntry:
		default:
			// Client's send channel is full, skip this log entry
			ls.logger.Debug("Skipping log entry for slow client", "appId", appID)
		}
	}
}

// GetLogStatus returns the current log streaming status for an application
func (h *DebugHandler) GetLogStatus(appID string) map[string]interface{} {
	if h.logStreamer == nil {
		return map[string]interface{}{
			"streaming": false,
			"clients":   0,
		}
	}

	h.logStreamer.mu.RLock()
	clients, exists := h.logStreamer.clients[appID]
	clientCount := 0
	if exists {
		clientCount = len(clients)
	}
	h.logStreamer.mu.RUnlock()

	return map[string]interface{}{
		"streaming": clientCount > 0,
		"clients":   clientCount,
	}
}
