package main

import (
	"context"
	// "fmt" // Removed as it's unused. Lint ID: fa9219a1-aae4-4e75-a3b9-d99fa1d52c1b
	"log/slog"
	"os"
	"os/signal" // Added for filepath.Dir. Lint IDs: 4681a3e8-caed-4e19-be9a-3b436920e3fa, d23ae48c-77ba-496b-ac1c-e03f86de2ad1
	"syscall"
	"time"

	"github.com/tomyedwab/yesterday/database/processes"
)

func main() {
	// 1. Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	logger.Info("Starting NexusHub Process Manager")

	// 2. Create AppInstanceProvider with startup instances
	sampleInstances := []processes.AppInstance{
		{
			InstanceID: "3bf3e3c0-6e51-482a-b180-00f6aa568ee9",
			WasmPath:   "dist/0001-0001/app.wasm",
			DbName:     "users.db",
		},
	}
	appProvider := processes.NewSimpleAppInstanceProvider(sampleInstances)

	// 3. Initialize PortManager
	// Ensure this range doesn't conflict with other services on your system.
	portManager, err := processes.NewPortManager(10000, 19999) // Example port range
	if err != nil {
		logger.Error("Failed to create PortManager", "error", err)
		os.Exit(1)
	}

	// 4. Configure and create ProcessManager
	// Assume we are running from the project root.
	projectRoot, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get current working directory", "error", err)
		os.Exit(1)
	}
	logger.Info("Project root", "path", projectRoot)

	pmConfig := processes.Config{
		AppProvider:            appProvider,
		PortManager:            portManager,
		Logger:                 logger,
		HealthCheckInterval:    10 * time.Second, // Check health more frequently for demo
		HealthCheckTimeout:     3 * time.Second,
		ConsecutiveFailures:    2, // Restart after 2 failures for demo
		RestartBackoffInitial:  2 * time.Second,
		RestartBackoffMax:      15 * time.Second,
		GracefulShutdownPeriod: 5 * time.Second,
		SubprocessWorkDir:      projectRoot, // Processes will run from the project root
	}

	processManager, err := processes.NewProcessManager(pmConfig)
	if err != nil {
		logger.Error("Failed to create ProcessManager", "error", err)
		os.Exit(1)
	}

	// 5. Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received signal, shutting down...", "signal", sig.String())
		processManager.Stop() // Signal the process manager to stop
		cancel()              // Cancel the main context to allow Run to unblock if it respects context
	}()

	// 6. Run the ProcessManager
	logger.Info("Running ProcessManager... Press Ctrl+C to exit.")
	processManager.Run(ctx) // This blocks until Stop() is called or context is cancelled

	logger.Info("ProcessManager has shut down. Exiting demo.")
}
