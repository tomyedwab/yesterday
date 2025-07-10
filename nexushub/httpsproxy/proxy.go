package httpsproxy

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"            // For file system operations
	"path/filepath" // For path manipulation
	"strconv"
	"strings" // For string manipulation
	"time"

	"github.com/google/uuid"
	"github.com/tomyedwab/yesterday/nexushub/httpsproxy/access"
	httpsproxy_types "github.com/tomyedwab/yesterday/nexushub/httpsproxy/types"
	"github.com/tomyedwab/yesterday/nexushub/processes"
)

// Proxy represents the HTTPS reverse proxy server.
// It listens for incoming HTTPS requests, terminates SSL, and proxies them
// to the appropriate backend service based on the hostname.
// Proxy requires a HostnameResolver to find the backend services.
// It also needs paths to the SSL certificate and private key files.
// The ListenAddr specifies the address and port the proxy should listen on (e.g., ":443").
// Communication with backend services is over HTTP.
// Proxy uses httputil.NewSingleHostReverseProxy for the actual proxying.
// If a hostname cannot be resolved or the backend service is unavailable,
// appropriate HTTP error codes are returned (404 or 503).
// Errors during startup (e.g., loading certificates) are logged and cause a panic.
type Proxy struct {
	listenAddr     string
	certFile       string
	keyFile        string
	pm             httpsproxy_types.ProcessManagerInterface
	server         *http.Server
	transport      *http.Transport
	internalSecret string
}

// NewProxy creates and returns a new Proxy instance.
// It takes the listen address, paths to SSL cert and key files,
// and a HostnameResolver instance.
func NewProxy(listenAddr, certFile, keyFile, internalSecret string, pm httpsproxy_types.ProcessManagerInterface) *Proxy {
	dialer := net.Dialer{
		Timeout:   600 * time.Second,
		KeepAlive: 600 * time.Second,
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 180 * time.Second,
	}
	return &Proxy{
		listenAddr:     listenAddr,
		certFile:       certFile,
		keyFile:        keyFile,
		pm:             pm,
		transport:      transport,
		internalSecret: internalSecret,
	}
}

// Start initializes and starts the HTTPS reverse proxy server.
// It sets up an HTTP server with a handler that proxies requests.
// SSL/TLS is configured using the provided certificate and key files.
// The server listens on the address specified in p.listenAddr.
// This method blocks until the server stops or an error occurs.
// If certificate files cannot be loaded, it will log the error and panic.
func (p *Proxy) Start() error {
	// Load TLS certificates
	cert, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
	if err != nil {
		log.Printf("Error loading TLS certificate: %v", err)
		return err // Return error instead of panic to allow main to handle
	}

	p.server = &http.Server{
		Addr:    p.listenAddr,
		Handler: http.HandlerFunc(p.handleRequest),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTPS proxy server on %s", p.listenAddr)
	return p.server.ListenAndServeTLS("", "") // Cert and key are in TLSConfig
}

// handleRequest is the HTTP handler function for the proxy.
// It checks for "X-Application-Id" header. If set, uses it to find the AppInstance.
// Otherwise, it extracts the hostname from the request and resolves it to a backend AppInstance.
// If the instance has a StaticPath and the requested resource exists as a file, it serves the static file.
// Otherwise, it proxies the request to that instance.
// If resolution fails or the backend is unavailable, it returns an appropriate error.
func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	var instance *processes.AppInstance
	var port int
	var err error
	var resolutionIdentifier string        // For logging: "AppID 'xyz'" or "hostname 'abc.com'"
	var originalHostForLog string = r.Host // Capture original host for logging before it's modified

	traceID := uuid.New().String()

	// Special routing rule: /public/login always routes to the login service regardless of Host header
	if r.URL.Path == "/public/login" || r.URL.Path == "/public/logout" {
		resolutionIdentifier = "login-service"
		instance, port, err = p.pm.GetAppInstanceByID("3bf3e3c0-6e51-482a-b180-00f6aa568ee9")
		if err != nil {
			http.Error(w, "Login service not found", http.StatusNotFound)
			log.Printf("<%s> %s%s 404 [Login service not found]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}
		if instance == nil {
			http.Error(w, "Login service unavailable", http.StatusServiceUnavailable)
			log.Printf("<%s> %s%s 503 [Login service unavailable]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}

		// Route to the login service - treat as /public/* (unauthenticated)
		targetURL := &url.URL{
			Scheme: "http", // Backend services are HTTP
			Host:   "localhost:" + strconv.Itoa(port),
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
		reverseProxy.Transport = p.transport
		r.Host = targetURL.Host
		r.Header.Add("X-Trace-ID", traceID)

		log.Printf("<%s> %s%s => %s [login-service]", traceID, resolutionIdentifier, r.URL.Path, targetURL.String())
		reverseProxy.ServeHTTP(w, r)
		return
	}

	appID := r.Header.Get("X-Application-Id")

	if appID != "" {
		resolutionIdentifier = "app:" + appID
		instance, port, err = p.pm.GetAppInstanceByID(appID)
		if err != nil {
			http.Error(w, "Service not found for "+resolutionIdentifier, http.StatusNotFound)
			log.Printf("<%s> %s%s 404 [Service not found]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}
		if instance == nil {
			http.Error(w, "Service unavailable for "+resolutionIdentifier, http.StatusServiceUnavailable)
			log.Printf("<%s> %s%s 404 [No active instances]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}
	} else {
		// Fallback to hostname-based routing
		hostname := originalHostForLog // r.Host includes port if specified by client
		resolutionIdentifier = "host:" + hostname
		instance, port, err = p.pm.GetAppInstanceByHostName(hostname)
		if err != nil {
			http.Error(w, "Service not found for "+resolutionIdentifier, http.StatusNotFound)
			log.Printf("<%s> %s%s 404 [Service not found]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}
		if instance == nil {
			http.Error(w, "Service unavailable for "+resolutionIdentifier, http.StatusServiceUnavailable)
			log.Printf("<%s> %s%s 404 [No active instances]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}
	}

	if r.URL.Path == "/api/set_token" {
		// Get token and continue URL from URL parameters
		token := r.URL.Query().Get("token")
		continueURL := r.URL.Query().Get("continue")
		if token == "" || continueURL == "" {
			http.Error(w, "Missing token or continue URL", http.StatusBadRequest)
			log.Printf("<%s> %s/api/set_token 400", traceID, resolutionIdentifier)
			return
		}
		// Set the cookie with the token and redirect to the continue URL
		targetDomain := r.Host
		if strings.Contains(targetDomain, ":") {
			targetDomain = strings.Split(targetDomain, ":")[0]
		}
		w.Header().Set("Set-Cookie", "YRT="+token+"; Path=/; Domain="+targetDomain+"; HttpOnly; Secure; SameSite=None")
		w.Header().Set("Location", continueURL)
		w.WriteHeader(http.StatusFound)
		log.Printf("<%s> %s/api/set_token 302", traceID, resolutionIdentifier)
		return
	}
	if r.URL.Path == "/api/access_token" {
		code := access.HandleAccessTokenRequest(p.pm, instance, w, r, traceID)
		log.Printf("<%s> %s/api/access_token %d", traceID, resolutionIdentifier, code)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/public/") {
		// Token is valid, proxy the request
		targetURL := &url.URL{
			Scheme: "http", // Backend services are HTTP
			Host:   "localhost:" + strconv.Itoa(port),
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
		reverseProxy.Transport = p.transport
		r.Host = targetURL.Host
		r.Header.Add("X-Trace-ID", traceID)

		log.Printf("<%s> %s%s => %s", traceID, resolutionIdentifier, r.URL.Path, targetURL.String())
		reverseProxy.ServeHTTP(w, r)
		return
	}

	// Handle /api/ paths with authorization
	if strings.HasPrefix(r.URL.Path, "/api/") {
		// Validate authorization for API endpoints
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("<%s> %s%s => 401 [Missing token]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		valid := token == p.internalSecret
		if !valid {
			valid = access.ValidateAccessToken(token, instance.InstanceID)
		}
		if !valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("<%s> %s%s => 401 [Invalid token]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}

		// Token is valid, proxy the request
		targetURL := &url.URL{
			Scheme: "http", // Backend services are HTTP
			Host:   "localhost:" + strconv.Itoa(port),
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
		reverseProxy.Transport = p.transport
		r.Host = targetURL.Host
		r.Header.Add("X-Trace-ID", traceID)

		log.Printf("<%s> %s%s => %s", traceID, resolutionIdentifier, r.URL.Path, targetURL.String())
		reverseProxy.ServeHTTP(w, r)
		return
	}

	// Handle /internal/ paths
	if strings.HasPrefix(r.URL.Path, "/internal/") {
		if r.Header.Get("Authorization") != "Bearer "+p.internalSecret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("<%s> %s%s => 401 [Invalid token]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}
		targetURL := &url.URL{
			Scheme: "http", // Backend services are HTTP
			Host:   "localhost:" + strconv.Itoa(port),
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
		reverseProxy.Transport = p.transport
		r.Host = targetURL.Host
		r.Header.Add("X-Trace-ID", traceID)

		log.Printf("<%s> %s%s => %s", traceID, resolutionIdentifier, r.URL.Path, targetURL.String())
		reverseProxy.ServeHTTP(w, r)
		return
	}

	// Handle other paths through DebugPort and StaticPath checks
	if instance.DebugPort > 0 {
		// If debug port is enabled, proxy to the dev server
		targetURL := &url.URL{
			Scheme: "http", // Backend services are HTTP
			Host:   "localhost:" + strconv.Itoa(instance.DebugPort),
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Update the request host to match the target for the reverse proxy.
		r.Host = targetURL.Host
		r.Header.Add("X-Trace-ID", traceID)

		log.Printf("<%s> %s%s => %s", traceID, resolutionIdentifier, r.URL.Path, targetURL.String())
		reverseProxy.ServeHTTP(w, r)
		return

	} else {
		// Check for static file serving
		requestedPath := r.URL.Path
		if requestedPath == "/" {
			requestedPath = "/index.html" // Map root to index.html
		}

		filePath := filepath.Join(instance.PkgPath, "app", "static", filepath.Clean(requestedPath))
		cleanStaticPath := filepath.Clean(instance.PkgPath)
		cleanFilePath := filepath.Clean(filePath)

		if !strings.HasPrefix(cleanFilePath, cleanStaticPath) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			log.Printf("<%s> %s%s 403 [Invalid path]", traceID, resolutionIdentifier, r.URL.Path)
			return
		}

		fileInfo, statErr := os.Stat(cleanFilePath)
		if statErr == nil {
			if !fileInfo.IsDir() {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET")
				w.Header().Set("Access-Control-Max-Age", "86400")
				log.Printf("<%s> %s%s => %s", traceID, resolutionIdentifier, r.URL.Path, cleanFilePath)
				http.ServeFile(w, r, cleanFilePath)
				return
			}
		}
		// If file doesn't exist or is a directory, fall through to 404
	}

	// No matching route found
	http.Error(w, "Not Found", http.StatusNotFound)
	log.Printf("<%s> %s%s 404 [No route found]", traceID, resolutionIdentifier, r.URL.Path)
}

// Stop gracefully shuts down the proxy server.
func (p *Proxy) Stop() error {
	if p.server == nil {
		log.Printf("Proxy server was not running or not initialized, nothing to stop.")
		return nil
	}
	log.Printf("Stopping HTTPS proxy server...")
	return p.server.Shutdown(nil) // Use context.WithTimeout for graceful shutdown if needed
}
