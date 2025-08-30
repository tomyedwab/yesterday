package processes

import (
	"context"
	"log"
	"sync"
)

// StaticAppConfig represents a static application configuration that overrides admin service data
type StaticAppConfig struct {
	InstanceID    string
	HostName      string
	PkgPath       string
	Subscriptions map[string]bool
}

// AdminInstanceProvider fetches application instances from the admin service
type AdminInstanceProvider struct {
	adminAppID string                     // Application ID of the admin service
	staticApps map[string]StaticAppConfig // keyed by InstanceID
	installDir string

	mu          sync.RWMutex
	instances   []AppInstance
	initialized bool

	debugInstances   map[string]AppInstance // Keyed by InstanceID
	debugInstancesMu sync.RWMutex

	stopChan       chan struct{}
	internalSecret string
}

// NewAdminInstanceProvider creates a new provider that fetches from the admin service
func NewAdminInstanceProvider(adminAppID, internalSecret, installDir string, staticApps []StaticAppConfig) *AdminInstanceProvider {
	staticAppMap := make(map[string]StaticAppConfig)
	for _, app := range staticApps {
		staticAppMap[app.InstanceID] = app
	}

	return &AdminInstanceProvider{
		adminAppID:     adminAppID,
		staticApps:     staticAppMap,
		installDir:     installDir,
		instances:      make([]AppInstance, 0),
		initialized:    false,
		stopChan:       make(chan struct{}),
		internalSecret: internalSecret,
		debugInstances: make(map[string]AppInstance),
	}
}

// Start begins fetching application data and polling for changes
func (p *AdminInstanceProvider) Start(ctx context.Context) error {
	p.initializeWithStaticAppsOnly()
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
				InstanceID:    config.InstanceID,
				HostName:      config.HostName,
				PkgPath:       config.PkgPath,
				Subscriptions: config.Subscriptions,
			})
		}
		return staticInstances, nil
	}

	// Return a copy to prevent modification
	instCopy := make([]AppInstance, len(p.instances))
	copy(instCopy, p.instances)

	// Add debug instances
	p.debugInstancesMu.RLock()
	defer p.debugInstancesMu.RUnlock()
	for _, instance := range p.debugInstances {
		instCopy = append(instCopy, instance)
	}

	return instCopy, nil
}

// AddDebugInstance adds a temporary debug instance to the provider
func (p *AdminInstanceProvider) AddDebugInstance(instance AppInstance) {
	p.debugInstancesMu.Lock()
	defer p.debugInstancesMu.Unlock()
	p.debugInstances[instance.InstanceID] = instance
	log.Printf("Added debug instance %s", instance.InstanceID)
}

// RemoveDebugInstance removes a temporary debug instance from the provider
func (p *AdminInstanceProvider) RemoveDebugInstance(instanceID string) {
	p.debugInstancesMu.Lock()
	defer p.debugInstancesMu.Unlock()
	delete(p.debugInstances, instanceID)
	log.Printf("Removed debug instance %s", instanceID)
}

// initializeWithStaticAppsOnly initializes the provider with only static apps when admin service is unavailable
func (p *AdminInstanceProvider) initializeWithStaticAppsOnly() {
	instances := make([]AppInstance, 0, len(p.staticApps))
	for _, config := range p.staticApps {
		instances = append(instances, AppInstance{
			InstanceID:    config.InstanceID,
			HostName:      config.HostName,
			PkgPath:       config.PkgPath,
			Subscriptions: config.Subscriptions,
		})
	}

	p.mu.Lock()
	p.instances = instances
	p.initialized = true
	p.mu.Unlock()

	log.Printf("Initialized with %d static applications only", len(instances))
}
