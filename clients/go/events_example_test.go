package yesterdaygo_test

import (
	"context"
	"fmt"
	"log"
	"time"

	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

// ExampleEventPoller demonstrates basic event polling usage
func ExampleEventPoller() {
	// Create a new client
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	
	// Initialize client (refresh tokens, etc.)
	ctx := context.Background()
	if err := client.Initialize(ctx); err != nil {
		log.Printf("Failed to initialize client: %v", err)
		return
	}
	
	// Get the event poller
	poller := client.GetEventPoller()
	
	// Subscribe to event notifications
	eventCh := poller.SubscribeToEvents()
	
	// Start polling with 3-second interval
	if err := poller.StartEventPolling(3 * time.Second); err != nil {
		log.Printf("Failed to start event polling: %v", err)
		return
	}
	
	// Listen for events in a separate goroutine
	go func() {
		for eventNumber := range eventCh {
			fmt.Printf("New event number received: %d\n", eventNumber)
		}
	}()
	
	// Wait for a few events (in a real application, you'd do other work)
	time.Sleep(15 * time.Second)
	
	// Stop polling
	poller.StopEventPolling()
	
	fmt.Printf("Final event number: %d\n", poller.GetCurrentEventNumber())
}

// ExampleEventPoller_waitForEvent demonstrates waiting for a specific event
func ExampleEventPoller_waitForEvent() {
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	poller := client.GetEventPoller()
	
	// Start polling
	if err := poller.StartEventPolling(2 * time.Second); err != nil {
		log.Printf("Failed to start event polling: %v", err)
		return
	}
	defer poller.StopEventPolling()
	
	// Wait for the next event with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	eventNumber, err := poller.WaitForEvent(ctx)
	if err != nil {
		if err == context.DeadlineExceeded {
			fmt.Println("No new events within timeout period")
		} else {
			log.Printf("Error waiting for event: %v", err)
		}
		return
	}
	
	fmt.Printf("Received event number: %d\n", eventNumber)
}

// ExampleEventPoller_multipleSubscribers demonstrates multiple event subscribers
func ExampleEventPoller_multipleSubscribers() {
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	poller := client.GetEventPoller()
	
	// Create multiple subscribers
	subscriber1 := poller.SubscribeToEvents()
	subscriber2 := poller.SubscribeToEvents()
	
	// Start polling
	if err := poller.StartEventPolling(1 * time.Second); err != nil {
		log.Printf("Failed to start event polling: %v", err)
		return
	}
	defer poller.StopEventPolling()
	
	// Handle events from multiple subscribers
	go func() {
		for eventNumber := range subscriber1 {
			fmt.Printf("Subscriber 1 received event: %d\n", eventNumber)
		}
	}()
	
	go func() {
		for eventNumber := range subscriber2 {
			fmt.Printf("Subscriber 2 received event: %d\n", eventNumber)
		}
	}()
	
	// Wait for some events
	time.Sleep(5 * time.Second)
	
	fmt.Printf("Total subscribers: %d\n", poller.GetSubscriberCount())
}

// ExampleEventPoller_pollingStatus demonstrates checking polling status
func ExampleEventPoller_pollingStatus() {
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	poller := client.GetEventPoller()
	
	fmt.Printf("Is running initially: %t\n", poller.IsRunning())
	
	// Start polling
	if err := poller.StartEventPolling(5 * time.Second); err != nil {
		log.Printf("Failed to start event polling: %v", err)
		return
	}
	
	fmt.Printf("Is running after start: %t\n", poller.IsRunning())
	fmt.Printf("Poll interval: %v\n", poller.GetPollInterval())
	
	// Update poll interval
	poller.SetPollInterval(2 * time.Second)
	fmt.Printf("Updated poll interval: %v\n", poller.GetPollInterval())
	
	// Stop polling
	poller.StopEventPolling()
	fmt.Printf("Is running after stop: %t\n", poller.IsRunning())
}
