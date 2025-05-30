package processes

import (
	"fmt"
	"net/http"
	"time"
)

// HealthChecker defines the interface for performing health checks on a managed process.
type HealthChecker interface {
	// Check performs a health check on the given process.
	// It returns the determined ProcessState (e.g., StateRunning if healthy, StateUnhealthy if not).
	// An error is returned if the check itself fails due to network issues or misconfiguration.
	Check(process *ManagedProcess) (ProcessState, error)
}

// HTTPHealthChecker implements HealthChecker using HTTP GET requests.
// It checks the /api/status endpoint of a subprocess.
type HTTPHealthChecker struct {
	client         *http.Client
	requestTimeout time.Duration // Timeout for a single HTTP health check request
}

// NewHTTPHealthChecker creates a new HTTPHealthChecker.
// requestTimeout specifies the timeout for each health check HTTP request.
func NewHTTPHealthChecker(requestTimeout time.Duration) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		client: &http.Client{
			Timeout: requestTimeout,
		},
		requestTimeout: requestTimeout,
	}
}

// Check performs an HTTP health check on the given ManagedProcess.
// It targets http://localhost:<PORT>/api/status.
func (h *HTTPHealthChecker) Check(process *ManagedProcess) (ProcessState, error) {
	if process.Port <= 0 {
		return StateFailed, fmt.Errorf("invalid port %d for health check on instance %s", process.Port, process.Instance.InstanceID)
	}

	url := fmt.Sprintf("http://localhost:%d/api/status", process.Port)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return StateFailed, fmt.Errorf("failed to create health check request for %s: %w", process.Instance.InstanceID, err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		// Network error, timeout, connection refused, etc.
		return StateUnhealthy, fmt.Errorf("health check HTTP request for %s failed: %w", process.Instance.InstanceID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return StateRunning, nil // Healthy
	}

	// Non-200 status code indicates an issue with the service itself.
	return StateUnhealthy, fmt.Errorf("health check for %s at %s returned status %s", process.Instance.InstanceID, url, resp.Status)
}
