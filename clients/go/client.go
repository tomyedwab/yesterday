package yesterdaygo

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Client represents the main Yesterday API client
type Client struct {
	baseURL          string
	httpClient       *http.Client
	refreshTokenPath string
	accessToken      string
	mu               sync.RWMutex    // Protects accessToken
	eventPoller      *EventPoller    // Event polling system
	eventPublisher   *EventPublisher // Event publishing system
}

// ClientOption represents a functional option for configuring the Client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithRefreshTokenPath sets a custom path for storing the refresh token
func WithRefreshTokenPath(path string) ClientOption {
	return func(c *Client) {
		c.refreshTokenPath = path
	}
}

// NewClient creates a new Yesterday API client with the given base URL and options
func NewClient(baseURL string, options ...ClientOption) *Client {
	// Set default refresh token path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	defaultRefreshTokenPath := filepath.Join(homeDir, ".yesterday", "refresh_token")

	client := &Client{
		baseURL:          baseURL,
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		refreshTokenPath: defaultRefreshTokenPath,
	}

	// Initialize event poller and publisher
	client.eventPoller = NewEventPoller(client)
	client.eventPublisher = NewEventPublisher(client)

	// Apply options
	for _, option := range options {
		option(client)
	}

	return client
}

// GetBaseURL returns the client's base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetHTTPClient returns the underlying HTTP client
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// GetRefreshTokenPath returns the path where refresh tokens are stored
func (c *Client) GetRefreshTokenPath() string {
	return c.refreshTokenPath
}

// setAccessToken sets the access token in a thread-safe manner
func (c *Client) setAccessToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = token
}

// getAccessToken gets the access token in a thread-safe manner
func (c *Client) getAccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

// clearAccessToken clears the access token in a thread-safe manner
func (c *Client) clearAccessToken() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = ""
}

// makeRequest performs an HTTP request with authentication headers
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	url := c.baseURL + path
	
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header if we have an access token
	if token := c.getAccessToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default content type for JSON requests
	if req.Header.Get("Content-Type") == "" && (method == "POST" || method == "PUT" || method == "PATCH") {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// Get performs a GET request to the specified path
func (c *Client) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return c.makeRequest(ctx, "GET", path, nil, headers)
}

// Post performs a POST request to the specified path  
func (c *Client) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.makeRequest(ctx, "POST", path, body, headers)
}

// Put performs a PUT request to the specified path
func (c *Client) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.makeRequest(ctx, "PUT", path, body, headers)
}

// Delete performs a DELETE request to the specified path
func (c *Client) Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return c.makeRequest(ctx, "DELETE", path, nil, headers)
}

// Initialize performs initial setup including token refresh
func (c *Client) Initialize(ctx context.Context) error {
	// Try to refresh access token on initialization
	if err := c.RefreshAccessToken(ctx); err != nil {
		// Log the error but don't fail initialization - user can still login
		// In a real implementation, you might want to use a proper logger here
		return fmt.Errorf("failed to refresh access token during initialization: %w", err)
	}
	return nil
}

// GetEventPoller returns the event polling system
func (c *Client) GetEventPoller() *EventPoller {
	return c.eventPoller
}

// GetEventPublisher returns the event publishing system
func (c *Client) GetEventPublisher() *EventPublisher {
	return c.eventPublisher
}
