// Package build implements the build system functionality for the NexusDebug CLI tool.
//
// This module handles application building, package creation, validation, and
// build artifact management for debug applications.
//
// Reference: spec/nexusdebug.md - Task nexusdebug-build-system
package nexusdebug

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BuildManager handles application build operations
type BuildManager struct {
	buildCommand    string
	packageFilename string
	workingDir      string
}

// NewBuildManager creates a new BuildManager instance with the specified configuration
func NewBuildManager(buildCommand, packageFilename string) *BuildManager {
	// Get current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: could not determine working directory: %v", err)
		workingDir = "."
	}

	return &BuildManager{
		buildCommand:    buildCommand,
		packageFilename: packageFilename,
		workingDir:      workingDir,
	}
}

// BuildApplication executes the build command and creates the package
func (bm *BuildManager) BuildApplication(ctx context.Context) error {
	log.Printf("🔨 Starting build process...")
	log.Printf("Build command: %s", bm.buildCommand)
	log.Printf("Working directory: %s", bm.workingDir)
	log.Printf("Package filename: %s", bm.packageFilename)

	// Clean up previous build artifacts
	if err := bm.CleanupBuildArtifacts(); err != nil {
		log.Printf("Warning: failed to cleanup previous build artifacts: %v", err)
	}

	// Execute build command
	if err := bm.executeBuildCommand(ctx); err != nil {
		return fmt.Errorf("build command failed: %w", err)
	}

	// Validate build artifacts and package
	if err := bm.ValidatePackage(); err != nil {
		return fmt.Errorf("package validation failed: %w", err)
	}

	log.Printf("✅ Build process completed successfully!")
	return nil
}

// executeBuildCommand runs the configured build command and monitors output
func (bm *BuildManager) executeBuildCommand(ctx context.Context) error {
	// Parse build command into command and arguments
	parts := strings.Fields(bm.buildCommand)
	if len(parts) == 0 {
		return fmt.Errorf("empty build command")
	}

	command := parts[0]
	args := parts[1:]

	log.Printf("Executing: %s %s", command, strings.Join(args, " "))

	// Create command with context for timeout handling
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = bm.workingDir

	// Set up real-time output monitoring
	cmd.Stdout = &buildOutputWriter{prefix: "[BUILD]"}
	cmd.Stderr = &buildOutputWriter{prefix: "[ERROR]"}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start build command: %w", err)
	}

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("build command timed out")
		}
		return fmt.Errorf("build command failed with exit code: %w", err)
	}

	return nil
}

// buildOutputWriter provides real-time build output with prefixes
type buildOutputWriter struct {
	prefix string
}

func (bow *buildOutputWriter) Write(p []byte) (n int, err error) {
	output := strings.TrimSpace(string(p))
	if output != "" {
		log.Printf("%s %s", bow.prefix, output)
	}
	return len(p), nil
}

// ValidatePackage checks if the build artifacts exist and validates package structure
func (bm *BuildManager) ValidatePackage() error {
	log.Printf("🔍 Validating build artifacts...")

	// Check if package file exists
	packagePath := filepath.Join(bm.workingDir, bm.packageFilename)
	stat, err := os.Stat(packagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("package file not found: %s", packagePath)
		}
		return fmt.Errorf("failed to check package file: %w", err)
	}

	// Validate package is not empty
	if stat.Size() == 0 {
		return fmt.Errorf("package file is empty: %s", packagePath)
	}

	// Log package information
	log.Printf("📦 Package validation successful:")
	log.Printf("  File: %s", packagePath)
	log.Printf("  Size: %d bytes (%.2f MB)", stat.Size(), float64(stat.Size())/1024/1024)
	log.Printf("  Modified: %s", stat.ModTime().Format(time.RFC3339))

	// Validate package format based on extension
	if err := bm.validatePackageFormat(packagePath); err != nil {
		return fmt.Errorf("package format validation failed: %w", err)
	}

	return nil
}

// validatePackageFormat validates the package based on its file extension
func (bm *BuildManager) validatePackageFormat(packagePath string) error {
	ext := strings.ToLower(filepath.Ext(packagePath))

	switch ext {
	case ".zip":
		return bm.validateZipPackage(packagePath)
	case ".tar", ".tgz", ".tar.gz":
		return bm.validateTarPackage(packagePath)
	default:
		log.Printf("Warning: unknown package format %s, skipping format validation", ext)
		return nil
	}
}

// validateZipPackage validates ZIP package structure
func (bm *BuildManager) validateZipPackage(packagePath string) error {
	// Try to open the ZIP file to validate it's not corrupted
	cmd := exec.Command("unzip", "-t", packagePath)
	if err := cmd.Run(); err != nil {
		// If unzip is not available, try with file command
		cmd = exec.Command("file", packagePath)
		output, fileErr := cmd.Output()
		if fileErr != nil {
			log.Printf("Warning: could not validate ZIP package format (unzip and file commands failed)")
			return nil
		}

		if !strings.Contains(strings.ToLower(string(output)), "zip") {
			return fmt.Errorf("file does not appear to be a valid ZIP archive")
		}
	}

	log.Printf("✅ ZIP package format validation passed")
	return nil
}

// validateTarPackage validates TAR package structure
func (bm *BuildManager) validateTarPackage(packagePath string) error {
	// Try to list tar contents to validate it's not corrupted
	cmd := exec.Command("tar", "-tf", packagePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("invalid TAR archive: %w", err)
	}

	log.Printf("✅ TAR package format validation passed")
	return nil
}

// CleanupBuildArtifacts removes previous build artifacts to ensure clean builds
func (bm *BuildManager) CleanupBuildArtifacts() error {
	packagePath := filepath.Join(bm.workingDir, bm.packageFilename)

	// Check if package file exists
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		// No previous artifacts to clean up
		return nil
	}

	log.Printf("🧹 Cleaning up previous build artifacts...")

	// Remove existing package file
	if err := os.Remove(packagePath); err != nil {
		return fmt.Errorf("failed to remove existing package file: %w", err)
	}

	log.Printf("Previous package file removed: %s", packagePath)
	return nil
}

// GetPackagePath returns the full path to the package file
func (bm *BuildManager) GetPackagePath() string {
	return filepath.Join(bm.workingDir, bm.packageFilename)
}

// GetPackageSize returns the size of the package file in bytes
func (bm *BuildManager) GetPackageSize() (int64, error) {
	packagePath := bm.GetPackagePath()
	stat, err := os.Stat(packagePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get package size: %w", err)
	}
	return stat.Size(), nil
}

// BuildWithTimeout executes the build with a configurable timeout
func (bm *BuildManager) BuildWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return bm.BuildApplication(ctx)
}

// SetWorkingDirectory updates the working directory for build operations
func (bm *BuildManager) SetWorkingDirectory(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Verify directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absDir)
	}

	bm.workingDir = absDir
	log.Printf("Working directory updated to: %s", bm.workingDir)
	return nil
}

// UpdateBuildCommand allows updating the build command at runtime
func (bm *BuildManager) UpdateBuildCommand(command string) {
	bm.buildCommand = command
	log.Printf("Build command updated to: %s", bm.buildCommand)
}

// UpdatePackageFilename allows updating the package filename at runtime
func (bm *BuildManager) UpdatePackageFilename(filename string) {
	bm.packageFilename = filename
	log.Printf("Package filename updated to: %s", bm.packageFilename)
}
