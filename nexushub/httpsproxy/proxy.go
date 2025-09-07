package httpsproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http" // For file system operations
	"net/http/httputil"
	"net/url"
	"strconv"

	// For path manipulation
	"strings" // For string manipulation
	"time"

	"github.com/google/uuid"
	"github.com/tomyedwab/yesterday/nexushub/events"
	"github.com/tomyedwab/yesterday/nexushub/httpsproxy/access"
	"github.com/tomyedwab/yesterday/nexushub/httpsproxy/middleware"
	httpsproxy_types "github.com/tomyedwab/yesterday/nexushub/httpsproxy/types"
	"github.com/tomyedwab/yesterday/nexushub/internal/handlers"
	app_handlers "github.com/tomyedwab/yesterday/nexushub/internal/handlers/applications"
	event_handlers "github.com/tomyedwab/yesterday/nexushub/internal/handlers/events"
	"github.com/tomyedwab/yesterday/nexushub/internal/handlers/login"
	"github.com/tomyedwab/yesterday/nexushub/processes"
)

// Proxy represents the HTTPS reverse proxy server.
// It listens for incoming HTTPS requests, terminates SSL, and proxies them
// to the appropriate backend service based on the URL path or host name.
// Communication with backend services is over HTTP.
type Proxy struct {
	listenAddr     string
	host           string
	certFile       string
	keyFile        string
	pm             httpsproxy_types.ProcessManagerInterface
	server         *http.Server
	transport      *http.Transport
	internalSecret string
	debugHandler   *handlers.DebugHandler
	eventManager   *events.EventManager
}

// NewProxy creates and returns a new Proxy instance.
// It takes the listen address, paths to SSL cert and key files,
// and a HostnameResolver instance.
func NewProxy(
	listenAddr,
	host,
	certFile,
	keyFile,
	internalSecret string,
	pm httpsproxy_types.ProcessManagerInterface,
	instanceProvider httpsproxy_types.AppInstanceProvider,
	eventManager *events.EventManager,
) *Proxy {
	dialer := net.Dialer{
		Timeout:   600 * time.Second,
		KeepAlive: 600 * time.Second,
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 180 * time.Second,
	}

	// Create logger for debug handler
	logger := slog.Default()
	debugHandler := handlers.NewDebugHandler(pm, instanceProvider, logger, internalSecret)

	return &Proxy{
		listenAddr:     listenAddr,
		host:           host,
		certFile:       certFile,
		keyFile:        keyFile,
		pm:             pm,
		transport:      transport,
		internalSecret: internalSecret,
		debugHandler:   debugHandler,
		eventManager:   eventManager,
	}
}

func (p *Proxy) Start(contextFn func(net.Listener) context.Context, instanceProvider httpsproxy_types.AppInstanceProvider) error {
	// Load TLS certificates
	cert, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
	if err != nil {
		log.Printf("Error loading TLS certificate: %v", err)
		return err // Return error instead of panic to allow main to handle
	}

	p.server = &http.Server{
		BaseContext: contextFn,
		Addr:        p.listenAddr,
		Handler:     http.HandlerFunc(p.handleRequest),
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
	//var err error
	//var resolutionIdentifier string // For logging: "AppID 'xyz'" or "hostname 'abc.com'"
	//var originalHostForLog string = r.Host // Capture original host for logging before it's modified

	traceID := uuid.New().String()

	// Handle debug API endpoints first
	if strings.HasPrefix(r.URL.Path, "/debug/application") {
		// TODO(tom) STOPSHIP deprecate all this
		if r.URL.Path == "/debug/application" && r.Method == http.MethodPost {
			p.debugHandler.HandleCreateApplication(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/debug/application/") && r.Method == http.MethodDelete {
			p.debugHandler.HandleDeleteApplication(w, r)
			return
		}
		// Handle upload endpoints
		if strings.Contains(r.URL.Path, "/upload") {
			if strings.HasSuffix(r.URL.Path, "/upload") && r.Method == http.MethodPost {
				p.debugHandler.HandleUpload(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/upload/status") && r.Method == http.MethodGet {
				p.debugHandler.HandleUploadStatus(w, r)
				return
			}
		}
		// Handle install endpoint
		if strings.HasSuffix(r.URL.Path, "/install-dev") && r.Method == http.MethodPost {
			p.debugHandler.HandleInstallDevApplication(w, r)
			return
		}
		// Handle status endpoint
		if strings.HasSuffix(r.URL.Path, "/status") && r.Method == http.MethodGet {
			p.debugHandler.HandleApplicationStatus(w, r)
			return
		}
		// Handle logs endpoint
		if strings.HasSuffix(r.URL.Path, "/logs") && r.Method == http.MethodGet {
			p.debugHandler.HandleLogStream(w, r)
			return
		}
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Login endpoints

	adminHost, err := p.GetServiceHost("MBtskI6D")
	if err != nil {
		http.Error(w, "Service not found for admin", http.StatusNotFound)
		log.Printf("<%s> %s %s 404 [Service not found]", traceID, r.Host, r.URL.Path)
		return
	}

	if r.URL.Path == "/public/login" {
		middleware.CorsMiddleware(w, r, func(w http.ResponseWriter, r *http.Request) {
			login.HandleLogin(w, r, adminHost)
		})
		log.Printf("<%s> %s %s", traceID, r.Host, r.URL.Path)
		return
	}
	if r.URL.Path == "/public/logout" {
		middleware.CorsMiddleware(w, r, login.HandleLogout)
		log.Printf("<%s> %s %s", traceID, r.Host, r.URL.Path)
		return
	}
	if r.URL.Path == "/public/access_token" {
		middleware.CorsMiddleware(w, r, func(w http.ResponseWriter, r *http.Request) {
			login.HandleAccessToken(w, r, adminHost)
		})
		log.Printf("<%s> %s %s", traceID, r.Host, r.URL.Path)
		return
	}

	// Validate authorization for API endpoints
	if r.Method != "OPTIONS" {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("<%s> %s %s => 401 [Missing token]", traceID, r.Host, r.URL.Path)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		valid := token == p.internalSecret
		if !valid {
			valid = access.ValidateAccessToken(token)
		}
		if !valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("<%s> %s %s => 401 [Invalid token]", traceID, r.Host, r.URL.Path)
			return
		}
	}

	// Application registration endpoints
	if r.URL.Path == "/apps/register" {
		middleware.CorsMiddleware(w, r, func(w http.ResponseWriter, r *http.Request) {
			app_handlers.HandleRegistration(w, r)
		})
		log.Printf("<%s> %s %s", traceID, r.Host, r.URL.Path)
		return
	}
	if r.URL.Path == "/apps/install" {
		middleware.CorsMiddleware(w, r, func(w http.ResponseWriter, r *http.Request) {
			app_handlers.HandleInstall(w, r)
		})
		log.Printf("<%s> %s %s", traceID, r.Host, r.URL.Path)
		return
	}

	// Event endpoints
	if r.URL.Path == "/events/publish" {
		middleware.CorsMiddleware(w, r, func(w http.ResponseWriter, r *http.Request) {
			event_handlers.HandleEventPublish(w, r, p.eventManager, p.pm)
		})
		log.Printf("<%s> %s %s", traceID, r.Host, r.URL.Path)
		return
	}
	if r.URL.Path == "/events/poll" {
		middleware.CorsMiddleware(w, r, func(w http.ResponseWriter, r *http.Request) {
			event_handlers.HandleEventPoll(w, r, p.pm)
		})
		log.Printf("<%s> %s %s", traceID, r.Host, r.URL.Path)
		return
	}

	// Look for an application ID in the path string
	var instance *processes.AppInstance
	var port int
	var targetPath string
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 1 {
		appID := parts[1]
		if appID != "" {
			instance, port, err = p.pm.GetAppInstanceByID(appID)
			if err != nil {
				http.Error(w, "Service not found for app ID "+appID, http.StatusNotFound)
				log.Printf("<%s> %s %s 404 [Service not found]", traceID, r.Host, r.URL.Path)
				return
			}
			if instance == nil {
				http.Error(w, "Service unavailable for app ID "+appID, http.StatusServiceUnavailable)
				log.Printf("<%s> %s %s 404 [No active instances]", traceID, r.Host, r.URL.Path)
				return
			}
			targetPath = r.URL.Path[len("/"+appID+"/"):]
		}
	}

	if instance != nil {
		// Token is valid, proxy the request
		targetURL := &url.URL{
			Scheme: "http", // Backend services are HTTP
			Host:   "localhost:" + strconv.Itoa(port),
		}

		origPath := r.URL.Path
		reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
		reverseProxy.Transport = p.transport
		r.Host = targetURL.Host
		r.URL.Path = targetPath
		r.Header.Add("X-Trace-ID", traceID)

		log.Printf("<%s> %s %s => %s", traceID, r.Host, origPath, targetURL.String())
		reverseProxy.ServeHTTP(w, r)
		return
	}

	/*
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
		} else if strings.HasSuffix(originalHostForLog, "."+p.host) {
			// Fallback to hostname-based routing
			hostname := strings.TrimSuffix(originalHostForLog, "."+p.host)
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
		} else {
			http.Error(w, "Request for invalid host name "+originalHostForLog, http.StatusServiceUnavailable)
			log.Printf("<%s> %s%s 404 [Invalid host name]", traceID, originalHostForLog, r.URL.Path)
			return
		}
	*/

	// TODO(tom) STOPSHIP remove this once we consolidate domains
	/*
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
	*/

	/*
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
	*/

	// No matching route found
	http.Error(w, "Not Found", http.StatusNotFound)
	log.Printf("<%s> %s %s 404 [No route found]", traceID, r.Host, r.URL.Path)
}

func (p *Proxy) GetServiceHost(instanceID string) (string, error) {
	_, port, err := p.pm.GetAppInstanceByID(instanceID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://localhost:%d", port), nil
}

// Stop gracefully shuts down the proxy server.
func (p *Proxy) Stop() error {
	if p.server == nil {
		log.Printf("Proxy server was not running or not initialized, nothing to stop.")
		return nil
	}
	log.Printf("Stopping HTTPS proxy server...")
	return p.server.Shutdown(context.TODO()) // Use context.WithTimeout for graceful shutdown if needed
}
