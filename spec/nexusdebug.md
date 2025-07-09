# Technical Product Specification: NexusDebug CLI Tool

**Reference Design Document:** [design/nexusdebug.md](../design/nexusdebug.md)

## Introduction

This specification defines the NexusDebug CLI tool, a development utility that enables rapid iteration and debugging of Go server applications running within the NexusHub platform. The tool provides an automated workflow for building, deploying, and monitoring debug applications with hot-reload capabilities.

The CLI tool will be implemented in Go and located in the `nexusdebug/` directory. It leverages the Go client library at `clients/go/` for authentication and API communication with the NexusHub platform.

**Related Specifications:**
- [NexusHub Service Orchestrator](nexushub.md) - Target platform for debug applications
- [Go Client Library](clients/go.md) - Authentication and API communication

## Task `nexusdebug-cli-setup`: CLI Application Structure and Initialization
**Reference:** design/nexusdebug.md  
**Implementation status:** Completed  
**Files:** `nexusdebug/main.go`, `nexusdebug/go.mod`

**Details:**
- Create Go module structure for the CLI tool in `nexusdebug/` directory
- Initialize command-line argument parsing with the following parameters:
  - Admin app URL (required): Target NexusHub admin service URL
  - Application name (required): Used to generate AppID, DisplayName, and HostName
  - Build command (optional): Defaults to `make build`
  - Package filename (optional): Defaults to `dist/package.zip`
  - Static service URL (optional): For proxying frontend requests during development
- Implement CLI help text and usage information
- Set up structured logging for development workflow feedback
- Initialize configuration validation and error handling

## Task `nexusdebug-authentication`: Admin Service Authentication
**Reference:** design/nexusdebug.md  
**Implementation status:** Not Started  
**Files:** `nexusdebug/auth.go`

**Details:**
- Integrate with Go client library for authentication workflow
- Implement interactive username/password prompt using secure terminal input
- Execute login flow against `POST /public/login` endpoint via Admin app
- Handle authentication errors with clear user feedback
- Store authentication tokens for subsequent API requests
- Implement token refresh logic for long-running debug sessions
- Provide authentication status feedback to user

## Task `nexusdebug-application-management`: Debug Application Lifecycle
**Reference:** design/nexusdebug.md  
**Implementation status:** Not Started  
**Files:** `nexusdebug/application.go`

**Details:**
- Implement debug application creation via `POST /debug/application` endpoint
- Generate unique application identifiers from provided application name
- Configure debug application with appropriate metadata and hostname mapping
- Handle application creation conflicts and cleanup existing debug applications
- Implement application installation via `POST /debug/application/{id}/install` endpoint
- Monitor application startup and provide status feedback to user
- Implement application cleanup and removal on exit

## Task `nexusdebug-build-system`: Application Build and Package Management
**Reference:** design/nexusdebug.md  
**Implementation status:** Not Started  
**Files:** `nexusdebug/build.go`

**Details:**
- Execute configurable build command (default: `make build`) in application directory
- Monitor build process output and provide real-time feedback to user
- Validate build artifacts and package creation (default: `dist/package.zip`)
- Handle build failures with clear error reporting
- Implement build artifact cleanup between iterations
- Support custom build commands and package paths via CLI parameters
- Detect and validate package format and structure

## Task `nexusdebug-file-upload`: Chunked File Upload Implementation
**Reference:** design/nexusdebug.md  
**Implementation status:** Not Started  
**Files:** `nexusdebug/upload.go`

**Details:**
- Implement chunked file upload via `POST /debug/application/{id}/upload` endpoint
- Handle large package files with progress reporting
- Implement upload retry logic for network resilience
- Validate upload completion and package integrity
- Provide upload progress feedback to user
- Handle upload failures with retry mechanism
- Support resumable uploads for large packages

## Task `nexusdebug-monitoring`: Application Status and Log Monitoring
**Reference:** design/nexusdebug.md  
**Implementation status:** Not Started  
**Files:** `nexusdebug/monitor.go`

**Details:**
- Implement real-time log tailing via `GET /debug/application/{id}/logs` endpoint
- Monitor application status via `GET /debug/application/{id}/status` endpoint
- Display server logs with appropriate formatting and timestamps
- Implement log filtering and search capabilities
- Provide application health status indicators
- Handle log streaming interruptions with reconnection logic
- Support log level filtering and output formatting

## Task `nexusdebug-interactive-control`: User Input and Hot-Reload
**Reference:** design/nexusdebug.md  
**Implementation status:** Not Started  
**Files:** `nexusdebug/control.go`

**Details:**
- Implement non-blocking keyboard input detection
- Handle 'R' key press for rebuild and redeploy workflow:
  1. Stop current application instance
  2. Execute build command
  3. Upload new package
  4. Install and start updated application
  5. Resume log monitoring
- Handle 'Q' key press for graceful shutdown:
  1. Stop application instance
  2. Clean up debug application
  3. Close authentication session
  4. Exit CLI tool
- Provide clear user instructions and command feedback
- Implement graceful error handling during hot-reload operations

## Task `nexusdebug-error-handling`: Comprehensive Error Management
**Reference:** design/nexusdebug.md  
**Implementation status:** Not Started  
**Files:** `nexusdebug/errors.go`

**Details:**
- Define structured error types for different failure scenarios:
  - Authentication errors
  - Build process failures
  - Network and API communication errors
  - Application deployment failures
- Implement error recovery strategies for transient failures
- Provide clear error messages with actionable guidance for users
- Handle network connectivity issues with retry logic
- Implement timeout handling for long-running operations
- Log detailed error information for debugging while providing concise user feedback
- Support error reporting and diagnostic information collection

## Task `nexusdebug-integration-testing`: Testing and Quality Assurance
**Reference:** design/nexusdebug.md  
**Implementation status:** Not Started  
**Files:** `nexusdebug/*_test.go`

**Details:**
- Implement unit tests for all major CLI components
- Create integration tests with mock NexusHub API endpoints
- Test authentication flow with various error scenarios
- Validate build system integration with different project structures
- Test file upload functionality with various package sizes
- Implement end-to-end testing for complete debug workflow
