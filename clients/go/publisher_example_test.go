package yesterdaygo

import (
	"fmt"
	"log"
	"time"
)

// ExampleEventPublisher_basic demonstrates basic event publishing
func ExampleEventPublisher_basic() {
	// Create a client
	client := NewClient("https://api.yesterday.localhost")
	
	// Get the event publisher (automatically created with client)
	publisher := client.GetEventPublisher()
	
	// Publish an event
	err := publisher.PublishEvent("user.created", map[string]interface{}{
		"userId": "user123",
		"email":  "user@example.com",
	})
	if err != nil {
		log.Fatalf("Failed to publish event: %v", err)
	}
	
	fmt.Println("Event published successfully")
	// Output: Event published successfully
}

// ExampleEventPublisher_withOptions demonstrates creating a publisher with custom options
func ExampleEventPublisher_withOptions() {
	client := NewClient("https://api.yesterday.localhost")
	
	// Create publisher with custom configuration
	publisher := NewEventPublisher(client,
		WithRetryBackoff(2*time.Second),
		WithMaxRetries(5),
		WithBatchSize(10),
	)
	
	// Publish multiple events
	events := []struct {
		eventType string
		payload   interface{}
	}{
		{"user.created", map[string]string{"userId": "user1"}},
		{"user.updated", map[string]string{"userId": "user1", "field": "email"}},
		{"user.deleted", map[string]string{"userId": "user1"}},
	}
	
	for _, event := range events {
		err := publisher.PublishEvent(event.eventType, event.payload)
		if err != nil {
			log.Printf("Failed to publish event %s: %v", event.eventType, err)
			continue
		}
		fmt.Printf("Queued event: %s\n", event.eventType)
	}
	
	// Stop the publisher when done
	publisher.Stop()
	// Output: 
	// Queued event: user.created
	// Queued event: user.updated
	// Queued event: user.deleted
}

// ExampleEventPublisher_flushEvents demonstrates waiting for all events to be published
func ExampleEventPublisher_flushEvents() {
	client := NewClient("https://api.yesterday.localhost")
	publisher := client.GetEventPublisher()
	
	// Publish some events
	for i := 0; i < 5; i++ {
		err := publisher.PublishEvent("batch.event", map[string]interface{}{
			"index": i,
			"batch": "example",
		})
		if err != nil {
			log.Printf("Failed to queue event %d: %v", i, err)
		}
	}
	
	fmt.Printf("Queued %d events\n", publisher.GetQueueLength())
	
	// Wait for all events to be published (with timeout)
	err := publisher.FlushEvents(30 * time.Second)
	if err != nil {
		log.Printf("Failed to flush all events: %v", err)
	} else {
		fmt.Println("All events published successfully")
	}
	
	fmt.Printf("Remaining events in queue: %d\n", publisher.GetQueueLength())
	// Output:
	// Queued 5 events
	// All events published successfully
	// Remaining events in queue: 0
}

// ExampleEventPublisher_monitoring demonstrates monitoring publisher status
func ExampleEventPublisher_monitoring() {
	client := NewClient("https://api.yesterday.localhost")
	publisher := client.GetEventPublisher()
	
	// Check if publisher is running
	fmt.Printf("Publisher running: %t\n", publisher.IsRunning())
	
	// Publish an event
	err := publisher.PublishEvent("system.started", map[string]string{
		"service": "example",
		"version": "1.0.0",
	})
	if err != nil {
		log.Printf("Failed to publish event: %v", err)
	}
	
	// Check queue length
	fmt.Printf("Events in queue: %d\n", publisher.GetQueueLength())
	
	// Stop the publisher
	publisher.Stop()
	fmt.Printf("Publisher running after stop: %t\n", publisher.IsRunning())
	
	// Output:
	// Publisher running: true
	// Events in queue: 1
	// Publisher running after stop: false
}

// ExampleEventPublisher_errorHandling demonstrates error handling patterns
func ExampleEventPublisher_errorHandling() {
	client := NewClient("https://api.yesterday.localhost")
	
	// Create publisher with shorter retry settings for demo
	publisher := NewEventPublisher(client,
		WithRetryBackoff(100*time.Millisecond),
		WithMaxRetries(3),
	)
	
	// Publish events with different payload types
	events := []struct {
		name      string
		eventType string
		payload   interface{}
	}{
		{"valid event", "user.action", map[string]string{"action": "login"}},
		{"complex payload", "data.update", struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Active bool   `json:"active"`
		}{ID: 123, Name: "test", Active: true}},
		{"string payload", "message.sent", "Hello, world!"},
	}
	
	for _, event := range events {
		err := publisher.PublishEvent(event.eventType, event.payload)
		if err != nil {
			log.Printf("Failed to queue %s: %v", event.name, err)
		} else {
			fmt.Printf("Queued %s: %s\n", event.name, event.eventType)
		}
	}
	
	// Clean shutdown
	publisher.Stop()
	
	// Output:
	// Queued valid event: user.action
	// Queued complex payload: data.update
	// Queued string payload: message.sent
}

// ExampleEventPublisher_gracefulShutdown demonstrates proper shutdown procedures
func ExampleEventPublisher_gracefulShutdown() {
	client := NewClient("https://api.yesterday.localhost")
	publisher := client.GetEventPublisher()
	
	// Publish some events
	for i := 0; i < 3; i++ {
		publisher.PublishEvent("shutdown.test", map[string]int{"sequence": i})
	}
	
	fmt.Printf("Published events, queue length: %d\n", publisher.GetQueueLength())
	
	// Attempt to flush events before shutdown
	flushTimeout := 5 * time.Second
	fmt.Printf("Flushing events with %v timeout...\n", flushTimeout)
	
	err := publisher.FlushEvents(flushTimeout)
	if err != nil {
		fmt.Printf("Flush completed with warning: %v\n", err)
	} else {
		fmt.Println("All events flushed successfully")
	}
	
	// Final stop
	publisher.Stop()
	fmt.Println("Publisher stopped")
	
	// Output:
	// Published events, queue length: 3
	// Flushing events with 5s timeout...
	// All events flushed successfully
	// Publisher stopped
}
