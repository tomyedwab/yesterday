// Package nexusdebug implements tests for application monitoring functionality.
//
// This module provides unit tests for the Monitor struct and its methods,
// including log formatting, status monitoring, and reconnection logic.
//
// Reference: spec/nexusdebug.md - Task nexusdebug-monitoring
package nexusdebug

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestNewMonitor tests the creation of a new Monitor instance
func TestNewMonitor(t *testing.T) {
	app := &DebugApplication{
		ID:          "test-app-id",
		AppID:       "debug-test-app",
		DisplayName: "Debug: Test App",
		HostName:    "test.debug",
	}

	monitor := NewMonitor(nil, app)

	if monitor == nil {
		t.Fatal("Expected non-nil monitor")
	}

	if monitor.app != app {
		t.Errorf("Expected monitor.app to be %v, got %v", app, monitor.app)
	}

	if monitor.stopChan == nil {
		t.Error("Expected non-nil stopChan")
	}

	if monitor.statusChan == nil {
		t.Error("Expected non-nil statusChan")
	}

	if monitor.logChan == nil {
		t.Error("Expected non-nil logChan")
	}
}

// TestFormatLogEntry tests log entry formatting
func TestFormatLogEntry(t *testing.T) {
	tests := []struct {
		name     string
		entry    *LogEntry
		expected string
	}{
		{
			name: "Basic log entry",
			entry: &LogEntry{
				Timestamp: "2024-01-01T12:00:00Z",
				Level:     "info",
				Message:   "Test message",
			},
			expected: "12:00:00 🔵 INFO  Test message",
		},
		{
			name: "Error log entry with source",
			entry: &LogEntry{
				Timestamp: "2024-01-01T12:00:00Z",
				Level:     "error",
				Message:   "Error occurred",
				Source:    "main.go",
			},
			expected: "12:00:00 🔴 ERROR [main.go] Error occurred",
		},
		{
			name: "Debug log entry with process ID",
			entry: &LogEntry{
				Timestamp: "2024-01-01T12:00:00Z",
				Level:     "debug",
				Message:   "Debug info",
				ProcessID: 1234,
			},
			expected: "12:00:00 🟢 DEBUG (PID 1234) Debug info",
		},
		{
			name: "Warning log entry with source and process ID",
			entry: &LogEntry{
				Timestamp: "2024-01-01T12:00:00Z",
				Level:     "warn",
				Message:   "Warning message",
				Source:    "server.go",
				ProcessID: 5678,
			},
			expected: "12:00:00 🟡 WARN  [server.go] (PID 5678) Warning message",
		},
		{
			name: "Custom log level",
			entry: &LogEntry{
				Timestamp: "2024-01-01T12:00:00Z",
				Level:     "custom",
				Message:   "Custom message",
			},
			expected: "12:00:00      CUSTOM Custom message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLogEntry(tt.entry)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestFormatLogLevel tests log level formatting
func TestFormatLogLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected string
	}{
		{"error", "🔴 ERROR"},
		{"ERROR", "🔴 ERROR"},
		{"warn", "🟡 WARN "},
		{"warning", "🟡 WARN "},
		{"info", "🔵 INFO "},
		{"INFO", "🔵 INFO "},
		{"debug", "🟢 DEBUG"},
		{"DEBUG", "🟢 DEBUG"},
		{"custom", "     CUSTOM"},
		{"trace", "     TRACE"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := formatLogLevel(tt.level)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestFormatStatusUpdate tests status update formatting
func TestFormatStatusUpdate(t *testing.T) {
	tests := []struct {
		name     string
		status   *ApplicationStatus
		contains []string
	}{
		{
			name: "Basic status",
			status: &ApplicationStatus{
				Status:      "running",
				LastUpdated: "2024-01-01T12:00:00Z",
			},
			contains: []string{"🟢 Status: running", "12:00:00"},
		},
		{
			name: "Status with process and port",
			status: &ApplicationStatus{
				Status:      "running",
				ProcessID:   1234,
				Port:        8080,
				LastUpdated: "2024-01-01T12:00:00Z",
			},
			contains: []string{"🟢 Status: running", "PID: 1234", "Port: 8080"},
		},
		{
			name: "Status with health check",
			status: &ApplicationStatus{
				Status:      "running",
				HealthCheck: "healthy",
				LastUpdated: "2024-01-01T12:00:00Z",
			},
			contains: []string{"🟢 Status: running", "Health: ✅ healthy"},
		},
		{
			name: "Failed status with error",
			status: &ApplicationStatus{
				Status:      "failed",
				Error:       "Connection refused",
				LastUpdated: "2024-01-01T12:00:00Z",
			},
			contains: []string{"💥 Status: failed", "Error: Connection refused"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatStatusUpdate(tt.status)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, got %q", expected, result)
				}
			}
		})
	}
}

// TestGetStatusIcon tests status icon selection
func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"running", "🟢"},
		{"RUNNING", "🟢"},
		{"starting", "🟡"},
		{"stopped", "🔴"},
		{"failed", "💥"},
		{"unknown", "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := getStatusIcon(tt.status)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGetHealthIcon tests health icon selection
func TestGetHealthIcon(t *testing.T) {
	tests := []struct {
		health   string
		expected string
	}{
		{"healthy", "✅"},
		{"HEALTHY", "✅"},
		{"unhealthy", "❌"},
		{"starting", "🔄"},
		{"unknown", "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.health, func(t *testing.T) {
			result := getHealthIcon(tt.health)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestMonitorChannels tests that Monitor channels are properly initialized
func TestMonitorChannels(t *testing.T) {
	app := &DebugApplication{ID: "test-app"}
	monitor := NewMonitor(nil, app)

	// Test that channels are available
	select {
	case <-monitor.GetLogChannel():
		t.Error("Should not receive from empty log channel")
	default:
		// Expected behavior
	}

	select {
	case <-monitor.GetStatusChannel():
		t.Error("Should not receive from empty status channel")
	default:
		// Expected behavior
	}
}

// TestStopMonitoring tests that monitoring can be stopped
func TestStopMonitoring(t *testing.T) {
	app := &DebugApplication{ID: "test-app"}
	monitor := NewMonitor(nil, app)

	// Stop monitoring should not panic
	monitor.StopMonitoring()

	// Verify stop channel is closed
	select {
	case <-monitor.stopChan:
		// Expected behavior - channel should be closed
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected stop channel to be closed")
	}
}

// TestMonitorWithNilApp tests behavior when app is nil
func TestMonitorWithNilApp(t *testing.T) {
	monitor := NewMonitor(nil, nil)

	ctx := context.Background()
	err := monitor.StartMonitoring(ctx)

	if err == nil {
		t.Error("Expected error when starting monitoring with nil app")
	}

	if !strings.Contains(err.Error(), "no debug application to monitor") {
		t.Errorf("Expected 'no debug application to monitor' error, got %v", err)
	}
}

// BenchmarkFormatLogEntry benchmarks log entry formatting
func BenchmarkFormatLogEntry(b *testing.B) {
	entry := &LogEntry{
		Timestamp: "2024-01-01T12:00:00Z",
		Level:     "info",
		Message:   "Test message for benchmarking",
		Source:    "main.go",
		ProcessID: 1234,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatLogEntry(entry)
	}
}

// BenchmarkFormatStatusUpdate benchmarks status update formatting
func BenchmarkFormatStatusUpdate(b *testing.B) {
	status := &ApplicationStatus{
		Status:      "running",
		ProcessID:   1234,
		Port:        8080,
		HealthCheck: "healthy",
		LastUpdated: "2024-01-01T12:00:00Z",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatStatusUpdate(status)
	}
}
