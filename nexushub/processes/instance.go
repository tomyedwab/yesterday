package processes

// AppInstance defines the desired state of an application instance.
// It includes all necessary information to launch and manage a servicehost subprocess.
type AppInstance struct {
	InstanceID    string // Unique identifier for the application instance.
	HostName      string // Hostname for reverse proxy routing.
	PkgPath       string // File system path to the binary for this instance.
	Subscriptions map[string]bool
}
