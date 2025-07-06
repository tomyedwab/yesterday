package yesterdaygo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// EventPublisher provides reliable event publishing with queuing and retry logic
type EventPublisher struct {
	client        *Client
	queue         []PendingEvent
	queueMu       sync.RWMutex
	retryBackoff  time.Duration
	maxRetries    int
	batchSize     int
	running       bool
	runningMu     sync.RWMutex
	stopCh        chan struct{}
	flushCh       chan chan error
	wg            sync.WaitGroup
}

// PendingEvent represents an event awaiting publication
type PendingEvent struct {
	ID          string      `json:"id"`
	EventType   string      `json:"eventType"`
	Payload     interface{} `json:"payload"`
	Attempts    int         `json:"attempts"`
	LastAttempt time.Time   `json:"lastAttempt"`
}

// PublisherOption represents a functional option for configuring the EventPublisher
type PublisherOption func(*EventPublisher)

// WithRetryBackoff sets the base backoff duration for retry attempts
func WithRetryBackoff(backoff time.Duration) PublisherOption {
	return func(p *EventPublisher) {
		p.retryBackoff = backoff
	}
}

// WithMaxRetries sets the maximum retry attempts per event
func WithMaxRetries(maxRetries int) PublisherOption {
	return func(p *EventPublisher) {
		p.maxRetries = maxRetries
	}
}

// WithBatchSize sets the number of events to send in a single request
func WithBatchSize(batchSize int) PublisherOption {
	return func(p *EventPublisher) {
		p.batchSize = batchSize
	}
}

// NewEventPublisher creates a new EventPublisher with the given client and options
func NewEventPublisher(client *Client, options ...PublisherOption) *EventPublisher {
	publisher := &EventPublisher{
		client:       client,
		queue:        make([]PendingEvent, 0),
		retryBackoff: 1 * time.Second,
		maxRetries:   10,
		batchSize:    1,
		stopCh:       make(chan struct{}),
		flushCh:      make(chan chan error, 1),
	}

	// Apply options
	for _, option := range options {
		option(publisher)
	}

	// Start background publishing goroutine
	publisher.start()

	return publisher
}

// generateEventID generates a unique identifier for an event
func generateEventID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("event_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// generateClientID generates a random client ID for the publish request
func generateClientID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("client_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// PublishEvent adds an event to the publish queue and triggers immediate publish attempt
func (p *EventPublisher) PublishEvent(eventType string, payload interface{}) error {
	event := PendingEvent{
		ID:          generateEventID(),
		EventType:   eventType,
		Payload:     payload,
		Attempts:    0,
		LastAttempt: time.Time{},
	}

	p.queueMu.Lock()
	wasEmpty := len(p.queue) == 0
	p.queue = append(p.queue, event)
	p.queueMu.Unlock()

	// If queue was empty, this will trigger immediate processing
	// Otherwise, the background goroutine will pick it up
	if wasEmpty {
		select {
		case <-p.stopCh:
			return fmt.Errorf("publisher is stopped")
		default:
			// Non-blocking - background goroutine will process
		}
	}

	return nil
}

// start begins the background publishing goroutine
func (p *EventPublisher) start() {
	p.runningMu.Lock()
	defer p.runningMu.Unlock()

	if p.running {
		return
	}

	p.running = true
	p.wg.Add(1)
	go p.publishLoop()
}

// Stop gracefully stops the event publisher
func (p *EventPublisher) Stop() {
	p.runningMu.Lock()
	if !p.running {
		p.runningMu.Unlock()
		return
	}
	p.running = false
	p.runningMu.Unlock()

	close(p.stopCh)
	p.wg.Wait()
}

// FlushEvents blocks until all queued events are published or timeout is reached
func (p *EventPublisher) FlushEvents(timeout time.Duration) error {
	// Check if there are any events to flush
	p.queueMu.RLock()
	queueEmpty := len(p.queue) == 0
	p.queueMu.RUnlock()

	if queueEmpty {
		return nil
	}

	// Create response channel for flush operation
	responseCh := make(chan error, 1)

	// Send flush request to background goroutine
	select {
	case p.flushCh <- responseCh:
		// Request sent successfully
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting to initiate flush")
	case <-p.stopCh:
		return fmt.Errorf("publisher is stopped")
	}

	// Wait for flush completion or timeout
	select {
	case err := <-responseCh:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for events to be published")
	case <-p.stopCh:
		return fmt.Errorf("publisher stopped during flush")
	}
}

// IsRunning returns whether the publisher is currently running
func (p *EventPublisher) IsRunning() bool {
	p.runningMu.RLock()
	defer p.runningMu.RUnlock()
	return p.running
}

// GetQueueLength returns the current number of events in the queue
func (p *EventPublisher) GetQueueLength() int {
	p.queueMu.RLock()
	defer p.queueMu.RUnlock()
	return len(p.queue)
}

// publishLoop runs the background publishing process
func (p *EventPublisher) publishLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // Check queue frequently
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case responseCh := <-p.flushCh:
			// Handle flush request
			err := p.processFlush()
			select {
			case responseCh <- err:
			case <-p.stopCh:
				return
			}
		case <-ticker.C:
			// Regular processing
			p.processQueue()
		}
	}
}

// processQueue processes events in the queue
func (p *EventPublisher) processQueue() {
	p.queueMu.Lock()
	if len(p.queue) == 0 {
		p.queueMu.Unlock()
		return
	}

	// Take the first event for processing
	event := p.queue[0]
	p.queueMu.Unlock()

	// Check if we should retry this event (exponential backoff)
	if !event.LastAttempt.IsZero() {
		backoffDuration := p.calculateBackoff(event.Attempts)
		if time.Since(event.LastAttempt) < backoffDuration {
			return // Not time to retry yet
		}
	}

	// Attempt to publish the event
	success := p.publishSingleEvent(&event)

	p.queueMu.Lock()
	if success {
		// Remove the event from queue
		if len(p.queue) > 0 && p.queue[0].ID == event.ID {
			p.queue = p.queue[1:]
		}
	} else {
		// Update the event with retry information
		if len(p.queue) > 0 && p.queue[0].ID == event.ID {
			p.queue[0] = event
			// If max retries exceeded, remove the event
			if event.Attempts >= p.maxRetries {
				p.queue = p.queue[1:]
			}
		}
	}
	p.queueMu.Unlock()
}

// processFlush processes all events until queue is empty
func (p *EventPublisher) processFlush() error {
	maxWait := 30 * time.Second
	start := time.Now()

	for {
		p.queueMu.RLock()
		queueLen := len(p.queue)
		p.queueMu.RUnlock()

		if queueLen == 0 {
			return nil
		}

		if time.Since(start) > maxWait {
			return fmt.Errorf("timeout: %d events remain in queue after %v", queueLen, maxWait)
		}

		// Process events more aggressively during flush
		p.processQueue()
		time.Sleep(10 * time.Millisecond)
	}
}

// calculateBackoff calculates the backoff duration for retry attempts
func (p *EventPublisher) calculateBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return 0
	}

	// Exponential backoff: 1s, 2s, 4s, 8s, up to 5 minutes maximum
	backoff := p.retryBackoff
	for i := 1; i < attempts && i < 10; i++ {
		backoff *= 2
	}

	maxBackoff := 5 * time.Minute
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

// publishSingleEvent attempts to publish a single event to the API
func (p *EventPublisher) publishSingleEvent(event *PendingEvent) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Update attempt tracking
	event.Attempts++
	event.LastAttempt = time.Now()

	// Generate a random client ID for this request
	clientID := generateClientID()

	// Prepare the request payload
	payload := map[string]interface{}{
		"eventType": event.EventType,
		"payload":   event.Payload,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		// JSON marshaling error - this event is malformed, don't retry
		return true
	}

	// Build the URL with client ID parameter
	publishURL := "/api/publish?cid=" + url.QueryEscape(clientID)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.client.baseURL+publishURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	// Add authentication header if available
	if token := p.client.getAccessToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Execute the request
	resp, err := p.client.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true // Success
	}

	// For client errors (4xx), don't retry
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return true // Don't retry client errors
	}

	// For server errors (5xx), retry
	return false
}
