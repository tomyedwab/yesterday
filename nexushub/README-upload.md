# NexusHub Debug Upload API

This document describes the chunked file upload functionality implemented for the NexusHub debug application system.

## Overview

The upload API supports chunked file uploads with the following features:
- **Chunked Upload**: Large files can be split into smaller chunks and uploaded in sequence
- **Progress Tracking**: Real-time upload progress monitoring via status endpoint
- **File Integrity**: MD5 hash verification ensures uploaded files are complete and valid
- **Session Management**: Upload sessions track chunk state and prevent corruption
- **Thread Safety**: Concurrent uploads for different applications are supported

## API Endpoints

### 1. Upload Chunk
```
POST /debug/application/{id}/upload
```

**Request**: Multipart form data with the following fields:
- `chunkIndex` (form field): Zero-based index of the chunk (e.g., "0", "1", "2")
- `totalChunks` (form field): Total number of chunks in the upload (e.g., "5")
- `fileHash` (form field): MD5 hash of the complete file (hex-encoded)
- `chunk` (file field): Binary data of the chunk

**Response**: JSON object indicating upload progress
- `200 OK`: Chunk received successfully
- `400 Bad Request`: Invalid parameters or malformed request
- `404 Not Found`: Debug application not found
- `500 Internal Server Error`: Server-side processing error

**Example Response (chunk received)**:
```json
{
  "chunkReceived": true,
  "chunkIndex": 2,
  "totalChunks": 5,
  "progress": 60.0,
  "completed": false
}
```

**Example Response (upload completed)**:
```json
{
  "chunkReceived": true,
  "chunkIndex": 4,
  "totalChunks": 5,
  "progress": 100.0,
  "completed": true,
  "packagePath": "/tmp/nexushub-uploads/app123-package.zip"
}
```

### 2. Upload Status
```
GET /debug/application/{id}/upload/status
```

**Response**: JSON object with upload progress information
- `200 OK`: Status retrieved successfully
- `404 Not Found`: Upload session or debug application not found

**Example Response**:
```json
{
  "applicationId": "app123",
  "totalChunks": 5,
  "receivedChunks": 3,
  "progress": 60.0,
  "completed": false,
  "fileHash": "5d41402abc4b2a76b9719d911017c592"
}
```

## Upload Workflow

1. **Create Debug Application**: First create a debug application using `POST /debug/application`
2. **Calculate File Hash**: Calculate MD5 hash of the complete file to be uploaded
3. **Split File**: Divide the file into chunks (recommended size: 1-10MB per chunk)
4. **Upload Chunks**: Send each chunk via `POST /debug/application/{id}/upload`
5. **Monitor Progress**: Optionally check upload status via `GET /debug/application/{id}/upload/status`
6. **File Assembly**: Server automatically assembles chunks when all are received
7. **Hash Verification**: Server verifies the assembled file matches the provided hash
8. **Package Ready**: Successfully uploaded package is ready for installation

## Error Handling

- **Hash Mismatch**: If the assembled file's hash doesn't match the provided hash, the upload fails
- **Missing Chunks**: Uploads remain incomplete until all chunks are received
- **Invalid Parameters**: Chunk metadata must be consistent across all chunks in a session
- **Session Cleanup**: Failed uploads are automatically cleaned up (implementation pending)

## Security Considerations

- Upload sessions are tied to existing debug applications
- File uploads are stored in a secure upload directory
- Authentication and authorization are handled at the proxy level
- Temporary files are cleaned up after processing

## Implementation Details

### Upload Session Management
- Each debug application can have one active upload session
- Sessions track received chunks, total chunks, file hash, and completion status
- Thread-safe access using read-write mutexes

### File Assembly Process
1. Sort chunks by index to ensure correct order
2. Write chunks sequentially to create the complete file
3. Calculate MD5 hash of the assembled file
4. Compare with expected hash from upload session
5. Update debug application with package file path
6. Mark upload session as completed

### Storage
- Uploaded packages are stored in: `/tmp/nexushub-uploads/`
- File naming pattern: `{applicationId}-package.zip`
- Files persist until debug application is deleted

## Testing

A test client is available at `cmd/test-upload/main.go` that demonstrates:
- Creating a debug application
- Splitting test data into chunks
- Uploading chunks with progress monitoring
- Verifying successful upload completion

To run the test:
```bash
cd nexushub
go build -o test-upload ./cmd/test-upload
./test-upload
```

## Integration with NexusDebug CLI

The upload API is designed to work with the NexusDebug CLI tool's chunked upload feature. The CLI tool should:

1. Calculate file hash before upload
2. Split large files into manageable chunks
3. Upload chunks sequentially with retry logic
4. Monitor upload progress
5. Verify upload completion before proceeding

## Future Enhancements

- **Upload Resumption**: Support resuming interrupted uploads
- **Concurrent Chunks**: Allow parallel chunk uploads for faster transfer
- **Session Cleanup**: Automatic cleanup of stale upload sessions
- **Size Limits**: Configurable limits on chunk size and total file size
- **Compression**: Optional compression of chunks during transfer
