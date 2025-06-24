# Design Document: Go Process Manager

**Author:** System Architecture Team  
**Date:** 2025-06-24  
**Status:** Approved for Implementation

## 1. Problem Statement

The NexusHub system requires a robust process orchestration layer to manage multiple application instances dynamically. Each application instance runs as an isolated subprocess with its own configuration, database context, and network endpoint. The system must handle:

- **Dynamic Scaling**: Starting and stopping application instances based on demand
- **Configuration Management**: Supporting both static configurations for development and dynamic configurations from an admin service
- **Health Monitoring**: Detecting unhealthy processes and performing automatic recovery
- **Resource Management**: Efficient allocation of network ports and system resources
- **Integration**: Seamless integration with the HTTPS reverse proxy for request routing

Currently, manual process management is error-prone and doesn't scale. We need an automated solution that can reliably manage the lifecycle of multiple application instances while providing operational visibility and robustness.

## 2. Goals and Non-Goals

### 2.1 Goals

- **Declarative Process Management**: Manage processes based on declarative desired state rather than imperative commands
- **High Availability**: Automatically restart failed processes with intelligent backoff strategies
- **Dynamic Configuration**: Support real-time configuration changes from admin service with zero-downtime updates
- **Resource Efficiency**: Optimal allocation and cleanup of system resources (ports, memory, handles)
- **Operational Visibility**: Comprehensive logging and state tracking for debugging and monitoring
- **Developer Experience**: Simple static configuration overrides for local development workflows
- **Integration Ready**: Clean APIs for integration with reverse proxy and other system components

### 2.2 Non-Goals

- **Process Sandboxing**: Security isolation is handled at the OS/container level
- **Load Balancing**: Traffic distribution is handled by the reverse proxy layer
- **Persistent Storage**: Database and storage management is delegated to individual application instances
- **Cross-Node Orchestration**: This design focuses on single-node process management

## 3. High-Level Architecture

### 3.1 Core Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Admin Service  │    │  Static Config   │    │  HTTPS Proxy    │
│     (Remote)    │    │    (Local)       │    │                 │
└─────────┬───────┘    └─────────┬────────┘    └─────────┬───────┘
          │                      │                       │
          │ Config Updates       │ Dev Overrides         │ Route Queries
          ▼                      ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Process Manager                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Instance        │  │ Reconciliation  │  │ Health Monitor  │ │
│  │ Provider        │  │ Loop            │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Port Manager    │  │ Process State   │  │ API Interface   │ │
│  │                 │  │ Tracker         │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
          │                      │                       │
          │ Process Commands     │ Health Checks         │ State Queries
          ▼                      ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Subprocess A   │    │  Subprocess B   │    │  Subprocess C   │
│  Port: 30001    │    │  Port: 30002    │    │  Port: 30003    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 3.2 Component Responsibilities

- **Process Manager**: Central orchestrator responsible for lifecycle management and state reconciliation
- **Instance Provider**: Abstraction for fetching desired state from various sources (static config, admin service)
- **Reconciliation Loop**: Continuous synchronization between desired and actual process states
- **Health Monitor**: HTTP-based health checking with configurable intervals and failure thresholds
- **Port Manager**: Dynamic allocation and tracking of TCP ports for subprocess communication
- **Process State Tracker**: Thread-safe state management with atomic updates and queries

## 4. Detailed Design

### 4.1 Process Lifecycle State Machine

```
┌─────────────┐     start     ┌─────────────┐   health_ok   ┌─────────────┐
│   Unknown   │ ──────────────▶│  Starting   │ ─────────────▶│   Running   │
└─────────────┘               └─────────────┘               └─────────────┘
                                     │                              │
                               start_failed                   health_failed
                                     │                              │
                                     ▼                              ▼
┌─────────────┐    cleanup    ┌─────────────┐   restart     ┌─────────────┐
│   Stopped   │ ◀─────────────│   Failed    │ ─────────────▶│ Unhealthy   │
└─────────────┘               └─────────────┘               └─────────────┘
       ▲                             ▲                              │
       │                             │                         consecutive
    stop_ok                    stop_failed                      failures
       │                             │                              │
       │                      ┌─────────────┐                      │
       └──────────────────────│  Stopping   │ ◀────────────────────┘
                              └─────────────┘
```

### 4.2 Configuration Management Strategy

**Dual Source Architecture**: Support both static and dynamic configuration sources with a clear precedence model.

- **Static Configuration**: Local file-based or code-based configuration for development scenarios
- **Dynamic Configuration**: Admin service integration with event-driven updates and polling fallback
- **Override Mechanism**: Static configurations can override specific fields from dynamic sources

**Change Detection**: Event-based polling with exponential backoff on failures to minimize admin service load while maintaining responsiveness.

### 4.3 Resource Management Design

**Port Allocation Strategy**:
- **Range-Based Allocation**: Configurable port range (e.g., 30000-31000) for predictable resource usage
- **Availability Verification**: Actual socket binding to verify port availability before assignment
- **Round-Robin Assignment**: Efficient distribution across available ports to avoid clustering

**Process Resource Tracking**:
- **Memory Efficiency**: Minimal per-process overhead with efficient data structures
- **Resource Cleanup**: Automatic cleanup of ports, file handles, and process references on termination

### 4.4 Health Monitoring Architecture

**HTTP-Based Health Checks**:
- **Standardized Endpoint**: All subprocesses expose `/api/status` for health verification
- **Configurable Timeouts**: Independent timeout configuration for different deployment scenarios
- **State-Based Responses**: Clear mapping between HTTP responses and process health states

**Failure Handling Strategy**:
- **Consecutive Failure Threshold**: Configurable number of consecutive failures before declaring process unhealthy
- **Exponential Backoff**: Restart delays that increase exponentially to prevent thundering herd scenarios
- **Circuit Breaker Pattern**: Temporary suspension of health checks for consistently failing processes

### 4.5 Integration Points

**HTTPS Proxy Integration**:
- **Hostname-Based Routing**: Query interface for finding processes by hostname for request routing
- **Real-Time Port Information**: Dynamic port information for proxy configuration updates
- **Health State Awareness**: Only route to healthy processes

**Admin Service Integration**:
- **Authentication**: Internal secret-based authentication for cross-service communication
- **Event-Driven Updates**: Efficient change detection via event polling to minimize latency
- **Graceful Degradation**: Continued operation with cached configuration during admin service outages

## 5. Implementation Considerations

### 5.1 Concurrency and Thread Safety

- **Mutex-Protected State**: All shared state protected by appropriate synchronization primitives
- **Goroutine Management**: Controlled goroutine lifecycle with proper cleanup and cancellation
- **Atomic Operations**: Use of atomic operations for simple state queries to minimize lock contention

### 5.2 Error Handling and Resilience

- **Graceful Degradation**: System continues operation with reduced functionality during partial failures
- **Comprehensive Logging**: Structured logging with context for debugging and operational visibility
- **Recovery Mechanisms**: Automatic recovery from transient failures with appropriate backoff strategies

### 5.3 Performance Considerations

- **Efficient Reconciliation**: Minimize overhead of reconciliation loop through efficient state comparison
- **Health Check Optimization**: Configurable intervals balanced between responsiveness and resource usage
- **Memory Management**: Bounded memory usage with efficient cleanup of terminated processes

### 5.4 Development and Testing

- **Interface Abstraction**: Clean interfaces to enable unit testing and mocking
- **Static Configuration Support**: Development-friendly configuration overrides
- **Comprehensive Logging**: Detailed operational logs for debugging and development

## 6. Security Considerations

- **Process Isolation**: Each subprocess runs in its own process space with standard OS isolation
- **Internal Authentication**: Secure communication between process manager and admin service
- **Resource Limits**: Bounded resource usage to prevent resource exhaustion attacks
- **Secure Defaults**: Safe default configurations that can be overridden as needed

## 7. Operational Considerations

### 7.1 Monitoring and Observability

- **Process State Metrics**: Real-time visibility into process states and transitions
- **Health Check Metrics**: Success/failure rates and timing information for health checks
- **Resource Usage Tracking**: Port allocation, memory usage, and process count monitoring

### 7.2 Deployment and Configuration

- **Configuration Validation**: Early validation of configuration parameters to prevent runtime errors
- **Graceful Shutdown**: Clean shutdown procedures that properly terminate all managed processes
- **Hot Configuration Reload**: Dynamic configuration updates without process manager restart

## 8. Future Considerations

- **Horizontal Scaling**: Potential extension to multi-node process management
- **Advanced Health Checks**: Support for custom health check protocols beyond HTTP
- **Process Grouping**: Logical grouping of related processes for batch operations
- **Resource Quotas**: Per-process or per-group resource limits and enforcement

## 9. Success Metrics

- **Availability**: >99.9% uptime for managed processes under normal conditions
- **Recovery Time**: <30 seconds average recovery time for failed processes
- **Resource Efficiency**: <5% CPU overhead for process management on typical workloads
- **Configuration Sync**: <5 seconds latency for configuration changes from admin service

This design provides a robust foundation for automated process management while maintaining operational simplicity and developer productivity.
