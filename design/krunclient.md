# KrunClient Design Document

**Technical Specification**: spec/krunclient.md

## Problem Statement

NexusHub requires a secure, isolated runtime environment for executing application instances. Traditional process isolation is insufficient for multi-tenant applications that need strong security boundaries while maintaining performance and resource efficiency.

## Goals

- **Container-less Virtualization**: Provide lightweight VM-based isolation using libkrun for application instances
- **Port Mapping**: Enable TCP port forwarding between host and guest VM environments
- **Environment Propagation**: Pass critical configuration (hostname, secrets) from host to guest
- **Minimal Root Filesystem**: Maintain a lean base environment with only essential system libraries
- **Process Manager Integration**: Seamlessly integrate with the existing process manager for lifecycle management

## Non-Goals

- Multi-architecture support (initially x86_64 only)
- Complex networking beyond basic port mapping
- Persistent storage management (handled at application level)
- Container runtime compatibility (Docker, OCI)

## Architecture Overview

```
┌─────────────────┐
│ Process Manager │
└─────────┬───────┘
          │ exec
          ▼
┌─────────────────┐    ┌──────────────────┐
│   KrunClient    │───▶│   libkrun VM     │
│   (Host Binary) │    │                  │
└─────────────────┘    │ ┌──────────────┐ │
          │             │ │ /bin/app     │ │
          │             │ │ (Guest App)  │ │
          │             │ └──────────────┘ │
          │             └──────────────────┘
          │
    Port Mapping
    (guest:80 → host:PORT)
```

## Technical Architecture

### Core Components

1. **KrunClient Executable** (`nexushub/krunclient/main.c`)
   - C program that uses libkrun to create and manage VMs
   - Handles command-line argument parsing and VM configuration
   - Manages port mapping and environment variable propagation

2. **Root Filesystem** (`nexushub/krunclient/rootfs/`)
   - Minimal Linux environment for guest applications
   - Contains essential libraries and network configuration
   - Optimized for size and security

3. **Build Integration** (via Makefile)
   - Compiles krunclient binary with libkrun dependency
   - Distributes rootfs to application directories during build

### Process Flow

1. **Initialization**: Process Manager invokes krunclient with root path and port
2. **VM Setup**: krunclient creates libkrun context and configures VM parameters
3. **Environment**: HOST and INTERNAL_SECRET variables passed from parent process
4. **Execution**: Guest application `/bin/app` launched within VM
5. **Port Mapping**: VM port 80 mapped to specified host port for HTTP traffic

### Security Model

- **VM Isolation**: Each application instance runs in separate microVM
- **Network Isolation**: Only explicitly mapped ports accessible from host
- **Environment Secrets**: Secure propagation of authentication tokens
- **Filesystem Isolation**: Read-only root filesystem with application-specific overlays

## Integration Points

### Process Manager Integration

- **Command Line**: `krunclient <root_path> <local_port>`
- **Environment Variables**: HOST, INTERNAL_SECRET
- **Binary Path**: `dist/github.com/tomyedwab/yesterday/nexushub/bin/krunclient`
- **Working Directory**: Subprocess work directory from process manager

### Application Integration

- **Guest Binary**: Applications compiled as `/bin/app` within root filesystem
- **Port Convention**: Applications listen on port 80 within guest VM
- **Host Communication**: Access to `internal.yesterday.localhost` for host services

### Build System Integration

- **Compilation**: GCC with libkrun linking (`-l krun`)
- **Distribution**: Root filesystem copied to each application directory
- **Packaging**: Single executable with embedded dependencies

## Configuration

### Runtime Configuration

- **VM Resources**: 1 CPU, 512MB RAM (hardcoded in krunclient)
- **Port Mapping**: Dynamic assignment from process manager
- **Root Path**: Application-specific directory containing guest filesystem
- **Network**: Basic TCP port forwarding, internal host resolution

### Environment Variables

- `HOST`: Hostname for external access configuration
- `INTERNAL_SECRET`: Authentication token for internal service communication

## Error Handling and Resilience

### Error Scenarios

1. **Missing Root Filesystem**: Fail fast with clear error message
2. **Port Allocation Failure**: Return non-zero exit code for process manager retry
3. **VM Creation Failure**: Log libkrun errors and exit gracefully
4. **Guest Application Crash**: VM termination detected by process manager

### Recovery Strategies

- Process manager handles restart logic with exponential backoff
- No internal retry mechanism in krunclient (fail fast principle)
- Clean VM shutdown on SIGTERM/SIGINT signals

## Performance Considerations

### Resource Usage

- **Startup Time**: Fast VM initialization using libkrun's lightweight runtime
- **Memory Overhead**: Minimal footprint beyond guest application requirements
- **CPU Overhead**: Near-native performance with hardware virtualization

### Scalability

- **Concurrent VMs**: Limited by host resources and libkrun constraints
- **Port Range**: Process manager manages port allocation conflicts
- **File Descriptors**: Efficient resource management per VM instance

## Security Considerations

### Attack Surface

- **Host Interface**: Minimal exposure through command-line arguments only
- **Guest Isolation**: Strong VM boundaries prevent lateral movement
- **Secret Management**: Environment-based secret passing with process isolation

### Hardening

- **Minimal Root FS**: Only essential libraries included
- **Network Restrictions**: Limited connectivity to mapped ports only
- **Process Privileges**: Run with minimal required permissions

## Future Considerations

### Extensibility

- **Multi-architecture**: ARM64 support for diverse deployment environments
- **Storage Mounting**: Persistent volume support for stateful applications
- **Network Policies**: Advanced networking rules and service mesh integration
- **Resource Limits**: Dynamic CPU/memory allocation based on application requirements

### Integration Enhancements

- **Health Monitoring**: Built-in health check endpoints within VMs
- **Metrics Collection**: Resource usage monitoring and reporting
- **Logging Integration**: Structured log forwarding from guest to host
- **Debug Support**: Development mode with enhanced debugging capabilities

## Success Metrics

- **Security**: Zero privilege escalation incidents from guest to host
- **Performance**: <100ms VM startup time for typical applications
- **Reliability**: >99.9% successful VM launches under normal conditions
- **Resource Efficiency**: <50MB overhead per VM instance
