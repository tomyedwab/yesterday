package processes

import (
	"fmt"
	"net"
	"sync"
)

// PortManager handles the allocation and deallocation of TCP ports for subprocesses.
type PortManager struct {
	mu            sync.Mutex
	minPort       int
	maxPort       int
	allocated     map[int]bool // Tracks allocated ports
	nextCandidate int          // Next port to try allocating
}

// NewPortManager creates a new PortManager instance.
// It requires a minimum and maximum port to define the range for allocation.
func NewPortManager(minPort, maxPort int) (*PortManager, error) {
	if minPort <= 0 || maxPort <= 0 || minPort > maxPort {
		return nil, fmt.Errorf("invalid port range: min %d, max %d", minPort, maxPort)
	}
	return &PortManager{
		minPort:       minPort,
		maxPort:       maxPort,
		allocated:     make(map[int]bool),
		nextCandidate: minPort,
	}, nil
}

// AllocatePort finds and allocates an available TCP port within the configured range.
// It returns the allocated port or an error if no port is available or if an error occurs.
func (pm *PortManager) AllocatePort() (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	firstCandidate := pm.nextCandidate

	for {
		portToTry := pm.nextCandidate

		// Increment nextCandidate for the next attempt, wrapping around if necessary
		pm.nextCandidate++
		if pm.nextCandidate > pm.maxPort {
			pm.nextCandidate = pm.minPort
		}

		if pm.allocated[portToTry] {
			if pm.nextCandidate == firstCandidate { // Scanned all and returned to start, all allocated
				return 0, fmt.Errorf("no available ports in range [%d-%d]", pm.minPort, pm.maxPort)
			}
			continue // Port already marked as allocated, try next
		}

		// Check if the port is actually available by trying to listen on it
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", portToTry))
		if err == nil {
			// Port is available
			l.Close() // Close the listener immediately
			pm.allocated[portToTry] = true
			return portToTry, nil
		}

		// If we've searched the entire range and couldn't find a free port (even those not in pm.allocated but busy)
		if pm.nextCandidate == firstCandidate {
			return 0, fmt.Errorf("no available ports in range [%d-%d] after checking system availability", pm.minPort, pm.maxPort)
		}
	}
}

// ReleasePort marks a previously allocated port as available again.
func (pm *PortManager) ReleasePort(port int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if port < pm.minPort || port > pm.maxPort {
		// Port is outside the managed range, nothing to do or log an error
		return
	}

	delete(pm.allocated, port)
}
