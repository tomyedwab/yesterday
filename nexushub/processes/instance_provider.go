package processes

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"
)

// Application represents the application data structure returned by the admin service
type Application struct {
	InstanceID  string `json:"instanceId"`
	AppID       string `json:"appId"`
	DisplayName string `json:"displayName"`
	HostName    string `json:"hostName"`
}

// StaticAppConfig represents a static application configuration that overrides admin service data
type StaticAppConfig struct {
	InstanceID string
	HostName   string
	PkgPath    string
}

// AdminInstanceProvider fetches application instances from the admin service
type AdminInstanceProvider struct {
	adminAppID string                     // Application ID of the admin service
	staticApps map[string]StaticAppConfig // keyed by InstanceID
	installDir string

	mu          sync.RWMutex
	instances   []AppInstance
	lastEventID int
	initialized bool

	stopChan       chan struct{}
	pollInterval   time.Duration
	httpClient     *http.Client
	maxRetries     int
	retryDelay     time.Duration
	internalSecret string
}

// NewAdminInstanceProvider creates a new provider that fetches from the admin service
func NewAdminInstanceProvider(adminAppID, internalSecret, installDir string, staticApps []StaticAppConfig) *AdminInstanceProvider {
	staticAppMap := make(map[string]StaticAppConfig)
	for _, app := range staticApps {
		staticAppMap[app.InstanceID] = app
	}

	// Create HTTP client with TLS config similar to cross-service requests
	dialer := net.Dialer{
		Timeout:   600 * time.Second,
		KeepAlive: 600 * time.Second,
	}
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 180 * time.Second,
	}

	return &AdminInstanceProvider{
		adminAppID:   adminAppID,
		staticApps:   staticAppMap,
		installDir:   installDir,
		instances:    make([]AppInstance, 0),
		lastEventID:  0,
		initialized:  false,
		stopChan:     make(chan struct{}),
		pollInterval: 5 * time.Second,
		maxRetries:   5,
		retryDelay:   2 * time.Second,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   60 * time.Second,
		},
		internalSecret: internalSecret,
	}
}

// Start begins fetching application data and polling for changes
func (p *AdminInstanceProvider) Start(ctx context.Context) error {
	// Try to fetch initial data with retries
	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		if err := p.fetchApplications(ctx); err != nil {
			lastErr = err
			log.Printf("Failed to fetch initial applications (attempt %d/%d): %v", attempt+1, p.maxRetries, err)

			if attempt < p.maxRetries-1 {
				delay := time.Duration(math.Pow(2, float64(attempt))) * p.retryDelay
				log.Printf("Retrying in %v...", delay)

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
		} else {
			p.mu.Lock()
			p.initialized = true
			p.mu.Unlock()
			log.Printf("Successfully initialized AdminInstanceProvider with %d applications", len(p.instances))
			break
		}
	}

	// If we couldn't fetch initial data, start with static apps only
	if !p.initialized {
		log.Printf("Failed to fetch initial applications after %d attempts, starting with static apps only: %v", p.maxRetries, lastErr)
		p.initializeWithStaticAppsOnly()
	}

	// Start polling for changes
	go p.pollForChanges(ctx)

	return nil
}

// Stop stops the polling goroutine
func (p *AdminInstanceProvider) Stop() {
	close(p.stopChan)
}

// GetAppInstances returns the current list of app instances
func (p *AdminInstanceProvider) GetAppInstances(ctx context.Context) ([]AppInstance, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// If we haven't initialized yet, return static apps only
	if !p.initialized && len(p.instances) == 0 {
		log.Printf("AdminInstanceProvider not yet initialized, returning static apps only")
		staticInstances := make([]AppInstance, 0, len(p.staticApps))
		for _, config := range p.staticApps {
			staticInstances = append(staticInstances, AppInstance{
				InstanceID: config.InstanceID,
				HostName:   config.HostName,
				PkgPath:    config.PkgPath,
			})
		}
		return staticInstances, nil
	}

	// Return a copy to prevent modification
	instCopy := make([]AppInstance, len(p.instances))
	copy(instCopy, p.instances)
	return instCopy, nil
}

// makeRequest makes a cross-service request to the admin service
func (p *AdminInstanceProvider) makeRequest(ctx context.Context, method, path, query string, response any) error {
	req := &http.Request{
		Method: method,
		URL:    &url.URL{Scheme: "https", Host: "internal.yesterday.localhost:8443", Path: path, RawQuery: query},
		Header: http.Header{
			"Content-Type":     []string{"application/json"},
			"X-Application-Id": []string{p.adminAppID},
			"Authorization":    []string{"Bearer " + p.internalSecret},
		},
	}
	req = req.WithContext(ctx)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		// Special case for polling - no changes
		return fmt.Errorf("not modified")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("admin service returned status %d: %s", resp.StatusCode, string(body))
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// fetchApplications retrieves applications from the admin service
func (p *AdminInstanceProvider) fetchApplications(ctx context.Context) error {
	var response struct {
		Applications []Application `json:"applications"`
	}

	if err := p.makeRequest(ctx, "GET", "/api/applications", "", &response); err != nil {
		return fmt.Errorf("failed to fetch applications: %w", err)
	}

	// Convert applications to app instances and merge with static configs
	instances := make([]AppInstance, 0, len(response.Applications))
	for _, app := range response.Applications {
		instance := p.convertToAppInstance(app)
		instances = append(instances, instance)
	}

	p.mu.Lock()
	p.instances = instances
	if !p.initialized {
		p.initialized = true
	}
	p.mu.Unlock()

	log.Printf("Fetched %d applications from admin service", len(instances))
	return nil
}

// convertToAppInstance converts an Application to an AppInstance, applying static overrides
func (p *AdminInstanceProvider) convertToAppInstance(app Application) AppInstance {
	instance := AppInstance{
		InstanceID: app.InstanceID,
		HostName:   app.HostName,
		PkgPath:    filepath.Join(p.installDir, app.AppID),
		DebugPort:  0, // Default 0, will be set by static config if available
	}

	// Apply static configuration overrides if they exist
	if staticConfig, exists := p.staticApps[app.InstanceID]; exists {
		// Static apps override all fields
		instance.HostName = staticConfig.HostName
		instance.PkgPath = staticConfig.PkgPath
		log.Printf("Applied static config override for instance %s", app.InstanceID)
	}

	return instance
}

// initializeWithStaticAppsOnly initializes the provider with only static apps when admin service is unavailable
func (p *AdminInstanceProvider) initializeWithStaticAppsOnly() {
	instances := make([]AppInstance, 0, len(p.staticApps))
	for _, config := range p.staticApps {
		instances = append(instances, AppInstance{
			InstanceID: config.InstanceID,
			HostName:   config.HostName,
			PkgPath:    config.PkgPath,
		})
	}

	p.mu.Lock()
	p.instances = instances
	p.initialized = true
	p.mu.Unlock()

	log.Printf("Initialized with %d static applications only", len(instances))
}

// pollForChanges continuously polls the admin service for changes
func (p *AdminInstanceProvider) pollForChanges(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	consecutiveErrors := 0
	maxConsecutiveErrors := 10

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			if err := p.checkForChanges(ctx); err != nil {
				consecutiveErrors++
				log.Printf("Error checking for changes (%d consecutive errors): %v", consecutiveErrors, err)

				// If we have too many consecutive errors, back off polling frequency
				if consecutiveErrors >= maxConsecutiveErrors {
					log.Printf("Too many consecutive errors, backing off polling frequency")
					ticker.Reset(p.pollInterval * 3)
				}
			} else {
				// Reset error count and polling frequency on success
				if consecutiveErrors > 0 {
					log.Printf("Recovered from %d consecutive errors", consecutiveErrors)
					consecutiveErrors = 0
					ticker.Reset(p.pollInterval)
				}
			}
		}
	}
}

// checkForChanges polls the admin service for event changes
func (p *AdminInstanceProvider) checkForChanges(ctx context.Context) error {
	var response struct {
		ID      int    `json:"id"`
		Version string `json:"version"`
	}

	err := p.makeRequest(ctx, "GET", "/api/poll", fmt.Sprintf("e=%d", p.lastEventID+1), &response)
	if err != nil {
		if err.Error() == "not modified" {
			// No changes, this is normal
			return nil
		}
		return fmt.Errorf("failed to poll for changes: %w", err)
	}

	// If we got a new event ID, fetch the latest applications
	if response.ID > p.lastEventID {
		log.Printf("Detected changes in admin service (event ID: %d -> %d), refreshing applications", p.lastEventID, response.ID)
		p.lastEventID = response.ID

		if err := p.fetchApplications(ctx); err != nil {
			return fmt.Errorf("failed to refresh applications after change: %w", err)
		}
	}

	return nil
}
