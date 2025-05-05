package users

// Package users provides a simple user management service.
// Users are added to the database by the admin with a username and password.
// The users service allows users to login to one of multiple applications,
// which creates a long-lived refresh token and a short-lived session token
// containing application-specific profile information.
//
// This service is used with an authentication middleware to protect resources
// that require authentication.
//
// The server is initialized in cmd/serve/main.go and the state is managed in state/users.go.
