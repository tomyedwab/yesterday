// Simple test server to verify the NexusHub debug upload functionality
package main

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

func main() {
	// Test data for upload
	testData := []byte("This is test package data for chunked upload verification!")
	hash := md5.Sum(testData)
	fileHash := hex.EncodeToString(hash[:])
	
	chunkSize := 20
	totalChunks := (len(testData) + chunkSize - 1) / chunkSize
	
	fmt.Printf("Testing chunked upload with %d chunks\n", totalChunks)
	fmt.Printf("File hash: %s\n", fileHash)
	
	// First, create a debug application
	appID := "test-app-" + strconv.FormatInt(time.Now().Unix(), 10)
	if err := createDebugApp(appID); err != nil {
		log.Fatalf("Failed to create debug app: %v", err)
	}
	
	// Upload chunks
	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(testData) {
			end = len(testData)
		}
		
		chunkData := testData[start:end]
		
		fmt.Printf("Uploading chunk %d/%d (%d bytes)\n", i+1, totalChunks, len(chunkData))
		
		if err := uploadChunk(appID, i, totalChunks, fileHash, chunkData); err != nil {
			log.Fatalf("Failed to upload chunk %d: %v", i, err)
		}
		
		// Check status after each chunk
		if err := checkUploadStatus(appID); err != nil {
			log.Printf("Warning: failed to check status: %v", err)
		}
		
		time.Sleep(100 * time.Millisecond)
	}
	
	fmt.Println("Upload test completed successfully!")
}

func createDebugApp(appID string) error {
	reqBody := fmt.Sprintf(`{
		"appId": "%s",
		"displayName": "Test Upload App",
		"hostName": "test.localhost",
		"dbName": "test_db"
	}`, appID)
	
	resp, err := http.Post("https://localhost:8443/debug/application", "application/json", bytes.NewBuffer([]byte(reqBody)))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	
	fmt.Printf("Created debug application: %s\n", appID)
	return nil
}

func uploadChunk(appID string, chunkIndex, totalChunks int, fileHash string, chunkData []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	
	// Add form fields
	writer.WriteField("chunkIndex", strconv.Itoa(chunkIndex))
	writer.WriteField("totalChunks", strconv.Itoa(totalChunks))
	writer.WriteField("fileHash", fileHash)
	
	// Add chunk file
	part, err := writer.CreateFormFile("chunk", "chunk.dat")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	
	if _, err := part.Write(chunkData); err != nil {
		return fmt.Errorf("failed to write chunk data: %w", err)
	}
	
	writer.Close()
	
	// Make request
	url := fmt.Sprintf("https://localhost:8443/debug/application/%s/upload", appID)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	// Skip TLS verification for testing
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func checkUploadStatus(appID string) error {
	url := fmt.Sprintf("https://localhost:8443/debug/application/%s/upload/status", appID)
	
	// Skip TLS verification for testing
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Upload status: %s\n", string(body))
	}
	
	return nil
}
