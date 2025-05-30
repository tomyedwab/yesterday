package httpsproxy

import (
	"crypto/tls"
	"log"
	"net" // Added for SplitHostPort
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/tomyedwab/yesterday/database/processes"
)

// ProcessManagerInterface defines the methods the HostnameResolver needs
// from the ProcessManager. This helps in decoupling and testing.
// It should provide a way to get an AppInstance by its hostname.
type ProcessManagerInterface interface {
	GetAppInstanceByHostName(hostname string) (*processes.AppInstance, error)
}

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
	listenAddr string
	certFile   string
	keyFile    string
	pm         ProcessManagerInterface
	server     *http.Server
}

// NewProxy creates and returns a new Proxy instance.
// It takes the listen address, paths to SSL cert and key files,
// and a HostnameResolver instance.
func NewProxy(listenAddr, certFile, keyFile string, pm ProcessManagerInterface) *Proxy {
	return &Proxy{
		listenAddr: listenAddr,
		certFile:   certFile,
		keyFile:    keyFile,
		pm:         pm,
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
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTPS proxy server on %s", p.listenAddr)
	return p.server.ListenAndServeTLS("", "") // Cert and key are in TLSConfig
}

// handleRequest is the HTTP handler function for the proxy.
// It extracts the hostname from the request, resolves it to a backend AppInstance,
// and proxies the request to that instance.
// If resolution fails or the backend is unavailable, it returns an appropriate error.
func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	hostInput := r.Host // r.Host includes port if specified by client
	hostname, _, err := net.SplitHostPort(hostInput)
	if err != nil {
		// If SplitHostPort fails, it might be that hostInput doesn't have a port,
		// which is fine. In that case, hostInput is the hostname itself.
		// Or it could be an invalid format like "[::1]:%invalid%".
		// For simplicity, we'll assume if there's an error, hostInput is the hostname.
		// A more robust solution might inspect the error type.
		hostname = hostInput
	}

	instance, err := p.pm.GetAppInstanceByHostName(hostname)
	if err != nil {
		log.Printf("Error resolving hostname '%s': %v", hostname, err)
		http.Error(w, "Service not found for hostname: "+hostname, http.StatusNotFound)
		return
	}

	if instance == nil {
		log.Printf("No active instance found for hostname '%s'", hostname)
		http.Error(w, "Service unavailable for hostname: "+hostname, http.StatusServiceUnavailable)
		return
	}

	// Construct target URL for the backend service (HTTP)
	targetURL := &url.URL{
		Scheme: "http",                                  // Communication to backend is HTTP
		Host:   instance.HostName + ":" + instance.Port, // Assuming AppInstance has HostName and Port
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Update the request host to match the target for the reverse proxy.
	r.Host = targetURL.Host
	r.URL.Scheme = targetURL.Scheme
	r.URL.Host = targetURL.Host

	log.Printf("Proxying request for %s to %s", hostname, targetURL.String())
	reverseProxy.ServeHTTP(w, r)
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
