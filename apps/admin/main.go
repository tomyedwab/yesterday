package main

import (
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/applib"
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

	db := application.GetDatabase()

	tx := db.GetDB().MustBegin()
	err = state.InitUsers(tx)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()

	// User management event handlers
	//database.AddEventHandler(db, state.UserAddedEventType, state.UsersHandleAddedEvent)
	//database.AddEventHandler(db, state.UpdateUserPasswordEventType, state.UsersHandleUpdatePasswordEvent)
	//database.AddEventHandler(db, state.DeleteUserEventType, state.UsersHandleDeleteEvent)
	//database.AddEventHandler(db, state.UpdateUserEventType, state.UsersHandleUpdateEvent)

	err = db.Initialize()
	if err != nil {
		panic(err)
	}

	application.Serve()
}
