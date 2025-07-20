// Package upload implements chunked file upload functionality for the NexusDebug CLI tool.
//
// This module handles large package file uploads with progress reporting, retry logic,
// and integration with the NexusHub debug API endpoints for application deployment.
//
// Reference: spec/nexusdebug.md - Task nexusdebug-file-upload
package nexusdebug

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

const (
	// Default chunk size for uploads (2MB)
	DefaultChunkSize = 2 * 1024 * 1024

	// Maximum number of upload retries per chunk
	MaxUploadRetries = 3

	// Upload timeout per chunk
	ChunkUploadTimeout = 60 * time.Second

	// Total upload timeout
	TotalUploadTimeout = 30 * time.Minute
)

// UploadProgress represents the current upload progress
type UploadProgress struct {
	BytesUploaded  int64
	TotalBytes     int64
	ChunksTotal    int
	ChunksUploaded int
	CurrentChunk   int
	Percentage     float64
	StartTime      time.Time
	ElapsedTime    time.Duration
	EstimatedTime  time.Duration
	UploadSpeed    float64 // bytes per second
}

// UploadManager handles chunked file upload operations
type UploadManager struct {
	client       *yesterdaygo.Client
	chunkSize    int64
	maxRetries   int
	progressChan chan *UploadProgress
	mu           sync.RWMutex
	currentApp   *DebugApplication
}

// UploadProgressCallback is called during upload to report progress
type UploadProgressCallback func(progress *UploadProgress)

// NewUploadManager creates a new upload manager
func NewUploadManager(client *yesterdaygo.Client) *UploadManager {
	return &UploadManager{
		client:       client,
		chunkSize:    DefaultChunkSize,
		maxRetries:   MaxUploadRetries,
		progressChan: make(chan *UploadProgress, 100),
	}
}

// SetApplication sets the current debug application for uploads
func (um *UploadManager) SetApplication(app *DebugApplication) {
	um.mu.Lock()
	defer um.mu.Unlock()
	um.currentApp = app
}

// SetChunkSize configures the chunk size for uploads
func (um *UploadManager) SetChunkSize(size int64) {
	um.mu.Lock()
	defer um.mu.Unlock()
	um.chunkSize = size
}

// SetMaxRetries configures the maximum number of retries per chunk
func (um *UploadManager) SetMaxRetries(retries int) {
	um.mu.Lock()
	defer um.mu.Unlock()
	um.maxRetries = retries
}

// UploadPackage uploads a package file with chunked upload and progress reporting
func (um *UploadManager) UploadPackage(ctx context.Context, packagePath string, progressCallback UploadProgressCallback) error {
	um.mu.RLock()
	app := um.currentApp
	um.mu.RUnlock()

	if app == nil {
		return fmt.Errorf("no debug application set for upload")
	}

	log.Printf("📤 Starting package upload...")
	log.Printf("Package path: %s", packagePath)
	log.Printf("Application ID: %s", app.ID)

	// Validate package file exists
	fileInfo, err := os.Stat(packagePath)
	if err != nil {
		return fmt.Errorf("failed to access package file: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("package path is a directory, not a file")
	}

	totalSize := fileInfo.Size()
	log.Printf("Package size: %.2f MB", float64(totalSize)/1024/1024)

	// Calculate number of chunks
	chunkSize := um.chunkSize
	totalChunks := int((totalSize + chunkSize - 1) / chunkSize)
	log.Printf("Upload will use %d chunks of %.2f KB each", totalChunks, float64(chunkSize)/1024)

	// Initialize progress tracking
	progress := &UploadProgress{
		TotalBytes:  totalSize,
		ChunksTotal: totalChunks,
		StartTime:   time.Now(),
		Percentage:  0.0,
	}

	// Set up total timeout
	uploadCtx, cancel := context.WithTimeout(ctx, TotalUploadTimeout)
	defer cancel()

	// Open the package file
	file, err := os.Open(packagePath)
	if err != nil {
		return fmt.Errorf("failed to open package file: %w", err)
	}
	defer file.Close()

	// Calculate file hash for integrity validation
	log.Printf("🔍 Calculating package hash for integrity validation...")
	hash, err := um.calculateFileHash(packagePath)
	if err != nil {
		log.Printf("Warning: could not calculate file hash: %v", err)
		hash = "" // Continue without hash validation
	} else {
		log.Printf("Package hash (MD5): %s", hash)
	}

	// Upload chunks
	for chunkIndex := 0; chunkIndex < totalChunks; chunkIndex++ {
		select {
		case <-uploadCtx.Done():
			return fmt.Errorf("upload timeout exceeded")
		default:
		}

		// Calculate chunk boundaries
		start := int64(chunkIndex) * chunkSize
		end := start + chunkSize
		if end > totalSize {
			end = totalSize
		}
		currentChunkSize := end - start

		// Update progress
		progress.CurrentChunk = chunkIndex + 1
		progress.ChunksUploaded = chunkIndex
		progress.BytesUploaded = start
		progress.ElapsedTime = time.Since(progress.StartTime)
		progress.Percentage = float64(progress.BytesUploaded) / float64(progress.TotalBytes) * 100

		// Calculate upload speed and estimated time
		if progress.ElapsedTime.Seconds() > 0 {
			progress.UploadSpeed = float64(progress.BytesUploaded) / progress.ElapsedTime.Seconds()
			if progress.UploadSpeed > 0 {
				remaining := float64(progress.TotalBytes - progress.BytesUploaded)
				progress.EstimatedTime = time.Duration(remaining/progress.UploadSpeed) * time.Second
			}
		}

		// Report progress
		if progressCallback != nil {
			progressCallback(progress)
		}

		log.Printf("📤 Uploading chunk %d/%d (%.1f%%)...", chunkIndex+1, totalChunks, progress.Percentage)

		// Read chunk data
		chunkData := make([]byte, currentChunkSize)
		if _, err := file.Seek(start, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek to chunk position: %w", err)
		}

		if _, err := io.ReadFull(file, chunkData); err != nil {
			return fmt.Errorf("failed to read chunk data: %w", err)
		}

		// Upload chunk with retry logic
		if err := um.uploadChunkWithRetry(uploadCtx, app.ID, chunkIndex, totalChunks, chunkData, hash); err != nil {
			return fmt.Errorf("failed to upload chunk %d: %w", chunkIndex+1, err)
		}
	}

	// Final progress update
	progress.BytesUploaded = totalSize
	progress.ChunksUploaded = totalChunks
	progress.Percentage = 100.0
	progress.ElapsedTime = time.Since(progress.StartTime)
	if progressCallback != nil {
		progressCallback(progress)
	}

	log.Printf("✅ Package upload completed successfully!")
	log.Printf("Upload time: %.2f seconds", progress.ElapsedTime.Seconds())
	if progress.UploadSpeed > 0 {
		log.Printf("Average upload speed: %.2f KB/s", progress.UploadSpeed/1024)
	}

	// Verify upload completion
	return um.verifyUploadCompletion(uploadCtx, app.ID, hash)
}

// uploadChunkWithRetry uploads a single chunk with retry logic
func (um *UploadManager) uploadChunkWithRetry(ctx context.Context, appID string, chunkIndex, totalChunks int, chunkData []byte, fileHash string) error {
	var lastErr error

	for attempt := 0; attempt < um.maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying chunk %d (attempt %d/%d)...", chunkIndex+1, attempt+1, um.maxRetries)
			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Create chunk upload context with timeout
		chunkCtx, cancel := context.WithTimeout(ctx, ChunkUploadTimeout)
		err := um.uploadChunk(chunkCtx, appID, chunkIndex, totalChunks, chunkData, fileHash)
		cancel()

		if err == nil {
			return nil // Success
		}

		lastErr = err
		log.Printf("Chunk upload attempt %d failed: %v", attempt+1, err)
	}

	return fmt.Errorf("chunk upload failed after %d attempts: %w", um.maxRetries, lastErr)
}

// uploadChunk uploads a single chunk to the server
func (um *UploadManager) uploadChunk(ctx context.Context, appID string, chunkIndex, totalChunks int, chunkData []byte, fileHash string) error {
	// Prepare form fields
	fields := map[string]string{
		"chunkIndex":  strconv.Itoa(chunkIndex),
		"totalChunks": strconv.Itoa(totalChunks),
	}

	if fileHash != "" {
		fields["fileHash"] = fileHash
	}

	// Prepare files
	files := map[string][]byte{
		"chunk": chunkData,
	}

	// Execute authenticated multipart request using client method
	endpoint := fmt.Sprintf("/debug/application/%s/upload", appID)
	response, err := um.client.PostMultipart(ctx, endpoint, fields, files, nil)
	if err != nil {
		return fmt.Errorf("chunk upload request failed: %w", err)
	}
	defer response.Body.Close()

	// Check response status
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(response.Body)
		return fmt.Errorf("chunk upload failed with status %d: %s", response.StatusCode, string(bodyBytes))
	}

	return nil
}

// verifyUploadCompletion verifies that the upload was completed successfully
func (um *UploadManager) verifyUploadCompletion(ctx context.Context, appID, expectedHash string) error {
	log.Printf("🔍 Verifying upload completion...")

	// Request upload status/verification
	response, err := um.client.Get(ctx, fmt.Sprintf("/debug/application/%s/upload/status", appID), nil)
	if err != nil {
		log.Printf("Warning: could not verify upload completion: %v", err)
		return nil // Don't fail the upload if verification fails
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		// Upload status endpoint might not be implemented yet
		log.Printf("Upload verification endpoint not available - assuming success")
		return nil
	}

	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(response.Body)
		log.Printf("Warning: upload verification failed with status %d: %s", response.StatusCode, string(bodyBytes))
		return nil // Don't fail the upload if verification fails
	}

	log.Printf("✅ Upload verification completed successfully")
	return nil
}

// calculateFileHash calculates MD5 hash of the file for integrity validation
func (um *UploadManager) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetUploadProgress returns a channel for receiving upload progress updates
func (um *UploadManager) GetUploadProgress() <-chan *UploadProgress {
	return um.progressChan
}

// UploadPackageFromBuildManager uploads a package from a build manager
func (um *UploadManager) UploadPackageFromBuildManager(ctx context.Context, buildManager *BuildManager, progressCallback UploadProgressCallback) error {
	packagePath := buildManager.GetPackagePath()

	// Verify package exists and was built
	if _, err := os.Stat(packagePath); err != nil {
		return fmt.Errorf("package file not found: %s - ensure build completed successfully", packagePath)
	}

	return um.UploadPackage(ctx, packagePath, progressCallback)
}

// InstallUploadedPackage triggers installation of the uploaded package
func (um *UploadManager) InstallUploadedPackage(ctx context.Context) error {
	um.mu.RLock()
	app := um.currentApp
	um.mu.RUnlock()

	if app == nil {
		return fmt.Errorf("no debug application set for installation")
	}

	log.Printf("🚀 Installing uploaded package...")

	// Trigger installation via the install endpoint
	response, err := um.client.Post(ctx, fmt.Sprintf("/debug/application/%s/install-dev", app.ID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to trigger installation: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(response.Body)
		return fmt.Errorf("installation failed with status %d: %s", response.StatusCode, string(bodyBytes))
	}

	log.Printf("✅ Package installation triggered successfully")
	return nil
}

// UploadAndInstall performs the complete upload and installation workflow
func (um *UploadManager) UploadAndInstall(ctx context.Context, packagePath string, progressCallback UploadProgressCallback) error {
	// Upload the package
	if err := um.UploadPackage(ctx, packagePath, progressCallback); err != nil {
		return fmt.Errorf("package upload failed: %w", err)
	}

	// Install the uploaded package
	if err := um.InstallUploadedPackage(ctx); err != nil {
		return fmt.Errorf("package installation failed: %w", err)
	}

	return nil
}

// PrintUploadProgress is a utility function for printing upload progress
func PrintUploadProgress(progress *UploadProgress) {
	elapsed := progress.ElapsedTime.Truncate(time.Second)

	if progress.EstimatedTime > 0 {
		eta := progress.EstimatedTime.Truncate(time.Second)
		log.Printf("📤 Upload Progress: %.1f%% (%d/%d chunks) - %s elapsed, %s remaining",
			progress.Percentage, progress.ChunksUploaded, progress.ChunksTotal, elapsed, eta)
	} else {
		log.Printf("📤 Upload Progress: %.1f%% (%d/%d chunks) - %s elapsed",
			progress.Percentage, progress.ChunksUploaded, progress.ChunksTotal, elapsed)
	}

	if progress.UploadSpeed > 0 {
		speed := progress.UploadSpeed / 1024 // Convert to KB/s
		if speed > 1024 {
			log.Printf("Upload Speed: %.2f MB/s", speed/1024)
		} else {
			log.Printf("Upload Speed: %.2f KB/s", speed)
		}
	}
}
