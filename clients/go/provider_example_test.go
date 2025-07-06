package yesterdaygo_test

import (
	"context"
	"fmt"
	"log"
	"time"

	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

// User represents a user data structure
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// UserList represents a list of users
type UserList struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}

// ExampleDataProvider demonstrates basic data provider usage
func ExampleDataProvider() {
	// Create client and initialize
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	ctx := context.Background()
	if err := client.Initialize(ctx); err != nil {
		log.Printf("Warning: %v", err)
	}
	
	// Create a data provider for user data
	userProvider := yesterdaygo.NewDataProvider[User](client, "/api/users/123", nil)
	
	// Get user data (will fetch from API on first call)
	user, err := userProvider.Get()
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}
	
	fmt.Printf("User: %s (%s)\n", user.Username, user.Email)
	
	// Get user data again (will return cached data if no events)
	user2, err := userProvider.Get()
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}
	
	fmt.Printf("Cached user: %s (%s)\n", user2.Username, user2.Email)
	
	// Clean up
	userProvider.Close()
}

// ExampleDataProvider_withParameters demonstrates using query parameters
func ExampleDataProvider_withParameters() {
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	
	// Create provider with query parameters
	params := map[string]interface{}{
		"page":     1,
		"per_page": 10,
		"active":   true,
	}
	
	usersProvider := yesterdaygo.NewDataProvider[UserList](client, "/api/users", params)
	
	// Get users list
	userList, err := usersProvider.Get()
	if err != nil {
		log.Printf("Failed to get users: %v", err)
		return
	}
	
	fmt.Printf("Retrieved %d users (total: %d)\n", len(userList.Users), userList.Total)
	
	// Update parameters and get new data
	newParams := map[string]interface{}{
		"page":     2,
		"per_page": 20,
		"active":   true,
	}
	
	if err := usersProvider.SetParams(newParams); err != nil {
		log.Printf("Failed to update params: %v", err)
		return
	}
	
	// Get updated data
	userList2, err := usersProvider.Get()
	if err != nil {
		log.Printf("Failed to get updated users: %v", err)
		return
	}
	
	fmt.Printf("Updated: Retrieved %d users (total: %d)\n", len(userList2.Users), userList2.Total)
	
	usersProvider.Close()
}

// ExampleDataProvider_subscription demonstrates automatic refresh on events
func ExampleDataProvider_subscription() {
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	
	// Start event polling
	poller := client.GetEventPoller()
	if err := poller.StartEventPolling(2 * time.Second); err != nil {
		log.Printf("Failed to start event polling: %v", err)
		return
	}
	defer poller.StopEventPolling()
	
	// Create data provider
	userProvider := yesterdaygo.NewDataProvider[User](client, "/api/users/123", nil)
	defer userProvider.Close()
	
	// Subscribe to automatic updates
	err := userProvider.Subscribe(func(user User) {
		fmt.Printf("User data updated: %s (%s)\n", user.Username, user.Email)
	})
	if err != nil {
		log.Printf("Failed to subscribe: %v", err)
		return
	}
	
	// Get initial data
	user, err := userProvider.Get()
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}
	
	fmt.Printf("Initial user: %s (%s)\n", user.Username, user.Email)
	
	// Wait for automatic updates (in a real app, you'd do other work)
	time.Sleep(10 * time.Second)
	
	fmt.Printf("Last event number: %d\n", userProvider.GetLastEventNumber())
}

// ExampleDataProvider_manualRefresh demonstrates manual data refresh
func ExampleDataProvider_manualRefresh() {
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	userProvider := yesterdaygo.NewDataProvider[User](client, "/api/users/123", nil)
	defer userProvider.Close()
	
	// Get initial data
	user, err := userProvider.Get()
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}
	
	fmt.Printf("Initial: %s (event: %d)\n", user.Username, userProvider.GetLastEventNumber())
	
	// Wait a bit, then manually refresh
	time.Sleep(2 * time.Second)
	
	if err := userProvider.Refresh(); err != nil {
		log.Printf("Failed to refresh: %v", err)
		return
	}
	
	// Get refreshed data
	user2, err := userProvider.Get()
	if err != nil {
		log.Printf("Failed to get refreshed user: %v", err)
		return
	}
	
	fmt.Printf("Refreshed: %s (event: %d)\n", user2.Username, userProvider.GetLastEventNumber())
}

// ExampleDataProvider_multipleProviders demonstrates multiple data providers
func ExampleDataProvider_multipleProviders() {
	client := yesterdaygo.NewClient("https://api.yesterday.localhost")
	
	// Start event polling for automatic updates
	poller := client.GetEventPoller()
	if err := poller.StartEventPolling(3 * time.Second); err != nil {
		log.Printf("Failed to start polling: %v", err)
		return
	}
	defer poller.StopEventPolling()
	
	// Create multiple data providers
	userProvider := yesterdaygo.NewDataProvider[User](client, "/api/users/123", nil)
	usersProvider := yesterdaygo.NewDataProvider[UserList](client, "/api/users", map[string]interface{}{
		"active": true,
	})
	defer userProvider.Close()
	defer usersProvider.Close()
	
	// Subscribe both to automatic updates
	userProvider.Subscribe(func(user User) {
		fmt.Printf("Single user updated: %s\n", user.Username)
	})
	
	usersProvider.Subscribe(func(users UserList) {
		fmt.Printf("Users list updated: %d users\n", len(users.Users))
	})
	
	// Get initial data from both
	user, err := userProvider.Get()
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}
	
	usersList, err := usersProvider.Get()
	if err != nil {
		log.Printf("Failed to get users list: %v", err)
		return
	}
	
	fmt.Printf("Single user: %s\n", user.Username)
	fmt.Printf("Users list: %d users\n", len(usersList.Users))
	fmt.Printf("User subscribed: %t\n", userProvider.IsSubscribed())
	fmt.Printf("Users list subscribed: %t\n", usersProvider.IsSubscribed())
	
	// Wait for updates
	time.Sleep(10 * time.Second)
}
