package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/applib"
	"github.com/tomyedwab/yesterday/applib/database"
	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/apps/admin/handlers"
	"github.com/tomyedwab/yesterday/apps/admin/state"
)

func main() {
	application, err := applib.Init("0.0.1")
	if err != nil {
		log.Fatal(err)
	}

	// Internal login functionality, called by nexushub directly
	http.HandleFunc("/internal/dologin", handlers.HandleDoLogin)
	http.HandleFunc("/internal/checkAccess", handlers.HandleCheckAccess)

	// Register data views
	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		db := r.Context().Value(applib.ContextSqliteDatabaseKey).(*sqlx.DB)
		ret, err := state.GetUsers(db)
		httputils.HandleAPIResponse(w, r, map[string]any{
			"users": ret,
		}, err, http.StatusInternalServerError)
	})

	// Special method to hash a password for the client
	http.HandleFunc("/api/hash_password", func(w http.ResponseWriter, r *http.Request) {
		passwordBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read password: %v", err), http.StatusInternalServerError)
			return
		}
		var password string
		err = json.Unmarshal(passwordBytes, &password)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to unmarshal password: %v", err), http.StatusInternalServerError)
			return
		}

		// Create random salt
		salt := uuid.New().String()

		var passwordHash string
		// Hash the provided password
		hasher := sha256.New()
		hasher.Write([]byte(salt + password))
		passwordHash = hex.EncodeToString(hasher.Sum(nil))

		httputils.HandleAPIResponse(w, r, map[string]any{
			"salt":         salt,
			"passwordHash": passwordHash,
		}, nil, http.StatusOK)
	})

	db := application.GetDatabase()

	tx := db.GetDB().MustBegin()
	err = state.InitUsers(tx)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()

	// User management event handlers
	database.AddEventHandler(db, state.UserAddedEventType, state.UsersHandleAddedEvent)
	database.AddEventHandler(db, state.UpdateUserPasswordEventType, state.UsersHandleUpdatePasswordEvent)
	database.AddEventHandler(db, state.DeleteUserEventType, state.UsersHandleDeleteEvent)
	database.AddEventHandler(db, state.UpdateUserEventType, state.UsersHandleUpdateEvent)

	err = db.Initialize()
	if err != nil {
		panic(err)
	}

	application.Serve()
}
