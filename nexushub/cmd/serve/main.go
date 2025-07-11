package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"net/http"

	"github.com/google/uuid"
	"github.com/tomyedwab/yesterday/nexushub/httpsproxy"
	"github.com/tomyedwab/yesterday/nexushub/packages"
	"github.com/tomyedwab/yesterday/nexushub/processes"
)

func main() {
	var httpProxy *httpsproxy.Proxy // Declare proxy variable for access in shutdown handler

	internalSecret := uuid.New().String()

	// 1. Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	logger.Info("Starting NexusHub Process Manager")

	packageManager := packages.NewPackageManager()
	if !packageManager.IsInstalled("18736e4f-93f9-4606-a7be-863c7986ea5b") {
		logger.Warn("Admin app not installed, installing...")
		if err := packageManager.InstallPackage("github_com__tomyedwab__yesterday__apps__admin", "18736e4f-93f9-4606-a7be-863c7986ea5b"); err != nil {
			logger.Error("Failed to install admin app", "error", err)
			os.Exit(1)
		}
	}
	if !packageManager.IsInstalled("3bf3e3c0-6e51-482a-b180-00f6aa568ee9") {
		logger.Warn("Login app not installed, installing...")
		if err := packageManager.InstallPackage("github_com__tomyedwab__yesterday__apps__login", "3bf3e3c0-6e51-482a-b180-00f6aa568ee9"); err != nil {
			logger.Error("Failed to install login app", "error", err)
			os.Exit(1)
		}
	}

	installDir := packageManager.GetInstallDir()

	// 2. Create AdminInstanceProvider with static instances for critical services
	staticApps := []processes.StaticAppConfig{
		{
			InstanceID: "3bf3e3c0-6e51-482a-b180-00f6aa568ee9",
			HostName:   "login.yesterday.localhost:8443",
			PkgPath:    filepath.Join(installDir, "3bf3e3c0-6e51-482a-b180-00f6aa568ee9"),
		},
		{
			InstanceID: "18736e4f-93f9-4606-a7be-863c7986ea5b",
			HostName:   "admin.yesterday.localhost:8443",
			PkgPath:    filepath.Join(installDir, "18736e4f-93f9-4606-a7be-863c7986ea5b"),
		},
	}
	appProvider := processes.NewAdminInstanceProvider("18736e4f-93f9-4606-a7be-863c7986ea5b", internalSecret, installDir, staticApps)

	// 3. Initialize PortManager
	portManager, err := processes.NewPortManager(10000, 19999)
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

	processManager, err := processes.NewProcessManager(pmConfig, internalSecret)
	if err != nil {
		logger.Error("Failed to create ProcessManager", "error", err)
		os.Exit(1)
	}

	// 5. Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processManager.SetFirstReconcileCompleteCallback(func() {
		// Start the admin instance provider only once we've successfully
		// started the static applications, including the admin application
		// itself
		if err := appProvider.Start(ctx); err != nil {
			logger.Error("Failed to start AdminInstanceProvider", "error", err)
			os.Exit(1)
		}
	})

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

		// Stop admin instance provider
		if appProvider != nil {
			logger.Info("Stopping AdminInstanceProvider...")
			appProvider.Stop()
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
	proxyCertFile := "dist/certs/server.crt" // Placeholder
	proxyKeyFile := "dist/certs/server.key"  // Placeholder
	logger.Info("Attempting to configure HTTPS Proxy", "listenAddr", proxyListenAddr, "certFile", proxyCertFile, "keyFile", proxyKeyFile)

	// httpProxy is declared at the top of main for access in the shutdown handler
	httpProxy = httpsproxy.NewProxy(proxyListenAddr, proxyCertFile, proxyKeyFile, internalSecret, processManager, appProvider)

	// 7. Start the HTTPS Proxy server in a goroutine
	go func() {
		logger.Info("Starting HTTPS Proxy server...", "address", proxyListenAddr)
		if err := httpProxy.Start(appProvider); err != nil && err != http.ErrServerClosed {
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
