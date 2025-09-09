package types

import (
	"github.com/tomyedwab/yesterday/nexushub/processes"
)

// ProcessManagerInterface defines the methods the HostnameResolver needs
// from the ProcessManager. This helps in decoupling and testing.
// It should provide a way to get an AppInstance by its hostname.
type ProcessManagerInterface interface {
	GetAppInstanceByHostName(hostname string) (*processes.AppInstance, int, error)
	GetAppInstanceByID(id string) (*processes.AppInstance, int, error) // Added for AppID lookup

	EventPublished()
	AddEventStateCallback() (string, chan processes.EventCallbackInfo)
	RemoveEventStateCallback(cbID string)
	GetEventState(id string) int

	// Log streaming methods for debug applications
	GetProcessLogs(instanceID string, fromID int64) ([]processes.ProcessLogEntry, error)
	GetLatestProcessLogs(instanceID string, count int) ([]processes.ProcessLogEntry, error)
	GetProcessLogLatestID(instanceID string) (int64, error)
	AddLogCallback(callback processes.LogCallback)

	// Trigger a run of the reconciler ASAP
	Refresh()
}

// AppInstanceProvider defines the methods the DebugHandler needs
// from the AdminInstanceProvider. This helps in decoupling and testing.
type AppInstanceProvider interface {
	AddDebugInstance(instance processes.AppInstance)
	RemoveDebugInstance(instanceID string)
}
