# Design Document: NexusHub Service Orchestrator

**Author:** System Architecture Team  
**Date:** 2025-06-24  
**Status:** Approved for Implementation

**Related Design Documents:**
- [Process Manager](processes.md) - Application lifecycle management
- [HTTPS Proxy](httpsproxy.md) - Request routing and SSL termination  
- [KrunClient](krunclient.md) - Virtualized application execution

**Technical Specification:** [spec/nexushub.md](../spec/nexushub.md)

## 1. Problem Statement

The NexusHub platform requires a unified orchestration service that coordinates multiple complex subsystems to provide a robust application hosting environment. The system must integrate process management, network routing, security, and application lifecycle management into a cohesive service that can be deployed and managed as a single unit.

Current challenges include:

- **Service Coordination**: Multiple services need precise startup sequencing and shutdown coordination
- **Configuration Management**: Static development configurations must coexist with dynamic production configurations
- **Security Integration**: Inter-service communication requires secure authentication and authorization
- **Operational Complexity**: Monitoring and debugging across multiple services requires unified logging and error handling
- **Development Experience**: Local development workflows need simplified configuration while maintaining production fidelity

Without a central orchestrator, deploying and managing the NexusHub platform requires complex external coordination and is prone to race conditions and configuration drift.

## 2. Goals and Non-Goals

### 2.1 Goals

- **Unified Service Management**: Single binary that orchestrates all NexusHub components
- **Graceful Lifecycle Management**: Coordinated startup and shutdown sequences that prevent data loss
- **Security by Design**: Secure inter-service communication with generated secrets and minimal attack surface
- **Operational Simplicity**: Single service to monitor, deploy, and debug with unified logging
- **Development Friendly**: Support both static development configurations and dynamic production configurations
- **Production Ready**: Robust error handling, graceful degradation, and comprehensive observability

### 2.2 Non-Goals

- **Distributed Deployment**: Single-node deployment only, not designed for multi-node coordination
- **External Service Discovery**: Direct integration with existing orchestration platforms (Docker, Kubernetes)
- **Legacy Application Support**: Optimized specifically for NexusHub's application architecture
- **Hot Reloading**: Configuration changes require service restart for simplicity and reliability

## 3. Success Metrics

- **Startup Time**: Complete system initialization in <10 seconds
- **Availability**: >99.9% uptime during normal operation
- **Recovery Time**: <30 seconds from component failure to full recovery
- **Resource Efficiency**: <5% CPU overhead for orchestration activities
- **Error Rate**: <0.1% request failures due to orchestration issues

## 4. High-Level Architecture

### 4.1 System Overview

The NexusHub Service Orchestrator follows a centralized coordination pattern where a single main process manages the lifecycle of multiple subsystem components:

```
┌─────────────────────────────────────────────────────────────┐
│                     NexusHub Orchestrator                   │
├─────────────────────────────────────────────────────────────┤
│  Signal Handler  │  Logger  │  Internal Secret Manager     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌─────────────┐  ┌──────────────────┐   │
│  │   Process    │  │   HTTPS     │  │   Admin Instance │   │
│  │   Manager    │  │   Proxy     │  │   Provider       │   │
│  │              │  │             │  │                  │   │
│  └──────────────┘  └─────────────┘  └──────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Application    │  │  Incoming HTTPS │  │  Admin Service  │
│  Instances      │  │  Requests       │  │  API Calls      │
│  (KrunClient)   │  │                 │  │                 │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### 4.2 Component Integration

The orchestrator integrates three major subsystems, each with specific responsibilities:

1. **Process Manager** ([design/processes.md](processes.md))
   - Manages application instance lifecycle using declarative desired state
   - Monitors application health and performs automatic recovery
   - Allocates dynamic ports and manages resource cleanup

2. **HTTPS Proxy** ([design/httpsproxy.md](httpsproxy.md))  
   - Terminates SSL connections and routes requests based on hostname
   - Integrates with Process Manager for dynamic backend discovery
   - Provides secure external access to managed applications

3. **Admin Instance Provider**
   - Polls admin service for dynamic application configurations
   - Merges static development configurations with dynamic production configurations
   - Provides real-time configuration updates to Process Manager

## 5. Detailed Design

### 5.1 Service Initialization Sequence

The orchestrator follows a carefully designed initialization sequence to ensure dependencies are satisfied:

1. **Foundation Setup**
   - Initialize structured JSON logging with debug level
   - Generate cryptographically secure internal secret for inter-service authentication
   - Detect project root directory for relative path resolution

2. **Static Configuration**
   - Define static application configurations for critical services (login, admin)
   - Initialize AdminInstanceProvider with static configs and internal secret
   - Ensure admin service configuration is available for subsequent dynamic polling

3. **Process Manager Initialization**
   - Initialize PortManager with production port range (10000-19999)
   - Configure ProcessManager with health monitoring and restart policies
   - Set subprocess working directory to project root for consistent execution environment

4. **Proxy Initialization**  
   - Configure HTTPS proxy with SSL certificate paths and listen address
   - Connect proxy to ProcessManager for hostname-to-backend resolution
   - Start proxy in dedicated goroutine to avoid blocking main thread

5. **Service Coordination**
   - Start ProcessManager reconciliation in main thread (blocking)
   - Use first reconcile callback to start AdminInstanceProvider only after static apps are running
   - Handle graceful shutdown through signal handling

### 5.2 Configuration Strategy

The system supports both static and dynamic configuration through a layered approach:

**Static Configuration (Development)**
- Hardcoded application definitions in main.go for critical services
- Enables local development without external dependencies
- Includes debug port configuration for Vite development server integration

**Dynamic Configuration (Production)**  
- AdminInstanceProvider polls admin service for live configuration updates
- Started only after static applications are confirmed running
- Merges with static configuration, with dynamic taking precedence

**Shared Configuration**
- Internal secret generated at startup and shared across all components
- SSL certificate paths configurable but defaults to `dist/certs/`
- Port ranges and timeouts configurable through ProcessManager config struct

### 5.3 Security Architecture

Security is implemented through multiple layers:

**Inter-Service Authentication**
- Unique internal secret generated at startup using `uuid.New()`
- Secret shared between ProcessManager, HTTPS Proxy, and AdminInstanceProvider
- Prevents unauthorized communication between components

**Network Security**
- HTTPS-only external communication through proxy
- Internal HTTP communication between components on localhost
- Dynamic port allocation prevents port conflicts and reduces attack surface

**Process Isolation**
- Applications run in KrunClient virtual machines (see [design/krunclient.md](krunclient.md))
- Each application instance isolated with dedicated filesystem and network namespace
- Minimal privilege escalation through subprocess execution

### 5.4 Error Handling and Resilience

The orchestrator implements comprehensive error handling:

**Initialization Failures**
- Immediate exit with error code 1 on critical failures
- Structured error logging with context for debugging
- Early validation of dependencies (certificates, directories, ports)

**Runtime Failures**  
- Component failures handled gracefully without cascading failures
- HTTPS proxy failures logged but don't terminate other services
- Process Manager handles individual application failures independently

**Shutdown Failures**
- Each component stop operation wrapped in error handling
- Continue shutdown sequence even if individual components fail to stop
- Final context cancellation ensures all goroutines receive termination signal

### 5.5 Observability and Monitoring

Comprehensive observability is built into the orchestrator:

**Structured Logging**
- JSON-formatted logs with consistent field naming
- Debug level logging for detailed operational visibility
- Component-specific log context for filtering and analysis

**Startup Coordination**
- First reconcile completion callback provides startup milestone
- Clear logging of each initialization step with timing
- Component readiness signals through callback mechanisms

**Shutdown Tracing**
- Detailed logging of shutdown sequence with component status
- Error logging for failed stop operations
- Final completion signal when all components have stopped

## 6. Implementation Considerations

### 6.1 Concurrency and Thread Safety

The orchestrator manages multiple concurrent operations:

- **Main Thread**: Runs ProcessManager reconciliation loop (blocking)
- **Signal Handler**: Dedicated goroutine for shutdown signal processing  
- **HTTPS Proxy**: Dedicated goroutine for HTTP server operation
- **Admin Provider**: Background polling goroutine for configuration updates

### 6.2 Resource Management

Efficient resource utilization is ensured through:

- **Port Allocation**: Centralized port management prevents conflicts
- **Memory Management**: No resource leaks through proper cleanup in shutdown sequence
- **File Handles**: SSL certificates and log files managed with proper cleanup

### 6.3 Development Workflow

The orchestrator supports streamlined development:

- **Static Configuration**: Immediate startup without external service dependencies
- **Debug Integration**: Admin service debug port forwarding to Vite development server
- **Hot Reloading**: Application changes reflected through ProcessManager restart logic

## 7. Future Extensibility

The architecture supports several future enhancements:

**Multi-Environment Support**
- Configuration profiles for development, staging, production environments
- Environment-specific SSL certificate and port configuration

**Enhanced Monitoring**
- Metrics export for Prometheus integration
- Health check endpoints for external monitoring systems

**Distributed Deployment**
- Leader election for multi-node coordination
- Shared state management for distributed Process Manager instances

**Advanced Security**
- Certificate rotation and management
- OAuth2 integration for admin service authentication

## 8. Migration and Deployment

### 8.1 Deployment Requirements

- SSL certificates available at `dist/certs/server.crt` and `dist/certs/server.key`
- Project directory structure with `dist/` containing application binaries
- Network access to admin service API for dynamic configuration

### 8.2 Operational Procedures

**Service Startup**
```bash
cd /path/to/nexushub
./nexushub/cmd/serve/main
```

**Graceful Shutdown**
```bash
# Send SIGTERM for graceful shutdown
kill -TERM $PID

# Send SIGINT (Ctrl+C) for interactive shutdown
```

**Configuration Validation**
- Verify SSL certificates exist and are readable
- Confirm port ranges are available and not conflicting
- Test admin service connectivity for dynamic configuration

This design provides a robust foundation for the NexusHub service orchestrator that balances operational simplicity with production readiness while maintaining clear separation of concerns across integrated subsystems.
