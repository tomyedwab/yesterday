package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/database/middleware"
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

	LoginHandler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			loginApplication, err := state.GetApplication(db.GetDB(), "0001-0001")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			app := r.URL.Query().Get("app")
			if app == "" {
				http.Error(w, "Application name is required", http.StatusBadRequest)
				return
			}
			application, err := state.GetApplication(db.GetDB(), app)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var loginRequest auth.LoginRequest
			err = json.NewDecoder(r.Body).Decode(&loginRequest)
			if err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			refreshToken, err := auth.DoLogin(db, sessionManager, loginRequest, app)
			if err != nil {
				fmt.Printf("DoLogin failed: %v\n", err)
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			responseJson, err := json.Marshal(map[string]string{
				"domain": application.HostName,
			})
			if err != nil {
				http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
				return
			}

			// Set the refresh token in a cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "YRT",
				Value:    refreshToken,
				Domain:   loginApplication.HostName,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			})

			w.WriteHeader(http.StatusOK)
			w.Write(responseJson)
		},
		middleware.LogRequests,
	)

	RefreshHandler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			app := r.URL.Query().Get("app")
			if app != "" {
				// If a browser makes a CORS preflight request, the client needs
				// to specify the app name so we can provide the correct domain
				application, err := state.GetApplication(db.GetDB(), app)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.Header().Set("Access-Control-Allow-Origin", "https://"+application.HostName)
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				// Do not call through to the handler itself, just return immediately
				return
			}

			loginApplication, err := state.GetApplication(db.GetDB(), "0001-0001")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Get refresh token from cookie
			refreshToken, err := r.Cookie("YRT")
			if err != nil {
				http.Error(w, "No refresh token found", http.StatusUnauthorized)
				return
			}

			response, newRefreshToken, err := auth.DoRefresh(db, sessionManager, refreshToken.Value)
			if err != nil {
				fmt.Printf("DoRefresh failed: %v\n", err)
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			responseJson, err := json.Marshal(response)
			if err != nil {
				http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
				return
			}

			// Set the refresh token in a cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "YRT",
				Value:    newRefreshToken,
				Domain:   loginApplication.HostName,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			})
			w.WriteHeader(http.StatusOK)
			w.Write(responseJson)
		},
		middleware.LogRequests,
	)

	LogoutHandler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			loginApplication, err := state.GetApplication(db.GetDB(), "0001-0001")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

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
				Name:     "YRT",
				Value:    "",
				Domain:   loginApplication.HostName,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
				MaxAge:   -1,
			})
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Logged out"))
		},
		middleware.LogRequests,
	)

	http.HandleFunc("/api/login", LoginHandler)
	http.HandleFunc("/api/refresh", RefreshHandler)
	http.HandleFunc("/api/logout", LogoutHandler)
}

func RegisterAdminHandlers(db *database.Database, sessionManager *sessions.SessionManager) {
	// Endpoint used by an admin to change user password

	ChangePasswordHandler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			var changePasswordRequest ChangePasswordRequest
			err := json.NewDecoder(r.Body).Decode(&changePasswordRequest)
			if err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			userID, err := state.ChangePassword(db, changePasswordRequest.Username, changePasswordRequest.Password)
			if err != nil {
				http.Error(w, "Failed to change password", http.StatusInternalServerError)
				return
			}

			sessionManager.DeleteSessionsForUser(userID)

			w.WriteHeader(http.StatusOK)
		},
		middleware.LoginRequired,
		middleware.EnableCrossOrigin,
		middleware.LogRequests,
	)

	http.HandleFunc("/api/changepw", ChangePasswordHandler)
}

func main() {
	dbPath := "./users.db"

	// Define handlers for user state
	handlers := map[string]database.EventUpdateHandler{
		"users_v1":         state.UserStateHandler,
		"user_profiles_v1": state.UserProfileStateHandler,
		"applications_v1":  state.ApplicationStateHandler,
	}

	// Connect to the database and initialize schema/handlers
	// Use a version string for your user service schema/handlers
	db, err := database.Connect("sqlite3", dbPath, "v1.0.0", handlers)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sessionManager, err := sessions.NewManager(db, 5*time.Minute, 30*24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	// Initialize HTTP endpoints using our event mapper
	db.InitHandlers(state.MapUserEventType)

	state.InitUserHandlers(db)
	state.InitUserProfileHandlers(db)
	state.InitApplicationHandlers(db)

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
