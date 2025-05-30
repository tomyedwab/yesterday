# Technical Product Specification: Go Process Manager

**Author:** Cascade AI
**Date:** 2025-05-29
**Status:** Proposed

## 1. Introduction

This document outlines the technical specification for a Go-based process manager. The primary responsibility of this manager is to oversee a collection of subprocesses, ensuring they are running as declared and remain healthy. It will dynamically manage subprocesses based on a declarative configuration, monitor their health via HTTP endpoints, and handle restarts or recovery actions as needed. Each subprocess is an instance of the `dist/bin/servicehost` executable, configured with specific WASM paths, database names, and dynamically assigned ports.

The process manager itself will reside in the `database/processes` directory.

## 2. Goals

*   **Declarative State Reconciliation:** Maintain a set of running subprocesses that matches a declarative list of application instance IDs and their associated metadata.
*   **Dynamic Configuration:** Respond to runtime changes in the declarative application list, starting or stopping subprocesses as required.
*   **Health Monitoring:** Actively monitor the health of each subprocess via a dedicated `/api/status` HTTP endpoint.
*   **Automatic Restarts:** Restart subprocesses that terminate unexpectedly or repeatedly fail health checks.
*   **Dynamic Port Allocation:** Assign unique, available ports to each subprocess instance.
*   **Robust Subprocess Execution:** Reliably launch and manage `servicehost` executables with appropriate command-line arguments derived from instance metadata.

## 3. Non-Goals

*   **Resource Management:** Advanced resource management (CPU/memory limits) for subprocesses is out of scope for the initial version.
*   **Complex Orchestration:** Features like blue/green deployments, canary releases, or inter-process dependency management are not included.
*   **External Configuration Storage:** The initial version will assume the declarative list is provided programmatically or via a simple local configuration mechanism. Integration with external configuration stores (e.g., etcd, Consul) is a future consideration.
*   **UI/Dashboard:** A dedicated user interface for managing or monitoring processes is not part of this specification.

## 4. High-Level Design

The process manager will operate as a standalone Go application or a component within a larger system. It will maintain two primary states:

1.  **Desired State:** A list of application instances (App IDs, WASM paths, DB names) that *should* be running. This list can be updated at runtime.
2.  **Actual State:** An internal representation of currently running subprocesses, their PIDs, assigned ports, health status, and other relevant metadata.

A central **Reconciler Loop** will continuously compare the desired state with the actual state:
*   If an app instance is in the desired state but not in the actual state (or not running), the manager will attempt to start it.
*   If a running subprocess is in the actual state but not in the desired state, the manager will stop it.

A separate **Health Monitor** component will periodically poll the `/api/status` endpoint of each running subprocess. If a subprocess is unhealthy or stops responding, the health monitor will flag it, potentially triggering a restart attempt by the Reconciler.

## 5. Detailed Design

### 5.1. Configuration Management

*   **App Instance Definition:** The process manager will expect a declarative list of application instances. Each instance will be defined by:
    *   `InstanceID` (string): A unique identifier for the application instance.
    *   `WasmPath` (string): The file system path to the WASM module for this instance.
    *   `DbName` (string): The database name/identifier to be used by this instance.
    *   Other relevant metadata (e.g., specific environment variables, restart policies).
*   **Runtime Updates:** A mechanism to update the declarative list of application instances will be created later. It is out of scope for the current work.

### 5.2. Process Lifecycle Management

*   **Starting Subprocesses:**
    1.  Identify an available port (see Port Management).
    2.  Construct the command: `dist/bin/servicehost -wasm <PATH-TO-WASM> -dbPath <DB-NAME> -port <PORT>`.
    3.  Execute the command as a subprocess (`os/exec` package).
    4.  Store the subprocess's PID, assigned port, and other metadata in the internal 'actual state' list.
    5.  Log the start event.
*   **Stopping Subprocesses:**
    1.  Identify the subprocess to stop using its `InstanceID`.
    2.  Send a graceful termination signal (e.g., `SIGTERM`).
    3.  Allow a configurable grace period for the subprocess to shut down.
    4.  If the subprocess does not terminate within the grace period, send a `SIGKILL` signal.
    5.  Remove the subprocess from the 'actual state' list.
    6.  Log the stop event.
*   **Restarting Subprocesses:**
    1.  Triggered by unexpected termination (detected by monitoring the `cmd.Wait()` channel) or repeated health check failures.
    2.  Implement a backoff strategy (e.g., exponential backoff) for restart attempts to avoid rapid-fire restarts of a persistently failing process.
    3.  Log restart attempts and failures.

### 5.3. Health Monitoring

*   **HTTP Health Check:** Each `servicehost` subprocess is expected to expose an `/api/status` HTTP endpoint.
    *   A `GET` request to this endpoint should return an HTTP `200 OK` status if the service is healthy.
    *   The response body may optionally contain structured health information (e.g., JSON), but the primary indicator is the status code.
*   **Monitoring Loop:**
    1.  Periodically (configurable interval, e.g., every 15-30 seconds) iterate through all running subprocesses.
    2.  For each subprocess, make an HTTP GET request to `http://localhost:<ASSIGNED_PORT>/api/status`.
    3.  Set a reasonable timeout for the HTTP request.
*   **Health State:**
    *   **Healthy:** HTTP `200 OK` received.
    *   **Unhealthy:** Non-200 status code, timeout, or connection error.
*   **Failure Threshold:** If a subprocess fails a configurable number of consecutive health checks (e.g., 3 failures), it is deemed persistently unhealthy, and the manager should trigger a restart.
*   **Process Exit Monitoring:** Independently of HTTP health checks, the manager will monitor if a subprocess exits using `cmd.Wait()`. An unexpected exit will also trigger a restart attempt.

### 5.4. Port Management

*   **Dynamic Allocation:** The process manager will be responsible for assigning a unique, available TCP port to each `servicehost` instance it starts.
*   **Port Range:** A configurable range of ports (e.g., 30000-31000) can be defined for allocation.
*   **Availability Check:** Before assigning a port, the manager will attempt to bind to it briefly to ensure it's not already in use. Alternatively, it can maintain a list of allocated ports.
*   **Port Release:** When a subprocess is stopped, its assigned port should be marked as available again.

### 5.5. Subprocess Execution

*   **Executable Path:** The manager will assume `dist/bin/servicehost` is a fixed, known path or configurable.
*   **Command-Line Arguments:** Arguments will be constructed as follows:
    *   `-wasm <PATH-TO-WASM>`: Derived from the `WasmPath` metadata of the app instance.
    *   `-dbPath <DB-NAME>`: Derived from the `DbName` metadata of the app instance.
    *   `-port <PORT>`: The dynamically assigned port.
*   **Standard Streams:** The manager should capture `stdout` and `stderr` from subprocesses for logging and debugging purposes.
*   **Working Directory:** The working directory for subprocesses should be configurable or default to a sensible location.

## 6. API (Internal)

While a public API for external control might not be in the initial scope, internal Go interfaces will be defined for:

*   `AppInstanceProvider`: An interface to get the current list of desired app instances.
*   `ProcessController`: An interface to start, stop, and query the status of subprocesses.
*   `HealthChecker`: An interface for performing health checks.

## 7. Error Handling and Resilience

*   **Manager Crashes:** If the process manager itself crashes, running subprocesses will continue to run (orphaned). Upon restart, the manager should attempt to reconcile its state, potentially by identifying existing `servicehost` processes (e.g., by scanning processes or looking for lock files with port information, though this can be complex).
*   **Startup Failures:** If a subprocess fails to start (e.g., `servicehost` executable not found, invalid WASM path), this should be logged, and the manager should not continuously retry without a backoff.
*   **Health Check Failures:** Handled by the restart mechanism with backoff.
*   **Port Conflicts:** The port allocation mechanism should prevent conflicts. If a conflict occurs unexpectedly, it should be logged, and an alternative port should be tried.

## 8. Logging and Metrics

*   **Structured Logging:** Use a structured logging library (e.g., `log/slog` in Go 1.21+, or a third-party library like `zerolog` or `zap`).
*   **Key Log Events:**
    *   Process manager startup and shutdown.
    *   Updates to the declarative app list.
    *   Subprocess start, stop, restart attempts (including PIDs and assigned ports).
    *   Health check results (especially failures).
    *   Errors encountered (e.g., port allocation failure, command execution failure).
    *   `stdout` and `stderr` from subprocesses.
*   **Metrics (Future Consideration):**
    *   Number of desired instances.
    *   Number of running instances.
    *   Number of unhealthy instances.
    *   Restart counts per instance.
    *   CPU/Memory usage (if feasible to collect from subprocesses).

## 9. Future Considerations

*   Integration with external configuration management systems (Consul, etcd).
*   More sophisticated health check mechanisms (e.g., custom checks beyond HTTP).
*   Resource limits for subprocesses.
*   Secure communication with subprocesses (e.g., mTLS for health checks if exposed externally).
*   A simple CLI or API for querying the manager's status and the status of managed processes.
*   Persistence of assigned ports and process state to better handle manager restarts.

---

This specification provides a foundational plan for the Go process manager. Details may be refined during implementation.
