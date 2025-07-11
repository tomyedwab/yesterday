package processes

import (
	"os/exec"
	"sync"
	"time"
)

// ProcessLogEntry represents a single log entry from a managed process
type ProcessLogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Source    string    `json:"source"` // "stdout" or "stderr"
	Message   string    `json:"message"`
	PID       int       `json:"pid"`
}

// LogBuffer maintains a circular buffer of recent log entries
type LogBuffer struct {
	mu        sync.RWMutex
	entries   []ProcessLogEntry
	capacity  int
	nextID    int64
	callbacks []func(ProcessLogEntry)
}

// NewLogBuffer creates a new log buffer with the specified capacity
func NewLogBuffer(capacity int) *LogBuffer {
	return &LogBuffer{
		entries:  make([]ProcessLogEntry, 0, capacity),
		capacity: capacity,
		nextID:   1,
	}
}

// AddEntry adds a new log entry to the buffer
func (lb *LogBuffer) AddEntry(level, source, message string, pid int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	entry := ProcessLogEntry{
		ID:        lb.nextID,
		Timestamp: time.Now(),
		Level:     level,
		Source:    source,
		Message:   message,
		PID:       pid,
	}

	// Add to buffer (circular buffer behavior)
	if len(lb.entries) >= lb.capacity {
		// Remove oldest entry
		lb.entries = lb.entries[1:]
	}
	lb.entries = append(lb.entries, entry)
	lb.nextID++

	// Notify callbacks
	for _, callback := range lb.callbacks {
		go callback(entry) // Run in goroutine to avoid blocking
	}
}

// GetEntriesFromID returns all log entries with ID greater than the specified ID
func (lb *LogBuffer) GetEntriesFromID(fromID int64) []ProcessLogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]ProcessLogEntry, 0)
	for _, entry := range lb.entries {
		if entry.ID > fromID {
			result = append(result, entry)
		}
	}
	return result
}

// GetLatestEntries returns the most recent N log entries
func (lb *LogBuffer) GetLatestEntries(count int) []ProcessLogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if count <= 0 || len(lb.entries) == 0 {
		return []ProcessLogEntry{}
	}

	start := len(lb.entries) - count
	if start < 0 {
		start = 0
	}

	result := make([]ProcessLogEntry, len(lb.entries)-start)
	copy(result, lb.entries[start:])
	return result
}

// AddCallback adds a callback function to be called when new log entries are added
func (lb *LogBuffer) AddCallback(callback func(ProcessLogEntry)) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.callbacks = append(lb.callbacks, callback)
}

// GetLatestID returns the ID of the most recent log entry
func (lb *LogBuffer) GetLatestID() int64 {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.entries) == 0 {
		return 0
	}
	return lb.entries[len(lb.entries)-1].ID
}

// ProcessState represents the health status of a managed process.
type ProcessState int

const (
	// StateUnknown means the process state is not yet determined.
	StateUnknown ProcessState = iota
	// StateStarting means the process is in the process of being started.
	StateStarting
	// StateRunning means the process is running and healthy.
	StateRunning
	// StateUnhealthy means the process is running but failing health checks.
	StateUnhealthy
	// StateStopping means the process is in the process of being stopped.
	StateStopping
	// StateStopped means the process has been stopped.
	StateStopped
	// StateFailed means the process failed to start or crashed.
	StateFailed
)

// String returns a string representation of the ProcessState.
func (ps ProcessState) String() string {
	switch ps {
	case StateUnknown:
		return "Unknown"
	case StateStarting:
		return "Starting"
	case StateRunning:
		return "Running"
	case StateUnhealthy:
		return "Unhealthy"
	case StateStopping:
		return "Stopping"
	case StateStopped:
		return "Stopped"
	case StateFailed:
		return "Failed"
	default:
		return "InvalidState"
	}
}

// ManagedProcess represents a subprocess that is being managed by the process manager.
// It holds information about the desired AppInstance, the actual running os/exec.Cmd, and its current state.
type ManagedProcess struct {
	Instance AppInstance    // The desired configuration for this process.
	Cmd      *exec.Cmd      // The running command.
	Port     int            // The TCP port assigned to this process.
	PID      int            // Process ID of the running subprocess.
	State    ProcessState   // Current health/lifecycle state of the process.
	LogBuffer *LogBuffer    // Buffer for storing recent log entries from this process.

	mu             sync.Mutex // Protects access to this struct's mutable fields.
	startTime      time.Time  // Time when the process was last started.
	lastHealthCh   time.Time  // Time of the last successful health check.
	unhealthySince time.Time  // Time when the process first became unhealthy.
	restartCount   int        // Number of times this process has been restarted.
}

// NewManagedProcess creates a new ManagedProcess instance.
func NewManagedProcess(instance AppInstance, cmd *exec.Cmd, port int) *ManagedProcess {
	return &ManagedProcess{
		Instance:  instance,
		Cmd:       cmd,
		Port:      port,
		PID:       cmd.Process.Pid, // Assumes cmd.Process is not nil (i.e., process started)
		State:     StateStarting,   // Initial state after starting
		LogBuffer: NewLogBuffer(1000), // Keep last 1000 log entries
		startTime: time.Now(),
	}
}

// UpdateState sets the process state thread-safely.
func (mp *ManagedProcess) UpdateState(newState ProcessState) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.State = newState

	switch newState {
	case StateRunning:
		mp.lastHealthCh = time.Now()
		mp.unhealthySince = time.Time{} // Reset unhealthy timer
	case StateUnhealthy:
		if mp.unhealthySince.IsZero() {
			mp.unhealthySince = time.Now()
		}
	case StateFailed, StateStopped:
		mp.Cmd = nil // Clear the command as it's no longer running
	}
}

// RecordRestart increments the restart count.
func (mp *ManagedProcess) RecordRestart() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.restartCount++
	mp.startTime = time.Now()
}

// GetRestartCount returns the current restart count.
func (mp *ManagedProcess) GetRestartCount() int {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.restartCount
}

// GetState returns the current process state thread-safely.
func (mp *ManagedProcess) GetState() ProcessState {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.State
}
