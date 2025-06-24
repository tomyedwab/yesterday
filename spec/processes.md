# Technical Product Specification: Go Process Manager

**Reference Design Document:** [design/processes.md](../design/processes.md)

## Introduction

This specification defines a Go-based process manager that orchestrates the lifecycle of multiple servicehost subprocesses. The system reconciles a declarative list of application instances with running processes, monitors their health via HTTP endpoints, and handles automatic restarts and recovery. The process manager integrates with an admin service for dynamic configuration and supports static app overrides for local development.

The implementation resides in `nexushub/processes/` and provides robust process lifecycle management with features including dynamic port allocation, health monitoring, exponential restart backoff, and graceful shutdown capabilities.

## Task `processes-core-manager`: Process Manager Core Logic
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/manager.go` (892 lines), `nexushub/processes/process.go` (119 lines)

**Details:**
- Implement `ProcessManager` struct with reconciliation loop comparing desired vs actual state
- Implement `ManagedProcess` wrapper for subprocess state tracking with thread-safe state transitions
- Define `ProcessState` enum: `StateUnknown`, `StateStarting`, `StateRunning`, `StateUnhealthy`, `StateStopping`, `StateStopped`, `StateFailed`
- Implement graceful shutdown with SIGTERM/SIGKILL progression and configurable timeout (default 10s)
- Subprocess execution: `dist/github.com/tomyedwab/yesterday/nexushub/bin/krunclient <BinPath> <Port>`
- Environment variables: `HOST=<HostName>`, `INTERNAL_SECRET=<secret>`
- Capture stdout/stderr for logging and debugging
- First reconcile completion tracking with callback support for startup coordination

## Task `processes-instance-structure`: AppInstance Definition  
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/instance.go` (13 lines)

**Details:**
- Define `AppInstance` struct with fields:
  - `InstanceID string`: Unique identifier for the application instance
  - `HostName string`: Hostname for reverse proxy routing
  - `BinPath string`: File system path to the binary for this instance
  - `StaticPath string`: File system path to static files for this instance  
  - `DbName string`: Database name/identifier (currently unused)
  - `DebugPort int`: If set and Vite is running, proxy forwards requests to it

## Task `processes-port-manager`: Dynamic Port Allocation
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/port_manager.go` (84 lines)

**Details:**
- Implement `PortManager` struct with configurable port range (e.g., 30000-31000)
- `AllocatePort()` method with round-robin allocation and availability verification by attempting to bind
- `ReleasePort()` method for cleanup when processes terminate
- Thread-safe allocation tracking with mutex protection
- Handle port exhaustion with clear error messages

## Task `processes-health-checker`: HTTP Health Monitoring
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/health.go` (63 lines)

**Details:**
- Define `HealthChecker` interface with `Check(process *ManagedProcess) (ProcessState, error)` method
- Implement `HTTPHealthChecker` targeting `http://localhost:<PORT>/api/status`
- Configurable timeouts (default 5s per request) and intervals (default 15s)
- Health state mapping: HTTP 200 → `StateRunning`, errors/timeouts → `StateUnhealthy`, non-200 → `StateUnhealthy`
- Consecutive failure threshold (default 3) triggers restart via `StateFailed` transition

## Task `processes-instance-provider-static`: Static App Configuration
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/manager.go` (`SimpleAppInstanceProvider`)

**Details:**
- Implement `AppInstanceProvider` interface with `GetAppInstances(ctx context.Context) ([]AppInstance, error)`
- `SimpleAppInstanceProvider` for testing and simple scenarios with fixed list
- `UpdateAppInstances()` method for runtime configuration changes
- Thread-safe access with RWMutex protection

## Task `processes-instance-provider-admin`: Admin Service Integration
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/instance_provider.go` (336 lines)

**Details:**
- Implement `AdminInstanceProvider` fetching from admin service API
- Application fetching: `GET /api/applications` with `Application` struct mapping
- Change detection: `GET /api/poll?e=<eventID>` every 5 seconds with event ID tracking
- Authentication via `X-Internal-Secret` header for cross-service requests
- Automatic retry with exponential backoff on failures (max 5 retries, 2s initial delay)
- Graceful fallback to static apps when admin service unavailable
- Polling frequency backoff on consecutive errors (max 10 errors → 3x interval)

## Task `processes-static-overrides`: Static App Configuration Overrides  
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/instance_provider.go` (`StaticAppConfig`, `convertToAppInstance`)

**Details:**
- Define `StaticAppConfig` struct for development workflow overrides
- Override admin service data by matching `InstanceID`
- Support custom `BinPath`, `StaticPath`, `HostName`, `DebugPort` for local development
- Applied in `convertToAppInstance()` during admin service data processing

## Task `processes-host-lookup`: Instance Lookup for Proxy Integration
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/manager.go` (`GetAppInstanceByHostName`, `GetAppInstanceByID`)

**Details:**
- Implement `GetAppInstanceByHostName(hostname string) (*AppInstance, int, error)` for HTTPS proxy routing
- Implement `GetAppInstanceByID(id string) (*AppInstance, int, error)` for direct instance lookup  
- Return running and healthy instances only with their dynamically assigned ports
- Thread-safe access to actual state map with RWMutex protection

## Task `processes-restart-backoff`: Exponential Restart Backoff
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/manager.go` (`calculateBackoff`)

**Details:**
- Implement exponential backoff for restart attempts: initial 1s, doubling up to 30s maximum
- Restart count tracking per process for debugging and metrics
- Backoff calculation: `initialDelay * 2^(restartCount-1)` capped at `maxDelay`
- Reset restart count on successful health check after being unhealthy/failed

## Task `processes-first-reconcile-callback`: Startup Coordination
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/manager.go` (`SetFirstReconcileCompleteCallback`, `IsFirstReconcileComplete`)

**Details:**
- Track first reconcile completion when all desired processes are running and healthy
- `SetFirstReconcileCompleteCallback(callback func())` for startup coordination
- `IsFirstReconcileComplete() bool` for health check and status endpoints
- Thread-safe callback management with mutex protection
- One-time callback execution in separate goroutine to avoid blocking reconciliation

## Task `processes-graceful-shutdown`: Process Termination Handling
**Reference:** design/processes.md  
**Implementation status:** Completed  
**Files:** `nexushub/processes/manager.go` (`stopProcess`, `shutdown`)

**Details:**
- Graceful termination: send SIGTERM, wait for configurable period (default 10s), then SIGKILL
- Process cleanup: release allocated ports, remove from actual state map
- Manager shutdown: stop all managed processes in parallel with timeout handling
- Proper cleanup of goroutines and resources during shutdown sequence
