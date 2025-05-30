package processes

import (
	"os/exec"
	"sync"
	"time"
)

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
