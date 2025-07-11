// Package nexusdebug implements application status and log monitoring for the NexusDebug CLI tool.
//
// This module provides real-time log tailing and application status monitoring
// via NexusHub's debug API endpoints with formatting and reconnection capabilities.
//
// Reference: spec/nexusdebug.md - Task nexusdebug-monitoring
package nexusdebug

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

// LogEntry represents a single log entry from the debug application
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Source    string `json:"source,omitempty"`
	ProcessID int    `json:"processId,omitempty"`
}

// Monitor handles real-time log tailing and application status monitoring
type Monitor struct {
	client     *yesterdaygo.Client
	app        *DebugApplication
	logStream  io.ReadCloser
	stopChan   chan struct{}
	statusChan chan *ApplicationStatus
	logChan    chan *LogEntry
}

// NewMonitor creates a new monitoring instance for the given debug application
func NewMonitor(client *yesterdaygo.Client, app *DebugApplication) *Monitor {
	return &Monitor{
		client:     client,
		app:        app,
		stopChan:   make(chan struct{}),
		statusChan: make(chan *ApplicationStatus, 10),
		logChan:    make(chan *LogEntry, 100),
	}
}

// StartMonitoring begins monitoring the debug application logs and status
func (m *Monitor) StartMonitoring(ctx context.Context) error {
	if m.app == nil {
		return fmt.Errorf("no debug application to monitor")
	}

	log.Printf("Starting monitoring for debug application: %s", m.app.ID)

	// Start log tailing in a separate goroutine
	go m.startLogTailing(ctx)

	// Start status monitoring in a separate goroutine
	go m.startStatusMonitoring(ctx)

	return nil
}

// StopMonitoring stops all monitoring activities
func (m *Monitor) StopMonitoring() {
	close(m.stopChan)
	
	if m.logStream != nil {
		m.logStream.Close()
	}
}

// GetLogChannel returns the channel for receiving log entries
func (m *Monitor) GetLogChannel() <-chan *LogEntry {
	return m.logChan
}

// GetStatusChannel returns the channel for receiving status updates
func (m *Monitor) GetStatusChannel() <-chan *ApplicationStatus {
	return m.statusChan
}

// startLogTailing initiates real-time log tailing with reconnection logic
func (m *Monitor) startLogTailing(ctx context.Context) {
	retryDelay := 2 * time.Second
	maxRetryDelay := 30 * time.Second
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		default:
			// Attempt to connect to log stream
			if err := m.connectToLogStream(ctx); err != nil {
				log.Printf("Failed to connect to log stream: %v", err)
				log.Printf("Retrying in %v...", retryDelay)
				
				// Wait before retry
				select {
				case <-time.After(retryDelay):
					// Exponential backoff with max delay
					retryDelay *= 2
					if retryDelay > maxRetryDelay {
						retryDelay = maxRetryDelay
					}
				case <-ctx.Done():
					return
				case <-m.stopChan:
					return
				}
				continue
			}
			
			// Reset retry delay on successful connection
			retryDelay = 2 * time.Second
			
			// Read from log stream
			m.readLogStream(ctx)
			
			// If we reach here, the stream was closed
			log.Printf("Log stream closed, attempting to reconnect...")
		}
	}
}

// connectToLogStream establishes a connection to the log streaming endpoint
func (m *Monitor) connectToLogStream(ctx context.Context) error {
	endpoint := fmt.Sprintf("/debug/application/%s/logs", m.app.ID)
	
	// Create streaming request
	response, err := m.client.Get(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to log stream: %w", err)
	}
	
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return fmt.Errorf("log stream returned status %d", response.StatusCode)
	}
	
	m.logStream = response.Body
	log.Printf("Connected to log stream for application: %s", m.app.ID)
	return nil
}

// readLogStream reads and processes log entries from the stream
func (m *Monitor) readLogStream(ctx context.Context) {
	if m.logStream == nil {
		return
	}
	
	defer func() {
		m.logStream.Close()
		m.logStream = nil
	}()
	
	scanner := bufio.NewScanner(m.logStream)
	
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		default:
			line := scanner.Text()
			if line == "" {
				continue
			}
			
			// Try to parse as JSON log entry
			var logEntry LogEntry
			if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
				// If not JSON, treat as plain text log
				logEntry = LogEntry{
					Timestamp: time.Now().Format(time.RFC3339),
					Level:     "info",
					Message:   line,
				}
			}
			
			// Send log entry to channel
			select {
			case m.logChan <- &logEntry:
			case <-ctx.Done():
				return
			case <-m.stopChan:
				return
			default:
				// Drop log entry if channel is full
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading log stream: %v", err)
	}
}

// startStatusMonitoring polls application status at regular intervals
func (m *Monitor) startStatusMonitoring(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			status, err := m.getApplicationStatus(ctx)
			if err != nil {
				log.Printf("Error getting application status: %v", err)
				continue
			}
			
			// Send status update to channel
			select {
			case m.statusChan <- status:
			case <-ctx.Done():
				return
			case <-m.stopChan:
				return
			default:
				// Drop status update if channel is full
			}
		}
	}
}

// getApplicationStatus retrieves the current status of the debug application
func (m *Monitor) getApplicationStatus(ctx context.Context) (*ApplicationStatus, error) {
	endpoint := fmt.Sprintf("/debug/application/%s/status", m.app.ID)
	
	response, err := m.client.Get(ctx, endpoint, nil)
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

// FormatLogEntry formats a log entry for display with timestamps and colors
func FormatLogEntry(entry *LogEntry) string {
	// Parse timestamp for better formatting
	timestamp := entry.Timestamp
	if t, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
		timestamp = t.Format("15:04:05")
	}
	
	// Format level with color coding
	level := formatLogLevel(entry.Level)
	
	// Include source if available
	source := ""
	if entry.Source != "" {
		source = fmt.Sprintf("[%s] ", entry.Source)
	}
	
	// Include process ID if available
	processInfo := ""
	if entry.ProcessID > 0 {
		processInfo = fmt.Sprintf("(PID %d) ", entry.ProcessID)
	}
	
	return fmt.Sprintf("%s %s %s%s%s", timestamp, level, source, processInfo, entry.Message)
}

// formatLogLevel applies color coding to log levels
func formatLogLevel(level string) string {
	switch strings.ToLower(level) {
	case "error":
		return "🔴 ERROR"
	case "warn", "warning":
		return "🟡 WARN "
	case "info":
		return "🔵 INFO "
	case "debug":
		return "🟢 DEBUG"
	default:
		return fmt.Sprintf("     %s", strings.ToUpper(level))
	}
}

// FormatStatusUpdate formats an application status update for display
func FormatStatusUpdate(status *ApplicationStatus) string {
	statusIcon := getStatusIcon(status.Status)
	
	var parts []string
	parts = append(parts, fmt.Sprintf("%s Status: %s", statusIcon, status.Status))
	
	if status.ProcessID > 0 {
		parts = append(parts, fmt.Sprintf("PID: %d", status.ProcessID))
	}
	
	if status.Port > 0 {
		parts = append(parts, fmt.Sprintf("Port: %d", status.Port))
	}
	
	if status.HealthCheck != "" {
		healthIcon := getHealthIcon(status.HealthCheck)
		parts = append(parts, fmt.Sprintf("Health: %s %s", healthIcon, status.HealthCheck))
	}
	
	if status.Error != "" {
		parts = append(parts, fmt.Sprintf("Error: %s", status.Error))
	}
	
	timestamp := status.LastUpdated
	if t, err := time.Parse(time.RFC3339, status.LastUpdated); err == nil {
		timestamp = t.Format("15:04:05")
	}
	
	return fmt.Sprintf("[%s] %s", timestamp, strings.Join(parts, " | "))
}

// getStatusIcon returns an appropriate icon for the application status
func getStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "running":
		return "🟢"
	case "starting":
		return "🟡"
	case "stopped":
		return "🔴"
	case "failed":
		return "💥"
	default:
		return "❓"
	}
}

// getHealthIcon returns an appropriate icon for the health check status
func getHealthIcon(health string) string {
	switch strings.ToLower(health) {
	case "healthy":
		return "✅"
	case "unhealthy":
		return "❌"
	case "starting":
		return "🔄"
	default:
		return "❓"
	}
}

// DisplayLogs continuously displays log entries from the monitor
func (m *Monitor) DisplayLogs(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case logEntry := <-m.logChan:
			fmt.Println(FormatLogEntry(logEntry))
		case status := <-m.statusChan:
			fmt.Println(FormatStatusUpdate(status))
		}
	}
}