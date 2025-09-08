package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/tomyedwab/yesterday/nexushub/events"
	"github.com/tomyedwab/yesterday/nexushub/httpsproxy"
	"github.com/tomyedwab/yesterday/nexushub/packages"
	"github.com/tomyedwab/yesterday/nexushub/processes"
	"github.com/tomyedwab/yesterday/nexushub/sessions"
)

func main() {
	var httpProxy *httpsproxy.Proxy // Declare proxy variable for access in shutdown handler

	proxyListenAddr := ":8443"
	internalSecret := uuid.New().String()
	hostName := fmt.Sprintf("www.yesterday.localhost%s", proxyListenAddr)

	// 1. Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	logger.Info("Starting NexusHub Process Manager")

	packageManager, err := packages.NewPackageManager()
	if err != nil {
		logger.Error("Failed to initialize package manager", "error", err)
		os.Exit(1)
	}
	installDir := packageManager.GetInstallDir()

	if !packageManager.IsInstalled("MBtskI6D") {
		logger.Warn("Admin app not installed, installing...")
		// TODO(tom): Implement the package hash?
		if err := packageManager.InstallPackage("github_com__tomyedwab__yesterday__apps__admin", "", "MBtskI6D"); err != nil {
			logger.Error("Failed to install admin app", "error", err)
			os.Exit(1)
		}
	}

	sessionsDatabase := sqlx.MustConnect("sqlite3", path.Join(installDir, "sessions.db"))
	sessionManager, err := sessions.NewManager(sessionsDatabase, 10*time.Minute, 1*time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	// Create EventManager
	eventsDatabase := sqlx.MustConnect("sqlite3", path.Join(installDir, "events.db"))
	eventManager, err := events.CreateEventManager(eventsDatabase)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Create AdminInstanceProvider with static instances for critical services
	/*
		staticApps := []processes.StaticAppConfig{
			{
				InstanceID: "MBtskI6D",
				HostName:   hostName,
				PkgPath:    filepath.Join(installDir, "MBtskI6D"),
				Subscriptions: map[string]bool{
					"User:Add":            true,
					"User:UpdatePassword": true,
					"User:Delete":         true,
					"User:Update":         true,
				},
			},
			}
		appProvider := processes.NewAdminInstanceProvider("MBtskI6D", internalSecret, installDir, staticApps)
	*/

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
		InstanceProvider:       packageManager,
		PortManager:            portManager,
		Logger:                 logger,
		HealthCheckInterval:    10 * time.Second, // Check health more frequently for demo
		HealthCheckTimeout:     3 * time.Second,
		ConsecutiveFailures:    2, // Restart after 2 failures for demo
		RestartBackoffInitial:  2 * time.Second,
		RestartBackoffMax:      15 * time.Second,
		GracefulShutdownPeriod: 5 * time.Second,
		SubprocessWorkDir:      projectRoot, // Processes will run from the project root
		EventManager:           eventManager,
	}

	processManager, err := processes.NewProcessManager(pmConfig, internalSecret)
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
	certsDir := os.Getenv("CERTS_DIR")
	if certsDir == "" {
		certsDir = "/usr/local/etc/nexushub/certs"
	}
	proxyCertFile := fmt.Sprintf("%s/server.crt", certsDir)
	proxyKeyFile := fmt.Sprintf("%s/server.key", certsDir)
	logger.Info("Attempting to configure HTTPS Proxy", "listenAddr", proxyListenAddr, "certFile", proxyCertFile, "keyFile", proxyKeyFile)

	// httpProxy is declared at the top of main for access in the shutdown handler
	httpProxy = httpsproxy.NewProxy(
		proxyListenAddr,
		hostName,
		proxyCertFile,
		proxyKeyFile,
		internalSecret,
		processManager,
		packageManager,
		eventManager)

	contextFn := func(_ net.Listener) context.Context {
		return context.WithValue(context.Background(), sessions.SessionManagerKey, sessionManager)
	}

	// 7. Start the HTTPS Proxy server in a goroutine
	go func() {
		logger.Info("Starting HTTPS Proxy server...", "address", proxyListenAddr)
		if err := httpProxy.Start(contextFn); err != nil && err != http.ErrServerClosed {
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
