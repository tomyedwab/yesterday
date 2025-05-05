package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/users/auth"
	"github.com/tomyedwab/yesterday/users/sessions"
	"github.com/tomyedwab/yesterday/users/state"
)

type ChangePasswordRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func RegisterAuthHandlers(db *database.Database, sessionManager *sessions.SessionManager) {
	// Endpoints used during authentication flow

	LoginHandler := func(w http.ResponseWriter, r *http.Request) {
		var loginRequest auth.LoginRequest
		err := json.NewDecoder(r.Body).Decode(&loginRequest)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		refreshToken, err := auth.DoLogin(db, sessionManager, loginRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Set the refresh token in a cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "YRT",
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		w.WriteHeader(http.StatusOK)
	}

	RefreshHandler := func(w http.ResponseWriter, r *http.Request) {
		var refreshRequest auth.RefreshRequest
		err := json.NewDecoder(r.Body).Decode(&refreshRequest)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		response, err := auth.DoRefresh(db, sessionManager, refreshRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
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
		// Get refresh token from cookie
		refreshToken, err := r.Cookie("YRT")
		if err != nil {
			http.Error(w, "No refresh token found", http.StatusUnauthorized)
			return
		}

		err = auth.DoLogout(db, sessionManager, refreshToken.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Clear the YRT cookie
		http.SetCookie(w, &http.Cookie{
			Name:   "YRT",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		w.WriteHeader(http.StatusOK)
	}

	http.HandleFunc("/api/login", LoginHandler)
	http.HandleFunc("/api/refresh", RefreshHandler)
	http.HandleFunc("/api/logout", LogoutHandler)
}

func RegisterAdminHandlers(db *database.Database, sessionManager *sessions.SessionManager) {
	// Endpoint used by an admin to change user password

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

	http.HandleFunc("/api/changepw", ChangePasswordHandler)
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

	// Register auth endpoints
	RegisterAuthHandlers(db, sessionManager)

	// Register admin endpoints
	RegisterAdminHandlers(db, sessionManager)

	// Serve static files from the `www` directory
	fs := http.FileServer(http.Dir("./www"))
	http.Handle("/", fs)

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
