package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/users/sessions"
	"github.com/tomyedwab/yesterday/users/state"
)

type LoginRequest struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	ApplicationName string `json:"application"`
}

type LoginResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type ChangePasswordRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	SessionID       string `json:"session_id"`
	RefreshToken    string `json:"refresh_token"`
	ApplicationName string `json:"application"`
}

type RefreshResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type LogoutRequest struct {
	SessionID    string `json:"session_id"`
	RefreshToken string `json:"refresh_token"`
}

func RegisterLoginHandlers(db *database.Database, sessionManager *sessions.SessionManager) {
	// Endpoints used during authentication flow

	LoginHandler := func(w http.ResponseWriter, r *http.Request) {
		var loginRequest LoginRequest
		err := json.NewDecoder(r.Body).Decode(&loginRequest)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if loginRequest.ApplicationName == "" {
			http.Error(w, "Application name is required", http.StatusBadRequest)
			return
		}

		success, userId, err := state.AttemptLogin(db, loginRequest.Username, loginRequest.Password)
		if err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		if !success {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		profile, err := state.GetUserProfile(db.GetDB(), userId, loginRequest.ApplicationName)
		if err != nil {
			http.Error(w, "Failed to get user profile", http.StatusInternalServerError)
			return
		}

		session, err := sessionManager.CreateSession(userId)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		accessToken, refreshToken, err := sessionManager.RefreshAccessToken(session, "", loginRequest.ApplicationName, profile)
		if err != nil {
			http.Error(w, "Failed to refresh access token", http.StatusInternalServerError)
			return
		}

		response := LoginResponse{
			RefreshToken: refreshToken,
			AccessToken:  accessToken,
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseJson)
	}

	RefreshHandler := func(w http.ResponseWriter, r *http.Request) {
		var refreshRequest RefreshRequest
		err := json.NewDecoder(r.Body).Decode(&refreshRequest)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		session, err := sessionManager.GetSession(refreshRequest.SessionID)
		if err != nil {
			http.Error(w, "Failed to get session", http.StatusInternalServerError)
			return
		}

		profile, err := state.GetUserProfile(db.GetDB(), session.UserID, refreshRequest.ApplicationName)
		if err != nil {
			http.Error(w, "Failed to get user profile", http.StatusInternalServerError)
			return
		}

		accessToken, refreshToken, err := sessionManager.RefreshAccessToken(session, refreshRequest.RefreshToken, refreshRequest.ApplicationName, profile)
		if err != nil {
			http.Error(w, "Failed to refresh access token", http.StatusInternalServerError)
			return
		}

		response := RefreshResponse{
			RefreshToken: refreshToken,
			AccessToken:  accessToken,
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseJson)
	}

	LogoutHandler := func(w http.ResponseWriter, r *http.Request) {
		var logoutRequest LogoutRequest
		err := json.NewDecoder(r.Body).Decode(&logoutRequest)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		session, err := sessionManager.GetSession(logoutRequest.SessionID)
		if err != nil {
			http.Error(w, "Failed to get session", http.StatusInternalServerError)
			return
		}

		if session.RefreshToken != logoutRequest.RefreshToken {
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}

		err = session.DBDelete(db.GetDB())
		if err != nil {
			http.Error(w, "Failed to delete session", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}

	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/refresh", RefreshHandler)
	http.HandleFunc("/logout", LogoutHandler)

	// Endpoints used by an admin to change user password

	ChangePasswordHandler := func(w http.ResponseWriter, r *http.Request) {
		// TODO STOPSHIP: User has to be authenticated and an admin to do this
		var changePasswordRequest ChangePasswordRequest
		err := json.NewDecoder(r.Body).Decode(&changePasswordRequest)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		err = state.ChangePassword(db, changePasswordRequest.Username, changePasswordRequest.Password)
		if err != nil {
			http.Error(w, "Failed to change password", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}

	http.HandleFunc("/changepw", ChangePasswordHandler)
}

func main() {
	dbPath := "./users.db"

	// Define handlers for user state
	handlers := map[string]database.EventUpdateHandler{
		"users_v1":         state.UserStateHandler,
		"user_profiles_v1": state.UserProfileStateHandler,
	}

	// Connect to the database and initialize schema/handlers
	// Use a version string for your user service schema/handlers
	db, err := database.Connect("sqlite3", dbPath, "v1.0.0", handlers)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sessionManager, err := sessions.NewManager(db, 5*time.Minute, 30*24*time.Hour, "./jwt_secret.key")
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	// Initialize HTTP endpoints using our event mapper
	db.InitHandlers(state.MapUserEventType)

	// Register login endpoints
	RegisterLoginHandlers(db, sessionManager)

	// Once every 10 minutes, delete expired sessions
	go func() {
		for {
			sessionManager.DeleteExpiredSessions()
			time.Sleep(10 * time.Minute)
		}
	}()

	port := "8081"
	fmt.Printf("Database initialized. Starting user service server on :%s\n", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
