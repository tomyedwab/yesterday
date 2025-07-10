# HTTPS Reverse Proxy Specification

Reference: design/httpsproxy.md

## Overview

The HTTPS Reverse Proxy provides a secure entry point for all applications managed by the Process Manager, routing incoming HTTPS requests to backend application instances based on hostname or application ID. It supports TLS termination, authentication, static file serving, and debug/dev proxying.

## Task `proxy-server`: Core proxy server implementation
Reference: design/httpsproxy.md
Implementation status: Fully implemented

Details:
Implement the main `Proxy` struct and HTTP server functionality in `nexushub/httpsproxy/proxy.go`:

- TLS listener on configurable port (default :443)
- SSL/TLS certificate and private key handling
- Request routing based on `Host` header or `X-Application-Id` header
- Integration with ProcessManager for backend discovery
- Request/response logging with trace IDs
- Graceful start/stop functionality

**Key components:**
- `Proxy` struct with configuration (listen address, cert/key paths, internal secret)
- `Start()` method for server initialization
- `handleRequest()` method for request processing and routing
- TLS configuration and certificate loading

**Files:**
- `nexushub/httpsproxy/proxy.go` (892 lines)

## Task `process-manager-integration`: ProcessManager interface and types
Reference: design/httpsproxy.md
Implementation status: Fully implemented

Details:
Define ProcessManager interfaces and implement hostname/AppID-based lookup in `nexushub/httpsproxy/types/processmanager.go`:

- `ProcessManagerInterface` for backend discovery
- `GetAppInstanceByHostName(hostname string)` method
- `GetAppInstanceByID(appID string)` method
- AppInstance struct with required fields: `HostName`, `Port`, `DebugPort`, `StaticPath`

**Files:**
- `nexushub/httpsproxy/types/processmanager.go`

## Task `authentication-system`: Access token validation and login integration
Reference: design/httpsproxy.md
Implementation status: Fully implemented

Details:
Implement authentication middleware and access token handling:

- Bearer token validation for `/api/*` and `/internal/*` endpoints
- Internal secret authentication
- Access token validation via login service integration
- Cookie-based authentication support

**Key components:**
- Token validation functions
- Access token request handling
- Integration with external login service
- Security middleware for protected routes

**Files:**
- `nexushub/httpsproxy/access/request.go`
- `nexushub/httpsproxy/access/tokens.go`

## Task `routing-logic`: Multi-path routing implementation
Reference: design/httpsproxy.md
Implementation status: Fully implemented

Details:
Implement comprehensive request routing system with multiple backend discovery methods:

1. **Application ID routing**: Use `X-Application-Id` header with `GetAppInstanceByID()`
2. **Hostname routing**: Use `Host` header with `GetAppInstanceByHostName()`
3. **Special endpoint handling:**
   - `/public/login` and `/public/logout`: Always routes to the login service regardless of Host header (centralized authentication)
   - `/api/set_token`: Cookie setting and redirect functionality
   - `/api/access_token`: Access token request handling
   - `/public/*`: Unauthenticated proxying to backend
   - `/api/*`: Authenticated API proxying (Bearer token required)
   - `/internal/*`: Internal API access (internal secret required)
4. **Debug/dev proxying**: Route to debug port if `DebugPort > 0`
5. **Static file serving**: Serve from `StaticPath` with CORS headers
6. **404 handling**: Return appropriate error responses

**Security features:**
- Path traversal prevention for static files
- Authentication enforcement for protected routes
- CORS header management for static files

## Task `static-file-serving`: Static file handling with security
Reference: design/httpsproxy.md
Implementation status: Fully implemented

Details:
Implement secure static file serving functionality:

- Serve files from AppInstance `StaticPath` directory
- Root `/` request mapping to `/index.html`
- Path traversal attack prevention
- CORS headers for GET requests
- Proper MIME type detection and headers
- Fallback to 404 when files not found

**Security considerations:**
- Input validation and sanitization
- File path validation to prevent directory traversal
- Appropriate HTTP headers and status codes

## Task `error-handling`: Comprehensive error handling and logging
Reference: design/httpsproxy.md
Implementation status: Fully implemented

Details:
Implement robust error handling throughout the proxy:

- Certificate loading and validation errors (startup failure)
- Backend service discovery failures (404/503 responses)
- Backend communication errors (502/503 propagation)
- Authentication failures (401/403 responses)
- Request processing errors with trace ID logging
- Graceful degradation for service unavailability

**Logging requirements:**
- All requests logged with trace IDs
- Error conditions with appropriate detail levels
- Security events for audit purposes

## Task `main-integration`: Integration with cmd/nexushub/main.go
Reference: design/httpsproxy.md
Implementation status: Fully implemented

Details:
Integrate the HTTPS proxy server into the main NexusHub application:

1. Instantiate ProcessManager
2. Create Proxy instance with configuration:
   - Listen address (`:443`)
   - Certificate and key file paths
   - Internal secret for authentication
   - ProcessManager interface reference
3. Start ProcessManager reconciliation loop
4. Start HTTPS proxy server in separate goroutine
5. Implement graceful shutdown handling

**Configuration parameters:**
- `listen_address`: HTTPS listen port (default `:443`)
- `cert_file`: Path to SSL/TLS certificate
- `key_file`: Path to SSL/TLS private key
- `internal_secret`: Authentication secret for internal endpoints

**Files:**
- `cmd/nexushub/main.go` (integration code)

## Task `app-instance-hostname`: AppInstance hostname field extension
Reference: design/httpsproxy.md
Implementation status: Fully implemented

Details:
Extend the `AppInstance` struct in the Process Manager to support hostname-based routing:

- Add `HostName` field to `database/processes/AppInstance` struct
- Add `Port` field for backend service port information
- Update ProcessManager with `GetAppInstanceByHostName()` method
- Ensure hostname uniqueness validation during app registration

**Files:**
- `database/processes/instance.go` (AppInstance struct)
- `database/processes/manager.go` (hostname lookup method)

## Task `security-hardening`: Security measures and best practices
Reference: design/httpsproxy.md
Implementation status: Fully implemented

Details:
Implement comprehensive security measures:

- **HTTPS enforcement**: No HTTP support, TLS-only connections
- **Input validation**: Host header and path validation
- **Authentication**: Multi-tier token validation system
- **Path security**: Directory traversal prevention
- **Access control**: Route-based authentication requirements
- **Error disclosure**: Minimal error information in responses
- **Audit logging**: Security events and access patterns

**Security features:**
- Bearer token validation for API endpoints
- Internal secret for administrative access
- CORS policy management
- Request sanitization and validation
- Secure cookie handling for authentication

## Implementation Notes

The HTTPS Reverse Proxy is fully implemented with all core functionality:

- **Total implementation**: ~1,200+ lines across multiple files
- **Main proxy logic**: `nexushub/httpsproxy/proxy.go`
- **Authentication system**: `nexushub/httpsproxy/access/` package
- **Type definitions**: `nexushub/httpsproxy/types/` package
- **Integration**: Complete integration with ProcessManager and main application

**Key features implemented:**
- Multi-path routing with hostname and AppID support
- Comprehensive authentication and authorization
- Static file serving with security controls
- Debug/dev server proxying support
- Full TLS termination and certificate management
- Audit logging and error handling
- Graceful startup and shutdown procedures

**Testing considerations:**
- Certificate validation and TLS functionality
- Backend service discovery and routing
- Authentication and authorization flows
- Static file serving security
- Error handling and recovery scenarios
- Performance under load conditions
