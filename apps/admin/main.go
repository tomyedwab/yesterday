package main

import (
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/applib"
	"github.com/tomyedwab/yesterday/applib/database"
	"github.com/tomyedwab/yesterday/applib/events"
	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/apps/admin/handlers"
	"github.com/tomyedwab/yesterday/apps/admin/state"
)

func main() {
	application, err := applib.Init("0.0.1")
	if err != nil {
		log.Fatal(err)
	}

	// Internal login functionality, used by the login service
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

	http.HandleFunc("/api/applications", func(w http.ResponseWriter, r *http.Request) {
		db := r.Context().Value(applib.ContextSqliteDatabaseKey).(*sqlx.DB)
		ret, err := state.GetApplications(db)
		httputils.HandleAPIResponse(w, r, map[string]any{
			"applications": ret,
		}, err, http.StatusInternalServerError)
	})

	http.HandleFunc("/api/user-access-rules", func(w http.ResponseWriter, r *http.Request) {
		db := r.Context().Value(applib.ContextSqliteDatabaseKey).(*sqlx.DB)
		applicationId := r.URL.Query().Get("applicationId")
		var ret []state.UserAccessRule
		var err error

		if applicationId != "" {
			ret, err = state.GetUserAccessRulesForApplication(db, applicationId)
		} else {
			ret, err = state.GetAllUserAccessRules(db)
		}

		httputils.HandleAPIResponse(w, r, map[string]any{
			"rules": ret,
		}, err, http.StatusInternalServerError)
	})

	db := application.GetDatabase()

	// Register event handlers
	database.AddEventHandler(db, events.DBInitEventType, state.ApplicationsHandleInitEvent)
	database.AddEventHandler(db, events.DBInitEventType, state.UsersHandleInitEvent)
	database.AddEventHandler(db, events.DBInitEventType, state.UserAccessRulesHandleInitEvent)

	// User management event handlers
	database.AddEventHandler(db, state.UserAddedEventType, state.UsersHandleAddedEvent)
	database.AddEventHandler(db, state.UpdateUserPasswordEventType, state.UsersHandleUpdatePasswordEvent)
	database.AddEventHandler(db, state.DeleteUserEventType, state.UsersHandleDeleteEvent)
	database.AddEventHandler(db, state.UpdateUserEventType, state.UsersHandleUpdateEvent)

	// Application management event handlers
	database.AddEventHandler(db, state.CreateApplicationEventType, state.ApplicationsHandleCreateEvent)
	database.AddEventHandler(db, state.CreateDebugApplicationEventType, state.ApplicationsHandleCreateDebugApplicationEvent)
	database.AddEventHandler(db, state.UpdateApplicationEventType, state.ApplicationsHandleUpdateEvent)
	database.AddEventHandler(db, state.DeleteApplicationEventType, state.ApplicationsHandleDeleteEvent)

	// User access rules management event handlers
	database.AddEventHandler(db, state.CreateUserAccessRuleEventType, state.UserAccessRulesHandleCreateEvent)
	database.AddEventHandler(db, state.DeleteUserAccessRuleEventType, state.UserAccessRulesHandleDeleteEvent)

	err = db.Initialize()
	if err != nil {
		panic(err)
	}

	application.Serve()
}
