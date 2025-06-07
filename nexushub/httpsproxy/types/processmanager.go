package types

import "github.com/tomyedwab/yesterday/nexushub/processes"

// ProcessManagerInterface defines the methods the HostnameResolver needs
// from the ProcessManager. This helps in decoupling and testing.
// It should provide a way to get an AppInstance by its hostname.
type ProcessManagerInterface interface {
	GetAppInstanceByHostName(hostname string) (*processes.AppInstance, int, error)
	GetAppInstanceByID(id string) (*processes.AppInstance, int, error) // Added for AppID lookup
}
