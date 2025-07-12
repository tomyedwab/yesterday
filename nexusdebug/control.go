// Package nexusdebug implements interactive control for the NexusDebug CLI tool.
//
// This module provides non-blocking keyboard input detection and coordinates
// with the monitor to handle rebuild and shutdown workflows while managing
// terminal output effectively.
//
// Reference: spec/nexusdebug.md - Task nexusdebug-interactive-control
package nexusdebug

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/term"
	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

// Control handles interactive user input and hot-reload workflows
type Control struct {
	client         *yesterdaygo.Client
	app            *DebugApplication
	monitor        *Monitor
	buildManager   *BuildManager
	uploadManager  *UploadManager
	appManager     *ApplicationManager
	stopChan       chan struct{}
	terminalState  *term.State
	inputCheckInterval time.Duration
}

// ControlCallbacks defines callback functions for control operations
type ControlCallbacks struct {
	OnRebuild  func(ctx context.Context) error
	OnShutdown func(ctx context.Context) error
}

// NewControl creates a new interactive control instance
func NewControl(client *yesterdaygo.Client, app *DebugApplication, monitor *Monitor) *Control {
	return &Control{
		client:            client,
		app:               app,
		monitor:           monitor,
		stopChan:          make(chan struct{}),
		inputCheckInterval: 100 * time.Millisecond, // Check for input every 100ms
	}
}

// SetManagers configures the necessary managers for rebuild operations
func (c *Control) SetManagers(buildMgr *BuildManager, uploadMgr *UploadManager, appMgr *ApplicationManager) {
	c.buildManager = buildMgr
	c.uploadManager = uploadMgr
	c.appManager = appMgr
}

// StartInteractiveMode begins the interactive control loop
func (c *Control) StartInteractiveMode(ctx context.Context) error {
	// Display initial instructions
	c.showInstructions()

	// Start the interactive loop
	go c.interactiveLoop(ctx)

	return nil
}

// StopInteractiveMode stops the interactive control loop
func (c *Control) StopInteractiveMode() {
	close(c.stopChan)
	c.restoreTerminal()
}

// showInstructions displays user instructions for interactive commands
func (c *Control) showInstructions() {
	fmt.Println("🎮 Interactive Mode Enabled")
	fmt.Println("Press 'R' to rebuild and redeploy, 'Q' to quit gracefully")
	fmt.Println("Monitoring logs and status updates...")
	fmt.Println()
}

// interactiveLoop runs the main interactive control loop
func (c *Control) interactiveLoop(ctx context.Context) {
	ticker := time.NewTicker(c.inputCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			// Check for input in raw mode, then return to cooked mode for output
			if key, err := c.checkForInput(); err == nil && key != 0 {
				c.handleKeyPress(ctx, key)
			}
		}
	}
}

// checkForInput briefly switches to raw mode to check for keyboard input
func (c *Control) checkForInput() (byte, error) {
	// Switch to raw mode for input detection
	if err := c.setRawMode(); err != nil {
		return 0, fmt.Errorf("failed to set raw mode: %w", err)
	}

	// Always restore to cooked mode when done
	defer c.restoreTerminal()

	// Use a very simple approach: try to read one byte with immediate timeout
	// This avoids the goroutine complexity
	buf := make([]byte, 1)
	
	// Try to read without blocking by immediately restoring and checking
	// if there was input queued
	c.restoreTerminal()
	c.setRawMode()
	
	// Set a very short deadline for non-blocking read
	deadline := time.Now().Add(1 * time.Millisecond)
	os.Stdin.SetReadDeadline(deadline)
	defer os.Stdin.SetReadDeadline(time.Time{}) // Clear deadline
	
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return 0, nil // No input available
	}
	return buf[0], nil
}

// setRawMode switches the terminal to raw mode for input detection
func (c *Control) setRawMode() error {
	fd := int(os.Stdin.Fd())
	
	// Save current state if not already saved
	if c.terminalState == nil {
		state, err := term.GetState(fd)
		if err != nil {
			return fmt.Errorf("failed to get terminal state: %w", err)
		}
		c.terminalState = state
	}

	// Switch to raw mode
	_, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}

	return nil
}

// restoreTerminal restores the terminal to its original cooked mode
func (c *Control) restoreTerminal() {
	if c.terminalState != nil {
		fd := int(os.Stdin.Fd())
		term.Restore(fd, c.terminalState)
	}
}

// handleKeyPress processes keyboard input and executes appropriate actions
func (c *Control) handleKeyPress(ctx context.Context, key byte) {
	switch key {
	case 'r', 'R':
		fmt.Println("\n🔄 Rebuild requested...")
		if err := c.handleRebuild(ctx); err != nil {
			fmt.Printf("❌ Rebuild failed: %v\n", err)
		}
	case 'q', 'Q':
		fmt.Println("\n👋 Graceful shutdown requested...")
		if err := c.handleShutdown(ctx); err != nil {
			fmt.Printf("❌ Shutdown failed: %v\n", err)
		}
	case 3: // Ctrl+C
		fmt.Println("\n⚠️  Interrupt received, shutting down...")
		if err := c.handleShutdown(ctx); err != nil {
			fmt.Printf("❌ Shutdown failed: %v\n", err)
		}
	default:
		// Ignore other keys
	}
}

// handleRebuild executes the rebuild and redeploy workflow
func (c *Control) handleRebuild(ctx context.Context) error {
	if c.buildManager == nil || c.uploadManager == nil || c.appManager == nil {
		return fmt.Errorf("managers not configured for rebuild operations")
	}

	// Step 1: Stop current application instance
	fmt.Println("🛑 Stopping current application...")
	if err := c.appManager.StopApplication(ctx); err != nil {
		log.Printf("Warning: Failed to stop application: %v", err)
		// Continue with rebuild even if stop fails
	}

	// Step 2: Execute build command
	fmt.Println("🔨 Building application...")
	if err := c.buildManager.BuildApplication(ctx); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	fmt.Println("✅ Build completed successfully")

	// Step 3: Upload new package
	fmt.Println("📦 Uploading new package...")
	if err := c.uploadManager.UploadPackageFromBuildManager(ctx, c.buildManager, PrintUploadProgress); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	fmt.Println("✅ Package uploaded successfully")

	// Step 4: Install and start updated application
	fmt.Println("🚀 Installing and starting application...")
	if err := c.uploadManager.InstallUploadedPackage(ctx); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}
	fmt.Println("✅ Application redeployed successfully")

	// Step 5: Resume log monitoring (already running, just notify)
	fmt.Println("📊 Resuming log monitoring...")
	fmt.Println("Press 'R' to rebuild again, 'Q' to quit")

	return nil
}

// handleShutdown executes the graceful shutdown workflow
func (c *Control) handleShutdown(ctx context.Context) error {
	// Step 1: Stop monitoring
	fmt.Println("📊 Stopping monitoring...")
	if c.monitor != nil {
		c.monitor.StopMonitoring()
	}

	// Step 2: Stop application instance and clean up
	fmt.Println("🛑 Stopping application instance...")
	if c.appManager != nil {
		if err := c.appManager.Cleanup(ctx); err != nil {
			log.Printf("Warning: Failed to cleanup application: %v", err)
		}
	}

	// Step 3: Close authentication session
	fmt.Println("🔐 Closing authentication session...")
	if c.client != nil {
		if err := c.client.Logout(ctx); err != nil {
			log.Printf("Warning: Failed to logout: %v", err)
		}
	}

	// Step 4: Stop interactive mode
	c.StopInteractiveMode()

	fmt.Println("✅ Graceful shutdown completed")
	
	// Exit the program
	os.Exit(0)
	
	return nil
}