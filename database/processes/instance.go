package processes

// AppInstance defines the desired state of an application instance.
// It includes all necessary information to launch and manage a servicehost subprocess.
type AppInstance struct {
	InstanceID string // Unique identifier for the application instance.
	WasmPath   string // File system path to the WASM module for this instance.
	DbName     string // Database name/identifier to be used by this instance.
}
