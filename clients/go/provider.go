package yesterdaygo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// DataProvider provides type-safe data access with automatic refresh on event changes
type DataProvider[T any] struct {
	client          *Client
	uri             string
	params          map[string]interface{}
	data            T
	lastEventNumber int64
	refreshCallback func(T)
	mu              sync.RWMutex // Protects data, lastEventNumber, and refreshCallback
	//eventSubscription <-chan int64
	ctx            context.Context
	cancel         context.CancelFunc
	isSubscribed   bool
	subscriptionMu sync.Mutex // Protects subscription state
}

// NewDataProvider creates a new generic data provider
func NewDataProvider[T any](client *Client, uri string, params map[string]interface{}) *DataProvider[T] {
	ctx, cancel := context.WithCancel(context.Background())

	return &DataProvider[T]{
		client: client,
		uri:    uri,
		params: params,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Get returns the cached data, refreshing it if the event number has changed
func (dp *DataProvider[T]) Get() (T, error) {
	var zero T

	// Check if we need to refresh based on current event number
	//poller := dp.client.GetEventPoller()
	var currentEventNumber int64 = 1 // poller.GetCurrentEventNumber()

	dp.mu.RLock()
	needsRefresh := dp.lastEventNumber < currentEventNumber || dp.lastEventNumber == 0
	cachedData := dp.data
	dp.mu.RUnlock()

	if needsRefresh {
		if err := dp.Refresh(); err != nil {
			return zero, fmt.Errorf("failed to refresh data: %w", err)
		}

		// Return the freshly refreshed data
		dp.mu.RLock()
		cachedData = dp.data
		dp.mu.RUnlock()
	}

	return cachedData, nil
}

// Refresh manually refreshes the data from the API
func (dp *DataProvider[T]) Refresh() error {
	// Build the request URL with parameters
	requestURL := dp.uri
	if len(dp.params) > 0 {
		values := url.Values{}
		for key, value := range dp.params {
			values.Add(key, fmt.Sprintf("%v", value))
		}
		requestURL += "?" + values.Encode()
	}

	// Make the API request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := dp.client.Get(ctx, requestURL, nil)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Parse the response
	var newData T
	if err := json.NewDecoder(resp.Body).Decode(&newData); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Update the cached data and event number
	//poller := dp.client.GetEventPoller()
	//currentEventNumber := poller.GetCurrentEventNumber()
	var currentEventNumber int64 = 1

	dp.mu.Lock()
	dp.data = newData
	dp.lastEventNumber = currentEventNumber
	callback := dp.refreshCallback
	dp.mu.Unlock()

	// Call the refresh callback if one is set
	if callback != nil {
		callback(newData)
	}

	return nil
}

// Subscribe registers a callback for automatic data refresh notifications
func (dp *DataProvider[T]) Subscribe(callback func(T)) error {
	dp.subscriptionMu.Lock()
	defer dp.subscriptionMu.Unlock()

	if dp.isSubscribed {
		return fmt.Errorf("data provider is already subscribed")
	}

	// Set the callback
	dp.mu.Lock()
	dp.refreshCallback = callback
	dp.mu.Unlock()

	// Subscribe to event notifications
	//poller := dp.client.GetEventPoller()
	//dp.eventSubscription = poller.SubscribeToEvents()
	dp.isSubscribed = true

	// Start the event listening goroutine
	go dp.eventLoop()

	return nil
}

// Unsubscribe stops automatic data refresh notifications
func (dp *DataProvider[T]) Unsubscribe() {
	dp.subscriptionMu.Lock()
	defer dp.subscriptionMu.Unlock()

	if !dp.isSubscribed {
		return
	}

	dp.isSubscribed = false
	dp.cancel() // This will stop the event loop

	// Clear the callback
	dp.mu.Lock()
	dp.refreshCallback = nil
	dp.mu.Unlock()
}

// eventLoop handles automatic refresh when events are received
func (dp *DataProvider[T]) eventLoop() {
	for {
		select {
		/*
			case eventNumber := <-dp.eventSubscription:
				// Check if we need to refresh
				dp.mu.RLock()
				needsRefresh := dp.lastEventNumber < eventNumber
				dp.mu.RUnlock()

				if needsRefresh {
					// Refresh data and call callback
					if err := dp.Refresh(); err != nil {
						// In a production system, you might want to log this error
						continue
					}
				}
		*/
		case <-dp.ctx.Done():
			return // Subscription cancelled
		}
	}
}

// GetLastEventNumber returns the event number when data was last fetched
func (dp *DataProvider[T]) GetLastEventNumber() int64 {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return dp.lastEventNumber
}

// GetURI returns the API endpoint URI
func (dp *DataProvider[T]) GetURI() string {
	return dp.uri
}

// GetParams returns a copy of the query parameters
func (dp *DataProvider[T]) GetParams() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range dp.params {
		result[k] = v
	}
	return result
}

// SetParams updates the query parameters and triggers a refresh if subscribed
func (dp *DataProvider[T]) SetParams(params map[string]interface{}) error {
	dp.params = params

	// If we're subscribed, trigger a refresh
	dp.subscriptionMu.Lock()
	isSubscribed := dp.isSubscribed
	dp.subscriptionMu.Unlock()

	if isSubscribed {
		return dp.Refresh()
	}

	return nil
}

// IsSubscribed returns whether the data provider is subscribed to events
func (dp *DataProvider[T]) IsSubscribed() bool {
	dp.subscriptionMu.Lock()
	defer dp.subscriptionMu.Unlock()
	return dp.isSubscribed
}

// Close cleans up the data provider resources
func (dp *DataProvider[T]) Close() {
	dp.Unsubscribe()
	dp.cancel()
}
