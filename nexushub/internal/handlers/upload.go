// Package handlers implements the NexusHub debug upload API endpoints for chunked file uploads.
//
// This module provides REST API endpoints for handling chunked file uploads as part of the 
// NexusDebug development workflow, including upload progress tracking and file integrity validation.
//
// Reference: spec/nexushub.md - Task nexushub-debug-upload
package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// processUploadChunk processes an individual chunk and manages upload session state
func (h *DebugHandler) processUploadChunk(appID string, chunkIndex, totalChunks int, fileHash string, chunkData []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Get or create upload session
	uploadSession, exists := h.uploadSessions[appID]
	if !exists {
		uploadSession = &UploadSession{
			ApplicationID: appID,
			TotalChunks:   totalChunks,
			Chunks:        make(map[int]*UploadChunk),
			FileHash:      fileHash,
			Completed:     false,
			CreatedAt:     time.Now(),
		}
		h.uploadSessions[appID] = uploadSession
		h.logger.Info("Created new upload session", "appId", appID, "totalChunks", totalChunks)
	}

	// Validate session parameters
	if uploadSession.TotalChunks != totalChunks {
		return fmt.Errorf("total chunks mismatch: expected %d, got %d", uploadSession.TotalChunks, totalChunks)
	}
	if uploadSession.FileHash != fileHash {
		return fmt.Errorf("file hash mismatch: expected %s, got %s", uploadSession.FileHash, fileHash)
	}

	// Store the chunk
	uploadSession.mu.Lock()
	uploadSession.Chunks[chunkIndex] = &UploadChunk{
		ChunkIndex: chunkIndex,
		Data:       chunkData,
		Received:   true,
	}
	uploadSession.mu.Unlock()

	h.logger.Info("Chunk stored", "appId", appID, "chunkIndex", chunkIndex, "size", len(chunkData))

	// Check if all chunks are received
	if len(uploadSession.Chunks) == totalChunks {
		h.logger.Info("All chunks received, assembling file", "appId", appID)
		if err := h.assembleUploadedFile(appID, uploadSession); err != nil {
			return fmt.Errorf("failed to assemble uploaded file: %w", err)
		}
		uploadSession.Completed = true
	}

	return nil
}

// assembleUploadedFile assembles all chunks into the final package file
func (h *DebugHandler) assembleUploadedFile(appID string, session *UploadSession) error {
	// Create package file path
	packageFilename := fmt.Sprintf("%s-package.zip", appID)
	packagePath := filepath.Join(h.uploadDir, packageFilename)

	// Create the output file
	outFile, err := os.Create(packagePath)
	if err != nil {
		return fmt.Errorf("failed to create package file: %w", err)
	}
	defer outFile.Close()

	// Sort chunk indices to ensure correct order
	chunkIndices := make([]int, 0, len(session.Chunks))
	for index := range session.Chunks {
		chunkIndices = append(chunkIndices, index)
	}
	sort.Ints(chunkIndices)

	// Write chunks in order and calculate hash
	hasher := md5.New()
	for _, index := range chunkIndices {
		chunk := session.Chunks[index]
		if _, err := outFile.Write(chunk.Data); err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", index, err)
		}
		if _, err := hasher.Write(chunk.Data); err != nil {
			return fmt.Errorf("failed to hash chunk %d: %w", index, err)
		}
	}

	// Verify file hash
	calculatedHash := hex.EncodeToString(hasher.Sum(nil))
	if calculatedHash != session.FileHash {
		// Remove invalid file
		os.Remove(packagePath)
		return fmt.Errorf("file hash verification failed: expected %s, got %s", session.FileHash, calculatedHash)
	}

	// Update debug application with package path
	debugApp := h.debugApps[appID]
	debugApp.PackagePath = packagePath

	h.logger.Info("Package assembled successfully", 
		"appId", appID, "packagePath", packagePath, "hash", calculatedHash)

	return nil
}

// HandleUploadStatus handles GET /debug/application/{id}/upload/status for upload progress
func (h *DebugHandler) HandleUploadStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract application ID from URL path: /debug/application/{id}/upload/status
	path := strings.TrimPrefix(r.URL.Path, "/debug/application/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[1] != "upload" || parts[2] != "status" {
		http.Error(w, "Invalid upload status URL format", http.StatusBadRequest)
		return
	}
	appID := parts[0]

	if appID == "" {
		http.Error(w, "Missing application ID", http.StatusBadRequest)
		return
	}

	// Get upload session
	h.mu.RLock()
	uploadSession, exists := h.uploadSessions[appID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Upload session not found", http.StatusNotFound)
		return
	}

	// Calculate progress
	uploadSession.mu.RLock()
	receivedChunks := len(uploadSession.Chunks)
	totalChunks := uploadSession.TotalChunks
	completed := uploadSession.Completed
	fileHash := uploadSession.FileHash
	uploadSession.mu.RUnlock()

	progress := float64(receivedChunks) / float64(totalChunks) * 100.0

	status := &UploadStatus{
		ApplicationID:  appID,
		TotalChunks:    totalChunks,
		ReceivedChunks: receivedChunks,
		Progress:       progress,
		Completed:      completed,
		FileHash:       fileHash,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Error("Failed to encode upload status response", "error", err)
	}
}
