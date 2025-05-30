# Technical Product Specification: HTTPS Reverse Proxy

## 1. Overview

This document outlines the technical specification for an HTTPS Reverse Proxy server. The proxy will be implemented in Go and will integrate with the existing Process Manager. Its primary function is to route incoming HTTPS requests to the appropriate backend application instances managed by the Process Manager, based on the hostname in the request.

## 2. Goals

-   Provide a secure entry point for all applications managed by the Process Manager.
-   Dynamically route traffic to backend services based on hostname.
-   Support HTTPS only, using a user-provided SSL/TLS certificate.
-   Seamlessly integrate with the Process Manager to discover backend service locations (host/port).

## 3. Non-Goals

-   HTTP support (all traffic must be HTTPS).
-   Automatic certificate provisioning or management (e.g., Let's Encrypt integration). Certificates must be provided.
-   Load balancing beyond simple hostname-based routing to a single active instance for that hostname.
-   Path-based routing within a single hostname.

## 4. System Architecture

The HTTPS Reverse Proxy will be a Go package located at `nexushub/httpsproxy`. It will run as part of the main `nexushub` application, instantiated in `cmd/nexushub/main.go`.

### Components:

1.  **Proxy Server**:
    *   Listens for incoming HTTPS connections on a configurable port (e.g., 443).
    *   Uses a user-provided SSL/TLS certificate and private key for HTTPS.
    *   Parses incoming requests to extract the hostname.
2.  **Routing Logic**:
    *   Interfaces with the Process Manager to get a list of currently active `AppInstance`s.
    *   Each `AppInstance` (from the Process Manager) will need a `HostName` attribute.
    *   Matches the `HostName` from the incoming request against the `HostName` attributes of the managed `AppInstance`s.
    *   Forwards the request to the `host:port` of the matched `AppInstance`.
3.  **Process Manager Integration**:
    *   The proxy will require access to the Process Manager instance.
    *   It will query the Process Manager to get the running port number on which the application instance matching the given HostName is listening, *if* it is in a healthy state.
## 5. Detailed Design

### 5.1. Package Structure

-   `nexushub/httpsproxy/`:
    -   `proxy.go`: Contains the main proxy server logic, including listener setup, request handling, and routing.
    -   `config.go`: (Optional) Structs for proxy configuration (port, certificate paths).
    -   `resolver.go`: (Or similar) Logic for querying the Process Manager to resolve hostnames to backend ports.

### 5.2. Configuration

The proxy will require the following configuration parameters, likely passed during instantiation in `cmd/nexushub/main.go`:
-   Listening port (e.g., `:443`).
-   Path to the SSL/TLS certificate file.
-   Path to the SSL/TLS private key file.
-   A reference to the Process Manager instance.

### 5.3. Request Handling Flow

1.  An HTTPS request arrives at the proxy server.
2.  The proxy server terminates SSL.
3.  The `Host` header is extracted from the incoming HTTP request.
4.  The proxy queries the Process Manager (or a cached view of its state) for the running port number of an `AppInstance` whose `HostName` matches the extracted `Host` header and is in a healthy/running state.
5.  If a match is found:
    *   The proxy constructs the target URL (e.g., `http://localhost:<instance_port>`). Communication from proxy to backend service can be HTTP.
    *   The request (including headers and body) is proxied to the target backend service.
    *   The response from the backend service is proxied back to the original client.
6.  If no match is found (or the matched instance is not healthy):
    *   The proxy returns an appropriate HTTP error (e.g., 404 Not Found or 503 Service Unavailable).

### 5.5. Integration with `cmd/nexushub/main.go`

In `cmd/nexushub/main.go`:
1.  Instantiate the Process Manager.
2.  Instantiate the HTTPS Reverse Proxy, providing it with:
    *   Configuration (listen address, cert/key paths).
    *   A reference to the Process Manager instance (for resolving hostnames to ports).
3.  Start the Process Manager's reconciliation loop.
4.  Start the HTTPS Reverse Proxy server (likely in a separate goroutine).

## 6. Error Handling

-   **Certificate Errors**: Log errors related to loading or using the SSL/TLS certificate. The proxy should fail to start if certificates are invalid or not found.
-   **No Matching HostName**: Return HTTP 404 or 503.
-   **Backend Service Unavailable/Error**: Propagate 5xx errors from the backend, or return a 502 Bad Gateway/503 Service Unavailable if the backend is unreachable.
-   **Process Manager Communication Failure**: Log errors and potentially return 503 if backend information cannot be retrieved.

## 7. Security Considerations

-   **HTTPS Only**: Enforce HTTPS to protect data in transit.
-   **Certificate Management**: Secure handling of the private key is crucial. Key paths should be configurable and file permissions restricted.
-   **Input Validation**: Validate `Host` headers.
-   **Denial of Service**: Consider rate limiting or other DoS protection mechanisms if this proxy is exposed to the public internet directly (though this might be out of scope for the initial version).

## 8. Future Considerations (Out of Scope for V1)

-   Path-based routing.
-   Load balancing across multiple instances of the same application.
-   Automatic certificate management (e.g., Let's Encrypt).
-   WebSockets support.
-   More sophisticated health checks before proxying.
-   Dynamic updates to SSL certificates without restarting the proxy.
