package yesterdaygo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// EventPoller manages event number polling for detecting data changes
type EventPoller struct {
	client             *Client
	currentEventNumber int64
	pollInterval       time.Duration
	subscribers        []chan int64
	stopCh             chan struct{}
	mu                 sync.RWMutex // Protects currentEventNumber and subscribers
	running            bool
	runningMu          sync.Mutex // Protects running state
}

// PollResponse represents the response from the poll endpoint
type PollResponse struct {
	EventNumber int64 `json:"event_number"`
}

// NewEventPoller creates a new event poller for the given client
func NewEventPoller(client *Client) *EventPoller {
	return &EventPoller{
		client:       client,
		pollInterval: 5 * time.Second, // Default 5 second interval
		subscribers:  make([]chan int64, 0),
		stopCh:       make(chan struct{}),
	}
}

// GetCurrentEventNumber returns the current event number in a thread-safe manner
func (ep *EventPoller) GetCurrentEventNumber() int64 {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.currentEventNumber
}

// setCurrentEventNumber sets the current event number and notifies subscribers
func (ep *EventPoller) setCurrentEventNumber(eventNumber int64) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	
	if eventNumber <= ep.currentEventNumber {
		return // No change or older event
	}
	
	ep.currentEventNumber = eventNumber
	
	// Notify all subscribers of the new event number
	for _, subscriber := range ep.subscribers {
		select {
		case subscriber <- eventNumber:
		default:
			// Non-blocking send - if subscriber can't receive, skip
		}
	}
}

// StartEventPolling starts the background event polling goroutine
func (ep *EventPoller) StartEventPolling(interval time.Duration) error {
	ep.runningMu.Lock()
	defer ep.runningMu.Unlock()
	
	if ep.running {
		return fmt.Errorf("event polling is already running")
	}
	
	if interval > 0 {
		ep.pollInterval = interval
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
	for _, subscriber := range ep.subscribers {
		close(subscriber)
	}
	ep.subscribers = make([]chan int64, 0)
	ep.mu.Unlock()
}

// SubscribeToEvents returns a channel that receives event number updates
func (ep *EventPoller) SubscribeToEvents() <-chan int64 {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	
	ch := make(chan int64, 10) // Buffered channel to prevent blocking
	ep.subscribers = append(ep.subscribers, ch)
	
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	currentEventNumber := ep.GetCurrentEventNumber()
	
	// Create the poll URL with the current event number
	pollURL := fmt.Sprintf("/api/poll?e=%d", currentEventNumber)
	
	resp, err := ep.client.Get(ctx, pollURL, nil)
	if err != nil {
		// Log error but continue polling
		// In a production implementation, you might want to use a proper logger
		return
	}
	defer resp.Body.Close()
	
	// Handle 304 Not Modified - no new events
	if resp.StatusCode == http.StatusNotModified {
		return
	}
	
	// Handle successful response
	if resp.StatusCode == http.StatusOK {
		var pollResponse PollResponse
		if err := json.NewDecoder(resp.Body).Decode(&pollResponse); err != nil {
			// Log error but continue polling
			return
		}
		
		// Update event number if it has changed
		ep.setCurrentEventNumber(pollResponse.EventNumber)
		return
	}
	
	// Handle other status codes as errors
	// Log error but continue polling
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
func (ep *EventPoller) WaitForEvent(ctx context.Context) (int64, error) {
	eventCh := ep.SubscribeToEvents()
	
	select {
	case eventNumber := <-eventCh:
		return eventNumber, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// GetSubscriberCount returns the number of active event subscribers
func (ep *EventPoller) GetSubscriberCount() int {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return len(ep.subscribers)
}
