package processes

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	defaultHealthCheckInterval    = 15 * time.Second
	defaultHealthCheckTimeout     = 5 * time.Second
	defaultConsecutiveFailures    = 3
	defaultRestartBackoffInitial  = 1 * time.Second
	defaultRestartBackoffMax      = 30 * time.Second
	defaultGracefulShutdownPeriod = 10 * time.Second
	serviceHostExecutable         = "dist/bin/servicehost" // As per spec, can be made configurable
)

// AppInstanceProvider defines an interface to get the current list of desired app instances.
// This allows the source of desired state to be flexible (e.g., in-memory, config file, API).
type AppInstanceProvider interface {
	GetAppInstances(ctx context.Context) ([]AppInstance, error)
}

// ProcessManager orchestrates the lifecycle of multiple servicehost subprocesses.
// It ensures that the actual running processes match a desired state, monitors their health,
// and handles restarts or recovery actions.	type ProcessManager struct {
// and handles restarts or recovery actions.
type ProcessManager struct {
	mu sync.RWMutex

	desiredStateProvider AppInstanceProvider
	actualState          map[string]*ManagedProcess // Keyed by InstanceID

	portManager   *PortManager
	healthChecker HealthChecker
	logger        *slog.Logger

	// Configuration
	healthCheckInterval    time.Duration
	consecutiveFailures    int           // Number of consecutive health check failures before restart
	restartBackoffInitial  time.Duration // Initial delay for restart backoff
	restartBackoffMax      time.Duration // Maximum delay for restart backoff
	gracefulShutdownPeriod time.Duration // Time to wait for graceful shutdown before SIGKILL
	internalSecret         string        // Secret for authorizing cross-service requests

	// Control channels
	stopChan chan struct{}  // Signals the manager to stop
	wg       sync.WaitGroup // Waits for goroutines to finish

	// Working directory for subprocesses
	subprocessWorkDir string

	// First reconciliation completion callback management
	onFirstReconcileComplete func()     // Callback function to fire on first successful reconcile
	firstReconcileComplete   bool       // Flag to track if first reconcile has completed
	callbackMu               sync.Mutex // Protects callback-related fields
}

// Config holds configuration options for the ProcessManager.
type Config struct {
	AppProvider            AppInstanceProvider
	PortManager            *PortManager
	HealthChecker          HealthChecker // Optional, defaults to HTTPHealthChecker
	Logger                 *slog.Logger  // Optional, defaults to slog.Default()
	HealthCheckInterval    time.Duration // Optional, defaults to 15s
	HealthCheckTimeout     time.Duration // Optional, for default HTTPHealthChecker, defaults to 5s
	ConsecutiveFailures    int           // Optional, defaults to 3
	RestartBackoffInitial  time.Duration // Optional, defaults to 1s
	RestartBackoffMax      time.Duration // Optional, defaults to 30s
	GracefulShutdownPeriod time.Duration // Optional, defaults to 10s
	SubprocessWorkDir      string        // Optional, defaults to current directory
	// OnFirstReconcileComplete is an optional callback function that will be called exactly once
	// when the first reconciliation process completes successfully with all desired processes
	// running and healthy. This is useful for:
	// - Signaling to other parts of your system that startup is complete
	// - Starting additional services that depend on these processes
	// - Updating health check endpoints to show "ready"
	// - Sending notifications that the system is operational
	// The callback is executed in a separate goroutine to avoid blocking the reconciliation process.
	OnFirstReconcileComplete func()
}

// NewProcessManager creates a new ProcessManager instance.
func NewProcessManager(config Config, internalSecret string) (*ProcessManager, error) {
	if config.AppProvider == nil {
		return nil, fmt.Errorf("AppInstanceProvider is required")
	}
	if config.PortManager == nil {
		return nil, fmt.Errorf("PortManager is required")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	healthChecker := config.HealthChecker
	if healthChecker == nil {
		hcTimeout := config.HealthCheckTimeout
		if hcTimeout == 0 {
			hcTimeout = defaultHealthCheckTimeout
		}
		healthChecker = NewHTTPHealthChecker(hcTimeout)
	}

	hcInterval := config.HealthCheckInterval
	if hcInterval == 0 {
		hcInterval = defaultHealthCheckInterval
	}
	consFailures := config.ConsecutiveFailures
	if consFailures == 0 {
		consFailures = defaultConsecutiveFailures
	}
	restartInitial := config.RestartBackoffInitial
	if restartInitial == 0 {
		restartInitial = defaultRestartBackoffInitial
	}
	restartMax := config.RestartBackoffMax
	if restartMax == 0 {
		restartMax = defaultRestartBackoffMax
	}
	gracefulShutdown := config.GracefulShutdownPeriod
	if gracefulShutdown == 0 {
		gracefulShutdown = defaultGracefulShutdownPeriod
	}

	workDir := config.SubprocessWorkDir
	if workDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		workDir = wd
	}

	pm := &ProcessManager{
		desiredStateProvider:     config.AppProvider,
		actualState:              make(map[string]*ManagedProcess),
		portManager:              config.PortManager,
		healthChecker:            healthChecker,
		logger:                   logger.With("component", "ProcessManager"),
		healthCheckInterval:      hcInterval,
		consecutiveFailures:      consFailures,
		restartBackoffInitial:    restartInitial,
		restartBackoffMax:        restartMax,
		gracefulShutdownPeriod:   gracefulShutdown,
		stopChan:                 make(chan struct{}),
		subprocessWorkDir:        workDir,
		internalSecret:           internalSecret,
		onFirstReconcileComplete: config.OnFirstReconcileComplete,
	}

	return pm, nil
}

// SetFirstReconcileCompleteCallback registers a callback function that will be called
// exactly once when the first reconciliation process completes successfully with all
// desired processes running and healthy.
//
// This is useful for scenarios where you need to know when all managed processes are
// fully operational, such as:
//   - Marking the application as "ready" for traffic
//   - Starting dependent services
//   - Sending startup completion notifications
//   - Updating monitoring systems
//
// The callback is executed in a separate goroutine to prevent blocking the reconciliation
// process. If the callback panics, the panic is recovered and logged as an error.
//
// This method is thread-safe and can be called at any time. If the first reconciliation
// has already completed, the callback will not be called. To check if the first
// reconciliation has completed, use IsFirstReconcileComplete().
//
// Parameters:
//   - callback: The function to call when first reconciliation completes. Can be nil to clear the callback.
func (pm *ProcessManager) SetFirstReconcileCompleteCallback(callback func()) {
	pm.callbackMu.Lock()
	defer pm.callbackMu.Unlock()

	// Only set the callback if first reconcile hasn't completed yet
	if !pm.firstReconcileComplete {
		pm.onFirstReconcileComplete = callback
	}
}

// IsFirstReconcileComplete returns true if the first reconciliation has completed
// successfully with all desired processes running and healthy.
//
// A successful first reconciliation means that:
//   - All desired processes have been started
//   - All processes are in the StateRunning state (running and healthy)
//   - The reconciliation process has completed at least once
//
// This method is thread-safe and can be called at any time to check the reconciliation status.
// It's particularly useful for health checks or startup coordination logic.
func (pm *ProcessManager) IsFirstReconcileComplete() bool {
	pm.callbackMu.Lock()
	defer pm.callbackMu.Unlock()
	return pm.firstReconcileComplete
}

// Run starts the process manager's reconciliation and health monitoring loops.
// It blocks until Stop() is called or the context is cancelled.
func (pm *ProcessManager) Run(ctx context.Context) {
	pm.logger.Info("ProcessManager starting...")
	pm.wg.Add(2) // For reconciler and health monitor goroutines

	go pm.reconcilerLoop(ctx)
	go pm.healthMonitorLoop(ctx)

	pm.logger.Info("ProcessManager running.")
	// Wait for stop signal or context cancellation
	select {
	case <-pm.stopChan:
		pm.logger.Info("ProcessManager received stop signal.")
	case <-ctx.Done():
		pm.logger.Info("ProcessManager context cancelled.")
	}

	pm.shutdown(context.Background()) // Use a new context for shutdown tasks
}

// Stop gracefully shuts down the process manager and all managed subprocesses.
func (pm *ProcessManager) Stop() {
	pm.logger.Info("Stopping ProcessManager...")
	close(pm.stopChan)
	pm.wg.Wait()
	pm.logger.Info("ProcessManager stopped.")
}

// GetAppInstanceByHostName searches for a running and healthy AppInstance by its HostName.
// It returns a copy of the AppInstance (including its dynamically assigned port)
// if found, otherwise returns nil and an error.
// This method is thread-safe.
// GetAppInstanceByHostName searches for a running and healthy AppInstance by its HostName.
// It returns a copy of the AppInstance (including its dynamically assigned port)
// if found, otherwise returns nil and an error.
// This method is thread-safe.
func (pm *ProcessManager) GetAppInstanceByHostName(hostname string) (*AppInstance, int, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, process := range pm.actualState {
		if process.Instance.HostName == hostname {
			if process.GetState() == StateRunning { // Ensure the instance is actually running and healthy
				// Return a copy to prevent modification by the caller
				instanceCopy := process.Instance
				return &instanceCopy, process.Port, nil
			}
			pm.logger.Warn("Found instance by hostname but it's not running", "hostname", hostname, "instanceID", process.Instance.InstanceID, "state", process.GetState().String())
			return nil, 0, fmt.Errorf("instance for hostname '%s' found but not in a running state (current state: %s)", hostname, process.GetState().String())
		}
	}

	return nil, 0, fmt.Errorf("no active and running instance found for hostname: %s", hostname)
}

// GetAppInstanceByID searches for a running and healthy AppInstance by its InstanceID.
// It returns a copy of the AppInstance (including its dynamically assigned port)
// if found, otherwise returns nil and an error.
// This method is thread-safe.
func (pm *ProcessManager) GetAppInstanceByID(id string) (*AppInstance, int, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	process, exists := pm.actualState[id]
	if !exists {
		return nil, 0, fmt.Errorf("no instance found with ID: %s", id)
	}

	if process.GetState() == StateRunning { // Ensure the instance is actually running and healthy
		// Return a copy to prevent modification by the caller
		instanceCopy := process.Instance
		return &instanceCopy, process.Port, nil
	}

	pm.logger.Warn("Found instance by ID but it's not running", "instanceID", id, "state", process.GetState().String())
	return nil, 0, fmt.Errorf("instance with ID '%s' found but not in a running state (current state: %s)", id, process.GetState().String())
}

// shutdown handles the graceful termination of all managed subprocesses.
func (pm *ProcessManager) shutdown(ctx context.Context) {
	pm.logger.Info("Shutting down all managed processes...")
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var shutdownWg sync.WaitGroup
	for instanceID, process := range pm.actualState {
		if process.GetState() == StateRunning || process.GetState() == StateStarting || process.GetState() == StateUnhealthy {
			shutdownWg.Add(1)
			go func(id string, proc *ManagedProcess) {
				defer shutdownWg.Done()
				pm.logger.Info("Stopping process", "instanceID", id, "pid", proc.PID)
				if err := pm.stopProcess(ctx, proc, true); err != nil {
					pm.logger.Error("Error stopping process during shutdown", "instanceID", id, "error", err)
				}
			}(instanceID, process)
		}
	}
	shutdownWg.Wait()
	pm.logger.Info("All managed processes have been instructed to stop.")
}

// reconcilerLoop is the main loop that compares desired state with actual state and takes action.
func (pm *ProcessManager) reconcilerLoop(ctx context.Context) {
	defer pm.wg.Done()
	pm.logger.Info("Reconciler loop started.")

	// Initial reconciliation
	if err := pm.reconcileState(ctx); err != nil {
		pm.logger.Error("Initial reconciliation failed", "error", err)
		// Depending on the error, might need to retry or exit
	}

	// TODO: The spec mentions "Respond to runtime changes in the declarative application list".
	// This implies the reconciler should periodically re-fetch desired state or have a mechanism
	// to be notified of changes. For now, it reconciles once and then relies on health checks
	// and process exit monitoring to trigger actions.
	// A simple ticker can be added here to periodically call reconcileState if AppInstanceProvider
	// can change over time.

	ticker := time.NewTicker(pm.healthCheckInterval) // Re-evaluate desired state at similar frequency to health checks
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopChan:
			pm.logger.Info("Reconciler loop stopping.")
			return
		case <-ctx.Done():
			pm.logger.Info("Reconciler loop context cancelled.")
			return
		case <-ticker.C:
			if err := pm.reconcileState(ctx); err != nil {
				pm.logger.Error("Reconciliation failed", "error", err)
			}
		}
	}
}

// healthMonitorLoop periodically checks the health of all running subprocesses.
func (pm *ProcessManager) healthMonitorLoop(ctx context.Context) {
	defer pm.wg.Done()
	pm.logger.Info("Health monitor loop started.")

	ticker := time.NewTicker(pm.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopChan:
			pm.logger.Info("Health monitor loop stopping.")
			return
		case <-ctx.Done():
			pm.logger.Info("Health monitor loop context cancelled.")
			return
		case <-ticker.C:
			pm.performHealthChecks(ctx)
		}
	}
}

// reconcileState compares the desired application instances with the actual running processes
// and takes actions to start missing processes or stop unwanted ones.
func (pm *ProcessManager) checkAndFireFirstReconcileCallback() {
	pm.callbackMu.Lock()
	defer pm.callbackMu.Unlock()

	// If we've already fired the callback, nothing to do
	if pm.firstReconcileComplete {
		return
	}

	// Check if all processes are running and healthy
	desiredInstances, err := pm.desiredStateProvider.GetAppInstances(context.Background())
	if err != nil {
		// Can't determine desired state, so we can't say reconciliation is complete
		return
	}

	// Check that we have all desired processes running
	if len(pm.actualState) != len(desiredInstances) {
		return
	}

	// Check that all processes are healthy
	for _, instance := range desiredInstances {
		managedProcess, exists := pm.actualState[instance.InstanceID]
		if !exists {
			return // Process doesn't exist
		}

		if managedProcess.State != StateRunning {
			return // Process is not running and healthy
		}
	}

	// All processes are running and healthy - fire the callback!
	pm.firstReconcileComplete = true
	if pm.onFirstReconcileComplete != nil {
		// Fire the callback in a separate goroutine to avoid blocking
		go func() {
			defer func() {
				if r := recover(); r != nil {
					pm.logger.Error("First reconcile callback panicked", "error", r)
				}
			}()
			pm.onFirstReconcileComplete()
		}()
	}
}

func (pm *ProcessManager) reconcileState(ctx context.Context) error {
	pm.logger.Debug("Starting reconciliation cycle.")
	desiredInstances, err := pm.desiredStateProvider.GetAppInstances(ctx)
	if err != nil {
		return fmt.Errorf("failed to get desired app instances: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	desiredMap := make(map[string]AppInstance)
	for _, inst := range desiredInstances {
		desiredMap[inst.InstanceID] = inst
	}

	// 1. Identify processes to start (in desired but not actual, or actual but not running correctly)
	for instanceID, desired := range desiredMap {
		actual, exists := pm.actualState[instanceID]
		if exists && (actual.GetState() == StateRunning || actual.GetState() == StateUnhealthy || actual.GetState() == StateStarting) {
			// Process exists and is in a running-like state, check for configuration changes
			if actual.Instance.BinPath != desired.BinPath || actual.Instance.DbName != desired.DbName {
				pm.logger.Info("Configuration changed for process, initiating restart", "instanceID", instanceID, "oldBin", actual.Instance.BinPath, "newBin", desired.BinPath, "oldDb", actual.Instance.DbName, "newDb", desired.DbName)
				// Stop the process. The reconciler or exit handler will then pick it up for a restart with the new config.
				// We run this in a goroutine to avoid blocking the reconciler loop.
				go func(procToStop *ManagedProcess) {
					if err := pm.stopProcess(ctx, procToStop, true); err != nil { // true to remove from actualState, allowing a clean restart
						pm.logger.Error("Failed to stop process for config update", "instanceID", procToStop.Instance.InstanceID, "error", err)
					}
				}(actual)
				// After stopping, the next part of the logic will handle starting it if it's now considered 'missing' or 'stopped'.
				// Or, we can continue to the next desired instance, and let the next reconciliation cycle handle the start.
				// For simplicity, we'll let the stop happen and the existing logic below will catch it.
			}
		}

		// This part now correctly handles starting if it doesn't exist, or if it exists but is stopped/failed (e.g. after a config change stop)
		actual, exists = pm.actualState[instanceID] // Re-fetch actual state as it might have been removed by stopProcess
		if !exists || actual.GetState() == StateStopped || actual.GetState() == StateFailed {
			pm.logger.Info("Process needs to be started", "instanceID", instanceID)
			go pm.startProcess(ctx, desired) // Run in a goroutine to avoid blocking reconciler
		}
	}

	// 2. Identify processes to stop (in actual but not in desired)
	for instanceID, actual := range pm.actualState {
		if _, existsInDesired := desiredMap[instanceID]; !existsInDesired {
			if actual.GetState() == StateRunning || actual.GetState() == StateStarting || actual.GetState() == StateUnhealthy {
				pm.logger.Info("Process needs to be stopped (no longer in desired state)", "instanceID", instanceID)
				go func(procToStop *ManagedProcess) { // Goroutine to avoid blocking
					if err := pm.stopProcess(ctx, procToStop, true); err != nil {
						pm.logger.Error("Failed to stop undesired process", "instanceID", procToStop.Instance.InstanceID, "error", err)
					}
				}(actual)
			}
		}
	}
	pm.logger.Debug("Reconciliation cycle finished.")

	// Check if this is the first successful reconciliation and fire callback if needed
	pm.checkAndFireFirstReconcileCallback()

	return nil
}

// startProcess handles the logic for launching a new subprocess.
// This includes port allocation, command construction, execution, and updating actual state.
func (pm *ProcessManager) startProcess(ctx context.Context, instance AppInstance) {
	pm.logger.Info("Attempting to start process", "instanceID", instance.InstanceID)

	// Check if already starting or running (double-check with lock)
	pm.mu.Lock()
	existingProcess, exists := pm.actualState[instance.InstanceID]
	if exists && (existingProcess.GetState() == StateRunning || existingProcess.GetState() == StateStarting) {
		pm.logger.Info("Process is already running or starting", "instanceID", instance.InstanceID, "state", existingProcess.GetState().String())
		pm.mu.Unlock()
		return
	}
	// If process exists but is stopped/failed, prepare for restart
	if exists {
		existingProcess.RecordRestart()
		// Apply backoff strategy
		backoffDuration := calculateBackoff(existingProcess.GetRestartCount(), pm.restartBackoffInitial, pm.restartBackoffMax)
		pm.logger.Info("Applying restart backoff", "instanceID", instance.InstanceID, "duration", backoffDuration, "restartCount", existingProcess.GetRestartCount())
		pm.mu.Unlock() // Unlock before sleep
		time.Sleep(backoffDuration)
		pm.mu.Lock() // Re-lock to continue
	} else {
		// Create a new ManagedProcess entry if it doesn't exist, will be populated further down
		// This is a temporary placeholder to mark it as 'being started'
		mpPlaceholder := &ManagedProcess{Instance: instance, State: StateStarting}
		pm.actualState[instance.InstanceID] = mpPlaceholder
	}
	pm.mu.Unlock()

	port, err := pm.portManager.AllocatePort()
	if err != nil {
		pm.logger.Error("Failed to allocate port", "instanceID", instance.InstanceID, "error", err)
		pm.mu.Lock()
		if proc, ok := pm.actualState[instance.InstanceID]; ok {
			proc.UpdateState(StateFailed)
		}
		pm.mu.Unlock()
		return
	}
	pm.logger.Info("Allocated port for process", "instanceID", instance.InstanceID, "port", port)

	cmdArgs := []string{
		"-dbPath", instance.DbName,
		"-port", fmt.Sprintf("%d", port),
	}

	pm.logger.Info("Starting process with command line", instance.BinPath, strings.Join(cmdArgs, " "))
	cmd := exec.CommandContext(ctx, instance.BinPath, cmdArgs...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOST=%s", instance.HostName))
	cmd.Env = append(cmd.Env, fmt.Sprintf("INTERNAL_SECRET=%s", pm.internalSecret))
	cmd.Dir = pm.subprocessWorkDir
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		pm.logger.Error("Failed to get stdout pipe", "instanceID", instance.InstanceID, "error", err)
		pm.portManager.ReleasePort(port)
		pm.mu.Lock()
		if proc, ok := pm.actualState[instance.InstanceID]; ok {
			proc.UpdateState(StateFailed)
		}
		pm.mu.Unlock()
		return
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		pm.logger.Error("Failed to get stderr pipe", "instanceID", instance.InstanceID, "error", err)
		stdoutPipe.Close() // Close stdoutPipe if stderrPipe fails
		pm.portManager.ReleasePort(port)
		pm.mu.Lock()
		if proc, ok := pm.actualState[instance.InstanceID]; ok {
			proc.UpdateState(StateFailed)
		}
		pm.mu.Unlock()
		return
	}

	if err := cmd.Start(); err != nil {
		pm.logger.Error("Failed to start subprocess", "instanceID", instance.InstanceID, "error", err, "command", cmd.String())
		pm.portManager.ReleasePort(port)
		pm.mu.Lock()
		if proc, ok := pm.actualState[instance.InstanceID]; ok {
			proc.UpdateState(StateFailed)
		}
		pm.mu.Unlock()
		return
	}

	mp := NewManagedProcess(instance, cmd, port)
	mp.UpdateState(StateRunning) // Initially assume running, health check will verify

	pm.mu.Lock()
	pm.actualState[instance.InstanceID] = mp
	pm.mu.Unlock()

	pm.logger.Info("Subprocess starting", "instanceID", instance.InstanceID, "pid", cmd.Process.Pid, "port", port, "command", cmd.String())

	// Goroutine to log stdout
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		defer stdoutPipe.Close()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			pm.logger.Info("Subprocess stdout", "instanceID", instance.InstanceID, "pid", cmd.Process.Pid, "output", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			pm.logger.Error("Error reading stdout from subprocess", "instanceID", instance.InstanceID, "pid", cmd.Process.Pid, "error", err)
		}
	}()

	// Goroutine to log stderr
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		defer stderrPipe.Close()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			pm.logger.Error("Subprocess stderr", "instanceID", instance.InstanceID, "pid", cmd.Process.Pid, "output", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			pm.logger.Error("Error reading stderr from subprocess", "instanceID", instance.InstanceID, "pid", cmd.Process.Pid, "error", err)
		}
	}()

	pm.logger.Info("Subprocess started successfully and output streams captured", "instanceID", instance.InstanceID, "pid", cmd.Process.Pid, "port", port)

	// Goroutine to wait for the process to exit and handle cleanup
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		err := cmd.Wait()
		pm.handleProcessExit(ctx, mp, err)
	}()
}

// stopProcess handles the logic for stopping a running subprocess.
// It sends SIGTERM, waits for graceful shutdown, then SIGKILL if necessary.
// If `removeFromActual` is true, it removes the process from actualState map.
func (pm *ProcessManager) stopProcess(ctx context.Context, process *ManagedProcess, removeFromActual bool) error {
	process.UpdateState(StateStopping)
	pm.logger.Info("Stopping process", "instanceID", process.Instance.InstanceID, "pid", process.PID)

	if process.Cmd == nil || process.Cmd.Process == nil {
		pm.logger.Warn("Process command or process itself is nil, cannot stop", "instanceID", process.Instance.InstanceID)
		process.UpdateState(StateStopped)
		if removeFromActual {
			pm.mu.Lock()
			delete(pm.actualState, process.Instance.InstanceID)
			pm.mu.Unlock()
		}
		pm.portManager.ReleasePort(process.Port)
		return nil
	}

	// Attempt graceful shutdown
	if err := process.Cmd.Process.Signal(os.Interrupt); err != nil { // os.Interrupt is often SIGINT, syscall.SIGTERM on Unix
		pm.logger.Error("Failed to send SIGTERM to process", "instanceID", process.Instance.InstanceID, "pid", process.PID, "error", err)
		// If signal fails, proceed to SIGKILL or log and consider it potentially stopped/crashed
	}

	gracefulShutdownTimer := time.NewTimer(pm.gracefulShutdownPeriod)
	processExitChan := make(chan error, 1)

	go func() {
		processExitChan <- process.Cmd.Wait() // Wait for the process to exit
	}()

	select {
	case err := <-processExitChan:
		gracefulShutdownTimer.Stop()
		if err != nil {
			pm.logger.Info("Process exited after SIGTERM (with error)", "instanceID", process.Instance.InstanceID, "pid", process.PID, "error", err)
		} else {
			pm.logger.Info("Process exited gracefully after SIGTERM", "instanceID", process.Instance.InstanceID, "pid", process.PID)
		}
	case <-gracefulShutdownTimer.C:
		pm.logger.Warn("Process did not exit gracefully, sending SIGKILL", "instanceID", process.Instance.InstanceID, "pid", process.PID)
		if err := process.Cmd.Process.Kill(); err != nil {
			pm.logger.Error("Failed to send SIGKILL to process", "instanceID", process.Instance.InstanceID, "pid", process.PID, "error", err)
			// At this point, the process might be orphaned or in an unrecoverable state
			process.UpdateState(StateFailed) // Or a new state like StateOrphaned
			// Not removing from actual state here as it might need manual intervention or further checks
			return fmt.Errorf("failed to kill process %s (PID %d): %w", process.Instance.InstanceID, process.PID, err)
		} else {
			pm.logger.Info("Process killed with SIGKILL", "instanceID", process.Instance.InstanceID, "pid", process.PID)
			// Wait for a short period to allow OS to clean up, then check Wait() again if needed
			<-processExitChan // Should now receive the exit error from Kill
		}
	case <-ctx.Done():
		pm.logger.Warn("Stop process context cancelled", "instanceID", process.Instance.InstanceID, "pid", process.PID)
		// Attempt a quick kill if context is cancelled during stop
		process.Cmd.Process.Kill()
		process.UpdateState(StateFailed)
		return ctx.Err()
	}

	process.UpdateState(StateStopped)
	pm.portManager.ReleasePort(process.Port)

	if removeFromActual {
		pm.mu.Lock()
		delete(pm.actualState, process.Instance.InstanceID)
		pm.mu.Unlock()
		pm.logger.Info("Process removed from actual state", "instanceID", process.Instance.InstanceID)
	}

	return nil
}

// handleProcessExit is called when a managed subprocess exits.
// It updates the process state and may trigger a restart based on policy.
func (pm *ProcessManager) handleProcessExit(ctx context.Context, process *ManagedProcess, exitErr error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	currentState := process.GetState()
	pm.logger.Info("Process exited", "instanceID", process.Instance.InstanceID, "pid", process.PID, "exitError", exitErr, "currentState", currentState.String())

	// Release port if it hasn't been (e.g. if stopProcess wasn't called explicitly for this exit)
	// This check is important because stopProcess also releases the port.
	if process.State != StateStopping && process.State != StateStopped {
		pm.portManager.ReleasePort(process.Port)
	}

	process.UpdateState(StateFailed) // Mark as failed due to unexpected exit

	// If the manager is stopping, or the process was intentionally stopped, don't restart.
	select {
	case <-pm.stopChan:
		pm.logger.Info("Manager is stopping, not restarting exited process", "instanceID", process.Instance.InstanceID)
		return
	default:
	}
	if currentState == StateStopping || currentState == StateStopped {
		pm.logger.Info("Process was intentionally stopped or already handled, not restarting", "instanceID", process.Instance.InstanceID)
		return
	}

	// Check if this process is still in the desired state. If not, don't restart.
	desiredInstances, err := pm.desiredStateProvider.GetAppInstances(ctx)
	if err != nil {
		pm.logger.Error("Failed to get desired instances during process exit handling", "instanceID", process.Instance.InstanceID, "error", err)
		// Decide on behavior: maybe retry getting desired state or don't restart for safety
		return
	}
	stillDesired := false
	var desiredInstanceConfig AppInstance
	for _, di := range desiredInstances {
		if di.InstanceID == process.Instance.InstanceID {
			stillDesired = true
			desiredInstanceConfig = di
			break
		}
	}

	if !stillDesired {
		pm.logger.Info("Process no longer in desired state, not restarting", "instanceID", process.Instance.InstanceID)
		delete(pm.actualState, process.Instance.InstanceID) // Clean up from actual state
		return
	}

	// Automatic restart for unexpected exit, respecting backoff
	pm.logger.Info("Process exited unexpectedly, attempting restart", "instanceID", process.Instance.InstanceID)
	// The startProcess function handles backoff internally based on restartCount
	go pm.startProcess(ctx, desiredInstanceConfig) // Use the latest desired config
}

// performHealthChecks iterates through running processes and checks their health.
func (pm *ProcessManager) performHealthChecks(ctx context.Context) {
	pm.mu.RLock() // Read lock to iterate over actualState
	processesToCheck := make([]*ManagedProcess, 0, len(pm.actualState))
	for _, process := range pm.actualState {
		if process.GetState() == StateRunning || process.GetState() == StateUnhealthy || process.GetState() == StateStarting {
			processesToCheck = append(processesToCheck, process)
		}
	}
	pm.mu.RUnlock()

	for _, process := range processesToCheck {
		// Check context before each potentially long operation
		select {
		case <-ctx.Done():
			pm.logger.Info("Health check context cancelled, stopping checks.")
			return
		case <-pm.stopChan:
			pm.logger.Info("Manager stopping, stopping health checks.")
			return
		default:
		}

		pm.checkAndUpdateHealth(ctx, process)
	}
}

// checkAndUpdateHealth performs a health check for a single process and updates its state.
// It also handles logic for consecutive failures and triggering restarts.
func (pm *ProcessManager) checkAndUpdateHealth(ctx context.Context, process *ManagedProcess) {
	pm.logger.Debug("Performing health check", "instanceID", process.Instance.InstanceID, "port", process.Port)
	newState, err := pm.healthChecker.Check(process)

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Ensure process still exists in actualState (could have been removed by reconciler)
	actualProcess, exists := pm.actualState[process.Instance.InstanceID]
	if !exists || actualProcess != process { // Pointer comparison to ensure it's the same process object
		pm.logger.Info("Process no longer in actual state or replaced, skipping health update", "instanceID", process.Instance.InstanceID)
		return
	}

	if err != nil {
		pm.logger.Warn("Health check failed for process", "instanceID", process.Instance.InstanceID, "error", err)
		// newState from checker might be StateUnhealthy or StateFailed depending on error type
	}

	currentInternalState := process.GetState() // Get state from the locked process object

	if newState == StateRunning {
		if currentInternalState != StateRunning {
			pm.logger.Info("Process is now healthy", "instanceID", process.Instance.InstanceID)
			process.UpdateState(StateRunning)
			process.unhealthySince = time.Time{} // Reset unhealthy timer
			process.restartCount = 0             // Reset restart count on successful health after being unhealthy/failed
		}
		process.lastHealthCh = time.Now()
	} else { // Unhealthy or some other failure state from check
		if currentInternalState == StateRunning {
			pm.logger.Warn("Process became unhealthy", "instanceID", process.Instance.InstanceID)
			process.UpdateState(StateUnhealthy)
			process.unhealthySince = time.Now()
		} else if currentInternalState == StateUnhealthy {
			// Already unhealthy, check for consecutive failures
			if !process.unhealthySince.IsZero() && time.Since(process.unhealthySince) >= time.Duration(pm.consecutiveFailures)*pm.healthCheckInterval {
				pm.logger.Error("Process persistently unhealthy, triggering restart", "instanceID", process.Instance.InstanceID, "unhealthyDuration", time.Since(process.unhealthySince))
				process.UpdateState(StateFailed) // Mark as failed to trigger restart logic
				// Unlock before calling startProcess as it will re-lock
				desiredConfig := process.Instance // Use existing config for restart
				pm.mu.Unlock()
				go pm.startProcess(ctx, desiredConfig) // Restart logic is handled by startProcess
				return                                 // Return early as lock is released
			}
		} else if currentInternalState == StateStarting {
			// If it's still 'Starting' after a health check interval, and the check fails, mark as unhealthy.
			pm.logger.Warn("Process failed first health check after starting", "instanceID", process.Instance.InstanceID)
			process.UpdateState(StateUnhealthy)
			process.unhealthySince = time.Now()
		}
		// If newState is StateFailed from the checker, update it directly
		if newState == StateFailed && currentInternalState != StateFailed {
			process.UpdateState(StateFailed)
			// Potentially trigger restart if appropriate (similar to persistent unhealthiness)
			pm.logger.Error("Health checker reported process as failed, triggering restart", "instanceID", process.Instance.InstanceID)
			desiredConfig := process.Instance
			pm.mu.Unlock()
			go pm.startProcess(ctx, desiredConfig)
			return
		}
	}
}

// calculateBackoff computes the backoff duration for restarting a process.
func calculateBackoff(restartCount int, initialDelay, maxDelay time.Duration) time.Duration {
	if restartCount <= 0 {
		return 0 // No delay for the first attempt (or if count is somehow non-positive)
	}
	// Simple exponential backoff: initialDelay * 2^(restartCount-1)
	backoff := initialDelay
	for i := 1; i < restartCount; i++ {
		backoff *= 2
		if backoff > maxDelay {
			return maxDelay
		}
	}
	return backoff
}

// SimpleAppInstanceProvider is a basic implementation of AppInstanceProvider using a fixed list.
// Useful for testing or simple scenarios.
type SimpleAppInstanceProvider struct {
	instances []AppInstance
	mu        sync.RWMutex
}

// NewSimpleAppInstanceProvider creates a provider with an initial set of instances.
func NewSimpleAppInstanceProvider(initialInstances []AppInstance) *SimpleAppInstanceProvider {
	return &SimpleAppInstanceProvider{instances: initialInstances}
}

// GetAppInstances returns the list of app instances.
func (p *SimpleAppInstanceProvider) GetAppInstances(ctx context.Context) ([]AppInstance, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// Return a copy to prevent modification of the internal slice by the caller
	instCopy := make([]AppInstance, len(p.instances))
	copy(instCopy, p.instances)
	return instCopy, nil
}

// UpdateAppInstances allows updating the list of desired instances at runtime.
// This is a simple way to simulate dynamic configuration changes for the SimpleAppInstanceProvider.
func (p *SimpleAppInstanceProvider) UpdateAppInstances(newInstances []AppInstance) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.instances = newInstances
}
