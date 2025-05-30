package main

import (
	"context"
	// "fmt" // Removed as it's unused. Lint ID: fa9219a1-aae4-4e75-a3b9-d99fa1d52c1b
	"log/slog"
	"os"
	"os/signal" // Added for filepath.Dir. Lint IDs: 4681a3e8-caed-4e19-be9a-3b436920e3fa, d23ae48c-77ba-496b-ac1c-e03f86de2ad1
	"syscall"
	"time"

	"net/http" // Added for http.ErrServerClosed

	"github.com/tomyedwab/yesterday/database/processes"
	"github.com/tomyedwab/yesterday/nexushub/httpsproxy" // Added for HTTPS Proxy
)

func main() {
	var httpProxy *httpsproxy.Proxy // Declare proxy variable for access in shutdown handler

	// 1. Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	logger.Info("Starting NexusHub Process Manager")

	// 2. Create AppInstanceProvider with startup instances
	sampleInstances := []processes.AppInstance{
		{
			InstanceID: "3bf3e3c0-6e51-482a-b180-00f6aa568ee9",
			HostName:   "login.yesterday.localhost",
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
		logger.Info("Received signal, initiating graceful shutdown...", "signal", sig.String())

		// Initiate proxy shutdown first
		if httpProxy != nil {
			logger.Info("Attempting to stop HTTPS Proxy server...")
			if err := httpProxy.Stop(); err != nil {
				logger.Error("Error stopping HTTPS Proxy server", "error", err)
			} else {
				logger.Info("HTTPS Proxy server stopped gracefully.")
			}
		} else {
			logger.Info("HTTPS Proxy was not initialized, skipping stop.")
		}

		// Initiate process manager shutdown
		logger.Info("Attempting to stop ProcessManager...")
		if processManager != nil { // Good practice to check if it was initialized
			processManager.Stop()
			logger.Info("ProcessManager stop signal sent.")
		} else {
			logger.Info("ProcessManager was not initialized, skipping stop.")
		}

		logger.Info("Cancelling main context to allow all services to complete shutdown.")
		cancel() // Cancel the main context
	}()

	// 6. Initialize the HTTPS Proxy
	// TODO: Replace with actual paths to your SSL certificate and key files.
	// Ensure these files exist and are accessible.
	// You can generate self-signed certificates for testing if needed:
	// openssl genrsa -out server.key 2048
	// openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
	proxyListenAddr := ":8443"
	proxyCertFile := "dist/certs/server.crt" // Placeholder - ensure this file exists or path is correct
	proxyKeyFile := "dist/certs/server.key"  // Placeholder - ensure this file exists or path is correct
	logger.Info("Attempting to configure HTTPS Proxy", "listenAddr", proxyListenAddr, "certFile", proxyCertFile, "keyFile", proxyKeyFile)

	// httpProxy is declared at the top of main for access in the shutdown handler
	httpProxy = httpsproxy.NewProxy(proxyListenAddr, proxyCertFile, proxyKeyFile, processManager)

	// 7. Start the HTTPS Proxy server in a goroutine
	go func() {
		logger.Info("Starting HTTPS Proxy server...", "address", proxyListenAddr)
		if err := httpProxy.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTPS Proxy server failed to start or unexpectedly stopped", "error", err)
			// Consider a more robust way to signal main application failure if proxy is critical
			// For example, by closing a channel that main select{}s on, or calling sigChan <- syscall.SIGTERM
		} else if err == http.ErrServerClosed {
			logger.Info("HTTPS Proxy server closed.")
		}
	}()

	// 8. Run the ProcessManager (this is blocking)
	logger.Info("Running ProcessManager... Press Ctrl+C to exit.")
	processManager.Run(ctx) // This blocks until Stop() is called or context is cancelled
	<-ctx.Done()

	logger.Info("NexusHub components have completed their shutdown sequence. Exiting main.")
}
