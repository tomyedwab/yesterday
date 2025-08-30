package yesterdaygo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type EventPublishData struct {
	// The client ID for the publish request, used for deduplication
	ClientID string `json:"clientId"`
	// The event type
	Type string `json:"type"`
	// The event timestamp
	Timestamp time.Time `json:"timestamp"`
}

// EventPoller manages event number polling for detecting data changes
type EventPoller struct {
	client          *Client
	currentEventIds map[string]int
	pollInterval    time.Duration
	subscribers     map[string][]chan int
	stopCh          chan struct{}
	mu              sync.RWMutex // Protects currentEventNumber and subscribers
	running         bool
	runningMu       sync.Mutex // Protects running state
}

// NewEventPoller creates a new event poller for the given client
func NewEventPoller(client *Client) *EventPoller {
	poller := &EventPoller{
		client:          client,
		currentEventIds: make(map[string]int),
		pollInterval:    5 * time.Second, // Default 5 second interval
		subscribers:     make(map[string][]chan int),
		stopCh:          make(chan struct{}),
	}
	poller.StartEventPolling()
	return poller
}

// GetCurrentEventNumber returns the current event number in a thread-safe manner
func (ep *EventPoller) GetCurrentEventId(instanceID string) int {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.currentEventIds[instanceID]
}

// setCurrentEventNumber sets the current event number and notifies subscribers
func (ep *EventPoller) setCurrentEventIds(eventIds map[string]int) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	for instanceID, eventId := range eventIds {
		if eventId <= ep.currentEventIds[instanceID] {
			continue // No change or older event
		}

		ep.client.Log().Printf("App %s has new event ID %d", instanceID, eventId)
		ep.currentEventIds[instanceID] = eventId

		// Notify all subscribers of the new event number
		for _, subscriber := range ep.subscribers[instanceID] {
			select {
			case subscriber <- eventId:
			default:
				// Non-blocking send - if subscriber can't receive, skip
			}
		}
	}
}

// StartEventPolling starts the background event polling goroutine
func (ep *EventPoller) StartEventPolling() error {
	ep.runningMu.Lock()
	defer ep.runningMu.Unlock()

	if ep.running {
		return fmt.Errorf("event polling is already running")
	}

	ep.running = true
	ep.stopCh = make(chan struct{})

	// Start the polling goroutine
	go ep.pollLoop()

	return nil
}

// StopEventPolling stops the background event polling
func (ep *EventPoller) StopEventPolling() {
	ep.runningMu.Lock()
	defer ep.runningMu.Unlock()

	if !ep.running {
		return
	}

	ep.running = false
	close(ep.stopCh)

	// Close all subscriber channels
	ep.mu.Lock()
	for _, subscribers := range ep.subscribers {
		for _, subscriber := range subscribers {
			close(subscriber)
		}
	}
	ep.subscribers = make(map[string][]chan int)
	ep.mu.Unlock()
}

// SubscribeToEvents returns a channel that receives event number updates
func (ep *EventPoller) SubscribeToEvents(instanceID string) <-chan int {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Make sure we are polling for events for this instance
	if _, ok := ep.currentEventIds[instanceID]; !ok {
		ep.currentEventIds[instanceID] = 0
	}

	ch := make(chan int, 10) // Buffered channel to prevent blocking
	ep.subscribers[instanceID] = append(ep.subscribers[instanceID], ch)

	return ch
}

// pollLoop is the main polling loop that runs in a background goroutine
func (ep *EventPoller) pollLoop() {
	ticker := time.NewTicker(ep.pollInterval)
	defer ticker.Stop()

	// Perform initial poll
	ep.performPoll()

	for {
		select {
		case <-ticker.C:
			ep.performPoll()
		case <-ep.stopCh:
			return
		}
	}
}

// performPoll performs a single poll request to the API
func (ep *EventPoller) performPoll() {
	if len(ep.currentEventIds) == 0 {
		ep.client.Log().Printf("No event IDs to poll")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ep.client.Log().Printf("POLL: Polling for events...")
	resp, err := ep.client.Post(ctx, "/events/poll", ep.currentEventIds, nil)
	if err != nil {
		// Log error but continue polling
		ep.client.Log().Printf("POLL: Error: %v", err)
		return
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified - no new events
	if resp.StatusCode == http.StatusNotModified {
		ep.client.Log().Printf("POLL: No new events")
		return
	}

	// Handle successful response
	if resp.StatusCode == http.StatusOK {
		var pollResponse map[string]int
		if err := json.NewDecoder(resp.Body).Decode(&pollResponse); err != nil {
			// Log error but continue polling
			ep.client.Log().Printf("POLL: Invalid response: %v", err)
			return
		}

		// Update event IDs if they have changed
		ep.setCurrentEventIds(pollResponse)
		return
	} else {
		// Handle other status codes as errors
		// Log error but continue polling
		ep.client.Log().Printf("POLL: Unexpected status: %d", resp.StatusCode)
	}
}

// IsRunning returns whether the event poller is currently running
func (ep *EventPoller) IsRunning() bool {
	ep.runningMu.Lock()
	defer ep.runningMu.Unlock()
	return ep.running
}

// SetPollInterval updates the polling interval (only affects future polls)
func (ep *EventPoller) SetPollInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}
	ep.pollInterval = interval
}

// GetPollInterval returns the current polling interval
func (ep *EventPoller) GetPollInterval() time.Duration {
	return ep.pollInterval
}

// WaitForEvent waits for the next event number change with a timeout
func (ep *EventPoller) WaitForEvent(ctx context.Context, instanceID string) (int, error) {
	eventCh := ep.SubscribeToEvents(instanceID)

	select {
	case eventNumber := <-eventCh:
		return eventNumber, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
