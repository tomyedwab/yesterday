# Technical Product Specification: NexusHub Service Orchestrator

**Reference Design Document:** [design/nexushub.md](../design/nexushub.md)

## Introduction

This specification defines the main orchestration service for the NexusHub platform, implemented as a Go binary that coordinates multiple subsystems to provide a comprehensive application hosting environment. The service integrates process management, HTTPS reverse proxy, and application lifecycle management into a unified system.

The main service orchestrator resides in `nexushub/cmd/serve/main.go` and serves as the central entry point that initializes and coordinates all NexusHub components including the process manager, HTTPS proxy, and application provisioning systems.

**Related Specifications:**
- [Process Manager](processes.md) - Application instance lifecycle management
- [HTTPS Proxy](httpsproxy.md) - Request routing and SSL termination
- [KrunClient](krunclient.md) - Virtualized application execution

## Task `nexushub-main-initialization`: Service Initialization and Bootstrap
**Reference:** design/nexushub.md  
**Implementation status:** Completed  
**Files:** `nexushub/cmd/serve/main.go` (168 lines)

**Details:**
- Initialize structured JSON logging with debug level output via `slog` package
- Generate unique internal secret using `uuid.New().String()` for secure inter-service communication
- Set up project root directory detection for subprocess execution context
- Configure graceful shutdown signal handling for SIGINT and SIGTERM
- Exit with appropriate error codes on initialization failures

## Task `nexushub-static-apps`: Static Application Configuration
**Reference:** design/nexushub.md  
**Implementation status:** Completed  
**Files:** `nexushub/cmd/serve/main.go` (lines 28-44)

**Details:**
- Define static application configurations for critical services:
  - Login service: `login.yesterday.localhost:8443` with instance ID `3bf3e3c0-6e51-482a-b180-00f6aa568ee9`
  - Admin service: `admin.yesterday.localhost:8443` with instance ID `18736e4f-93f9-4606-a7be-863c7986ea5b`
- Configure static paths: `dist/github.com/tomyedwab/yesterday/apps/{login,admin}/static`
- Configure binary paths: `dist/github.com/tomyedwab/yesterday/apps/{login,admin}/`
- Set debug port for admin service (5173 for Vite development server)
- Initialize `AdminInstanceProvider` with static configuration and internal secret

## Task `nexushub-process-manager`: Process Manager Integration
**Reference:** design/nexushub.md  
**Implementation status:** Completed  
**Files:** `nexushub/cmd/serve/main.go` (lines 46-75)

**Details:**
- Initialize `PortManager` with port range 10000-19999 for dynamic allocation
- Configure `ProcessManager` with production-ready parameters:
  - Health check interval: 10 seconds
  - Health check timeout: 3 seconds  
  - Consecutive failures threshold: 2
  - Restart backoff: 2 seconds initial, 15 seconds maximum
  - Graceful shutdown period: 5 seconds
- Set subprocess working directory to project root
- Implement first reconcile completion callback to coordinate admin service startup
- Start admin instance provider polling after static applications are running

## Task `nexushub-https-proxy`: HTTPS Proxy Integration  
**Reference:** design/nexushub.md  
**Implementation status:** Completed  
**Files:** `nexushub/cmd/serve/main.go` (lines 125-148)

**Details:**
- Configure HTTPS proxy with listen address `:8443`
- Set SSL certificate paths: `dist/certs/server.crt` and `dist/certs/server.key`
- Initialize proxy with internal secret and process manager reference for hostname resolution
- Start proxy server in dedicated goroutine with error handling
- Handle `http.ErrServerClosed` as normal shutdown signal
- Log proxy startup, operation, and shutdown events

## Task `nexushub-graceful-shutdown`: Graceful Shutdown Orchestration
**Reference:** design/nexushub.md  
**Implementation status:** Completed  
**Files:** `nexushub/cmd/serve/main.go` (lines 82-123)

**Details:**
- Implement signal handler goroutine monitoring SIGINT and SIGTERM
- Orchestrate shutdown sequence:
  1. Stop HTTPS proxy server first to stop accepting new requests
  2. Stop admin instance provider to halt dynamic configuration changes
  3. Stop process manager to gracefully terminate all managed processes
  4. Cancel main context to signal all goroutines to terminate
- Log each shutdown step with appropriate error handling
- Wait for all components to complete shutdown before main function exit
- Ensure proxy variable initialization check to handle early shutdown scenarios

## Task `nexushub-service-coordination`: Inter-Service Coordination
**Reference:** design/nexushub.md  
**Implementation status:** Completed  
**Files:** `nexushub/cmd/serve/main.go` (lines 77-81, 155-168)

**Details:**
- Implement first reconcile callback to ensure admin service is available before starting dynamic configuration
- Use shared internal secret for secure communication between components
- Coordinate startup sequence: static apps → admin provider → dynamic configuration
- Block main thread on process manager execution while allowing signal handling
- Ensure proper context cancellation propagates to all running services
- Implement final wait on context completion to ensure clean shutdown

## Task `nexushub-configuration`: Service Configuration Management
**Reference:** design/nexushub.md  
**Implementation status:** Completed  
**Files:** `nexushub/cmd/serve/main.go` (lines 55-74)

**Details:**
- Use struct-based configuration for `ProcessManager` with explicit parameter setting
- Configure health monitoring with frequent checks suitable for demonstration environment
- Set aggressive restart parameters for quick recovery during development
- Use project root as subprocess working directory for relative path resolution
- Placeholder SSL certificate configuration with clear TODO for production deployment
- Environment-aware port allocation ranges that don't conflict with development servers

## Task `nexushub-error-handling`: Error Handling and Resilience
**Reference:** design/nexushub.md  
**Implementation status:** Completed  
**Files:** `nexushub/cmd/serve/main.go` (throughout)

**Details:**
- Immediate exit with error code 1 on critical initialization failures
- Structured error logging with context for all failure scenarios
- Handle HTTP server shutdown errors distinctly from startup errors
- Non-blocking error handling for proxy server to prevent main thread blocking
- Graceful degradation when components fail to stop during shutdown
- Nil pointer checks before calling stop methods during shutdown sequence
