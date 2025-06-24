# KrunClient Technical Specification

Reference: design/krunclient.md
Implementation status: Fully implemented (2025-06-24)

## Introduction

KrunClient is a C-based virtualization wrapper that provides secure, isolated runtime environments for NexusHub application instances using libkrun microVMs. It serves as the execution layer between the Process Manager and guest applications, enabling strong security boundaries while maintaining performance efficiency.

## Architecture Overview

The system consists of a host binary (`krunclient`) that creates and manages libkrun-based microVMs for application instances. Each VM runs with a minimal root filesystem and provides TCP port mapping between guest and host environments.

**Key Components:**
- Host executable written in C using libkrun API
- Minimal root filesystem with essential system libraries  
- Build system integration for binary compilation and rootfs distribution
- Process Manager integration for lifecycle management

## Implementation Details

### Source Structure

```
nexushub/krunclient/
├── main.c                    # Main krunclient executable (47 lines)
└── rootfs/                   # Minimal root filesystem
    ├── etc/hosts            # Network configuration with internal.yesterday.localhost
    ├── lib/x86_64-linux-gnu/ # Essential system libraries
    └── lib64/               # 64-bit library loader
```

### Core Implementation

The main executable (`main.c`) implements:

1. **Command Line Processing**: Validates argc/argv for root path and port parameters
2. **Environment Propagation**: Extracts HOST and INTERNAL_SECRET from parent environment
3. **VM Configuration**: Creates libkrun context with 1 CPU, 512MB RAM
4. **Port Mapping**: Maps guest port 80 to specified host port
5. **Execution**: Launches `/bin/app` within the VM environment

**Key Functions:**
```c
int main(int argc, char *argv[])
krun_create_ctx()                    // VM context creation
krun_set_vm_config(ctx_id, 1, 512)  // Resource allocation
krun_set_root(ctx_id, argv[1])      // Root filesystem setup
krun_set_port_map(ctx_id, port_map) // Network configuration
krun_set_exec(ctx_id, "/bin/app", 0, envp) // Guest execution
krun_start_enter(ctx_id)            // VM startup
```

### Build Integration

**Makefile Integration:**
- Binary compilation: `gcc -o dist/github.com/tomyedwab/yesterday/nexushub/bin/krunclient nexushub/krunclient/main.c -l krun`
- Root filesystem distribution to application directories during build process
- Integration with process manager binary path resolution

### Process Manager Integration

**Command Line Interface:**
```bash
krunclient <root_path> <local_port>
```

**Environment Variables:**
- `HOST`: Hostname for application configuration
- `INTERNAL_SECRET`: Authentication token for internal services

**Execution Context:**
- Binary path: `dist/github.com/tomyedwab/yesterday/nexushub/bin/krunclient`
- Working directory: Process manager subprocess work directory
- Process lifecycle: Managed by ProcessManager with health monitoring

## Configuration

### VM Configuration
- **CPU**: 1 virtual CPU (hardcoded)
- **Memory**: 512MB RAM (hardcoded)
- **Port Mapping**: Guest port 80 → Host port (dynamic from process manager)
- **Root Filesystem**: Application-specific directory path

### Network Configuration
- Internal hostname resolution: `127.0.0.1 internal.yesterday.localhost`
- TCP port forwarding for HTTP traffic
- Isolated network namespace per VM

### Environment Configuration
- HOST variable propagation for external hostname configuration
- INTERNAL_SECRET passing for secure internal communication
- Standard environment inheritance from parent process

## Tasks

## Task `krunclient-binary`: Main executable implementation
Reference: design/krunclient.md
Implementation status: Completed

**Details:**
C program using libkrun API to create and manage microVMs. Handles command-line arguments for root path and port, propagates environment variables, configures VM with 1 CPU and 512MB RAM, sets up port mapping from guest:80 to host:PORT, and executes /bin/app within the VM.

**Implementation:**
- File: `nexushub/krunclient/main.c` (47 lines)
- Command line validation with usage message
- Environment variable extraction and propagation
- libkrun context creation and configuration
- Port mapping setup with dynamic port assignment
- Guest application execution with environment passing

## Task `rootfs-structure`: Minimal root filesystem
Reference: design/krunclient.md  
Implementation status: Completed

**Details:**
Minimal Linux root filesystem containing only essential system libraries and configuration files needed for guest application execution. Includes network configuration for internal host communication.

**Implementation:**
- Directory: `nexushub/krunclient/rootfs/`
- Network config: `etc/hosts` with internal.yesterday.localhost mapping
- System libraries: `lib/x86_64-linux-gnu/libc.so.6` and `lib64/ld-linux-x86-64.so.2`
- Optimized for size and security with minimal attack surface

## Task `build-integration`: Compilation and distribution
Reference: design/krunclient.md
Implementation status: Completed

**Details:**
Integration with project build system for binary compilation and root filesystem distribution. Compiles krunclient with libkrun linking and copies rootfs to application directories.

**Implementation:**
- Makefile target: `gcc -o dist/github.com/tomyedwab/yesterday/nexushub/bin/krunclient nexushub/krunclient/main.c -l krun`
- Root filesystem distribution: `cp -R nexushub/krunclient/rootfs/* dist/github.com/tomyedwab/yesterday/apps/*/`
- Binary path resolution in process manager

## Task `process-manager-integration`: Lifecycle management integration  
Reference: design/krunclient.md
Implementation status: Completed

**Details:**
Integration with ProcessManager for application instance lifecycle management. ProcessManager invokes krunclient with appropriate arguments and manages VM process lifecycle.

**Implementation:**
- Command execution in `nexushub/processes/manager.go` startProcess function
- Binary path: `dist/github.com/tomyedwab/yesterday/nexushub/bin/krunclient`
- Arguments: instance.BinPath (root path), port (allocated by PortManager)
- Environment: HOST from instance.HostName, INTERNAL_SECRET from manager
- Process monitoring and restart handling via ProcessManager

## Task `environment-propagation`: Host-to-guest variable passing
Reference: design/krunclient.md
Implementation status: Completed

**Details:**  
Secure propagation of configuration and authentication data from host process manager to guest application environment. Handles HOST and INTERNAL_SECRET variables.

**Implementation:**
- Environment scanning in main.c for HOST= and INTERNAL_SECRET= prefixes
- Dynamic string allocation for environment variable values
- Environment array construction for krun_set_exec call
- Process manager environment setup with instance-specific values

## Task `port-mapping`: Network connectivity configuration
Reference: design/krunclient.md
Implementation status: Completed

**Details:**
TCP port mapping between guest VM and host system to enable HTTP traffic forwarding. Maps guest port 80 to dynamically allocated host port.

**Implementation:**
- Dynamic port mapping string construction: `snprintf(port_mapping, sizeof(port_mapping), "%s:80", argv[2])`
- Port map array creation for libkrun API
- krun_set_port_map configuration with generated mapping
- Integration with ProcessManager PortManager for host port allocation

## Task `error-handling`: Failure scenarios and recovery
Reference: design/krunclient.md
Implementation status: Completed

**Details:**
Error handling for command-line validation, environment setup, and VM lifecycle management. Implements fail-fast strategy with clear error messaging.

**Implementation:**
- Command line argument validation with usage message on failure
- Environment variable validation and error logging
- libkrun API error handling with appropriate exit codes
- Process manager integration for restart and recovery logic

## Security Considerations

- **VM Isolation**: Strong security boundaries through hardware virtualization
- **Minimal Attack Surface**: Reduced root filesystem with only essential libraries  
- **Environment Security**: Secure propagation of secrets via environment variables
- **Network Isolation**: Limited connectivity through explicit port mapping only
- **Process Isolation**: Each application instance runs in separate microVM

## Performance Characteristics

- **Startup Time**: Fast VM initialization using libkrun's lightweight runtime
- **Memory Overhead**: 512MB base allocation plus application requirements
- **CPU Performance**: Near-native execution with hardware virtualization
- **Resource Efficiency**: Minimal host overhead beyond VM resource allocation

## Integration Points

- **Process Manager**: Lifecycle management and health monitoring
- **Application Build**: Root filesystem and binary distribution
- **Port Manager**: Dynamic port allocation and conflict resolution  
- **HTTPS Proxy**: Traffic routing to allocated VM ports
- **Admin Service**: Configuration and monitoring integration
