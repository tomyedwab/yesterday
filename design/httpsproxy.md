# Design Document: HTTPS Reverse Proxy

## 1. Motivation

As the Yesterday platform expands to support multiple web applications and native executables running in libkrun microVMs, there is a need for a unified, secure, and flexible entry point for all external traffic. The HTTPS Reverse Proxy will provide this, enabling hostname-based routing, TLS termination, and seamless integration with the dynamic process management system.

## 2. Requirements

The HTTPS Reverse Proxy is part of the `nexushub` application, which is a
standalone application which also manages the application processes. See
`design/nexushub.md` and `design/processes.md` for more details.

### Functional Requirements
- Accept incoming HTTPS connections on a configurable port (default: 443).
- Terminate SSL/TLS using a user-provided certificate and private key.
- Route requests to backend application instances based on the HTTP Host header or an explicit application ID.
- Integrate with the Process Manager to discover running AppInstances and their ports.
- Support secure and authenticated access to API endpoints, with integration to the login system.
- Serve static files for applications that provide them.
- Support proxying to debug/dev servers if configured.
- Log all requests with trace IDs for debugging and auditing.

### Non-Functional Requirements
- Must not support HTTP (unencrypted) traffic.
- Should be robust to backend failures and log all errors.
- Should prevent path traversal and unauthorized static file access.
- Should be extensible for future features (e.g., load balancing, WebSockets, rate limiting).

## 3. Architecture

### High-Level Diagram

```
[Client] --HTTPS--> [Reverse Proxy] --HTTP--> [AppInstance]
```

### Components
- **Proxy Server**: Listens for HTTPS connections, terminates SSL, parses requests, and routes them.
- **Routing Logic**: Determines the correct backend instance based on Host header or X-Application-Id, and the current state of the Process Manager.
- **Process Manager Integration**: Queries the Process Manager for AppInstance details (host, port, health).
- **Authentication & Access Control**: Validates Bearer tokens for API/internal routes, integrates with the login service for access tokens.
- **Static File Server**: Serves static files if configured for an AppInstance, with CORS support and path validation.
- **Debug/Dev Proxy**: Proxies requests to a debug/dev server if the AppInstance is running in development mode.

## 4. Routing and Endpoint Design

- `/public/login` and `/public/logout`: Special routing rule - always routes to the login service regardless of Host header. Used for centralized authentication.
- `/api/set_token`: Sets a cookie and redirects to a provided URL. Used after login.
- `/public/access_token`: Handles access token requests, integrating with the login service.
- `/public/*`: Proxies to the backend without authentication.
- `/api/*`: Requires a Bearer token (internal secret or validated access token for the app instance).
- `/internal/*`: Requires the internal secret for authorization.
- Other paths: If DebugPort is set, proxy to dev server. If StaticPath is set and file exists, serve static file. Otherwise, 404.

## 5. Security Considerations

- Enforce HTTPS for all traffic.
- Validate all input headers and paths.
- Prevent path traversal in static file serving.
- Restrict access to API/internal endpoints via token validation.
- Log all requests and errors with trace IDs.
- Do not store or manage certificates beyond loading from the filesystem.

## 6. Error Handling

- Startup fails if certificates are missing or invalid.
- 404/503 for missing or unhealthy AppInstances.
- 401/403 for authentication failures or unauthorized access.
- 502/503 for backend errors.
- All errors are logged.

## 7. Extensibility & Future Work

- Path-based routing.
- Load balancing across multiple instances.
- WebSockets support.
- Automatic certificate management.
- Rate limiting and DoS protection.
- More granular static file permissions.
- Centralized logging/audit system.

## 8. References
- `spec/httpsproxy.md` (specification)
