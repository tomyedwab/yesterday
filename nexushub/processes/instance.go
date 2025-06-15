package processes

// AppInstance defines the desired state of an application instance.
// It includes all necessary information to launch and manage a servicehost subprocess.
type AppInstance struct {
	InstanceID string // Unique identifier for the application instance.
	HostName   string // Hostname for reverse proxy routing.
	BinPath    string // File system path to the binary for this instance.
	StaticPath string // File system path to static files for this instance.
	DbName     string // Database name/identifier to be used by this instance.
	DebugPort  int    // If set and vite is running on this port, the proxy will forward requests to it.
}
