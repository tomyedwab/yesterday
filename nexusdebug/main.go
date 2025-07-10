// Package main implements the NexusDebug CLI tool for rapid iteration and debugging
// of Go server applications running within the NexusHub platform.
//
// This tool provides an automated workflow for building, deploying, and monitoring
// debug applications with hot-reload capabilities.
//
// Reference: spec/nexusdebug.md - Task nexusdebug-cli-setup
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Config holds the CLI configuration parameters
type Config struct {
	AdminURL         string // Required: Target NexusHub admin service URL
	AppName          string // Required: Used to generate AppID, DisplayName, and HostName
	BuildCommand     string // Optional: Defaults to "make build"
	PackageFilename  string // Optional: Defaults to "dist/package.zip"
	StaticServiceURL string // Optional: For proxying frontend requests during development
}

// validateConfig validates the configuration and returns an error if invalid
func validateConfig(config *Config) error {
	if config.AdminURL == "" {
		return fmt.Errorf("admin URL is required")
	}

	if config.AppName == "" {
		return fmt.Errorf("application name is required")
	}

	// Validate that package filename directory exists or can be created
	packageDir := filepath.Dir(config.PackageFilename)
	if packageDir != "." {
		if _, err := os.Stat(packageDir); os.IsNotExist(err) {
			log.Printf("Warning: Package directory %s does not exist", packageDir)
		}
	}

	return nil
}

// printUsage prints the CLI usage information
func printUsage() {
	fmt.Fprintf(os.Stderr, `NexusDebug CLI Tool

A development utility for rapid iteration and debugging of Go server applications
running within the NexusHub platform.

Usage:
  %s [options]

Options:
`, os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Examples:
  %s -admin-url=https://admin.example.com -app-name=myapp
  %s -admin-url=https://admin.example.com -app-name=myapp -build-cmd="go build" -package="build/app.zip"

Interactive Commands (during execution):
  R - Rebuild and redeploy application
  Q - Quit and cleanup debug application

`, os.Args[0], os.Args[0])
}

func main() {
	var config Config
	var showHelp bool

	// Define command-line flags
	flag.StringVar(&config.AdminURL, "admin-url", "", "Target NexusHub admin service URL (required)")
	flag.StringVar(&config.AppName, "app-name", "", "Application name for generating AppID, DisplayName, and HostName (required)")
	flag.StringVar(&config.BuildCommand, "build-cmd", "make build", "Build command to execute")
	flag.StringVar(&config.PackageFilename, "package", "dist/package.zip", "Package filename path")
	flag.StringVar(&config.StaticServiceURL, "static-url", "", "Static service URL for proxying frontend requests during development")
	flag.BoolVar(&showHelp, "help", false, "Show this help message")
	flag.BoolVar(&showHelp, "h", false, "Show this help message")

	// Custom usage function
	flag.Usage = printUsage

	// Parse command-line arguments
	flag.Parse()

	// Show help if requested
	if showHelp {
		printUsage()
		os.Exit(0)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		printUsage()
		os.Exit(1)
	}

	// Initialize structured logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("NexusDebug CLI starting...")
	log.Printf("Configuration:")
	log.Printf("  Admin URL: %s", config.AdminURL)
	log.Printf("  Application Name: %s", config.AppName)
	log.Printf("  Build Command: %s", config.BuildCommand)
	log.Printf("  Package Filename: %s", config.PackageFilename)
	if config.StaticServiceURL != "" {
		log.Printf("  Static Service URL: %s", config.StaticServiceURL)
	}

	// Initialize authentication manager
	authManager := NewAuthManager(config.AdminURL)

	// Create context with timeout for authentication
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Perform authentication
	log.Printf("Authentication status: %s", authManager.GetAuthenticationStatus(ctx))
	if err := authManager.Login(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Authentication status: %s", authManager.GetAuthenticationStatus(ctx))

	// Initialize application manager
	appManager := NewApplicationManager(authManager.client, config.AppName, config.StaticServiceURL)

	// Setup graceful shutdown handling
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Setup cleanup on exit
	defer func() {
		log.Printf("Cleaning up debug application...")
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := appManager.Cleanup(cleanupCtx); err != nil {
			log.Printf("Cleanup error: %v", err)
		}
	}()

	// Create debug application
	log.Printf("Creating debug application...")
	createCtx, createCancel := context.WithTimeout(ctx, 30*time.Second)
	defer createCancel()
	app, err := appManager.CreateApplication(createCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create debug application: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nDebug application created successfully!\n")
	fmt.Printf("Application ID: %s\n", app.ID)
	fmt.Printf("Display Name: %s\n", app.DisplayName)
	fmt.Printf("Host Name: %s\n", app.HostName)
	if app.StaticServiceURL != "" {
		fmt.Printf("Static Service URL: %s\n", app.StaticServiceURL)
	}

	// Install the application
	log.Printf("Installing debug application...")
	installCtx, installCancel := context.WithTimeout(ctx, 120*time.Second)
	defer installCancel()
	if err := appManager.InstallApplication(installCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to install debug application: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Debug application is now running!\n")
	fmt.Printf("🌐 Access your application at: https://%s\n", app.HostName)
	fmt.Printf("\n📋 Next Steps:\n")
	fmt.Printf("  1. Upload your application package using the build/upload tasks\n")
	fmt.Printf("  2. Use 'R' key for hot-reload after making changes\n")
	fmt.Printf("  3. Use 'Q' key to exit gracefully\n")
	fmt.Printf("\n🔄 Application lifecycle management completed successfully!")

	fmt.Println("NexusDebug CLI initialized successfully!")
	fmt.Println("Press 'R' to rebuild and redeploy, 'Q' to quit")
	fmt.Println("(Authentication implemented, remaining functionality will be implemented in subsequent tasks)")

	// Cleanup on exit
	defer func() {
		if err := authManager.Logout(context.Background()); err != nil {
			log.Printf("Warning: logout failed: %v", err)
		}
	}()
}
