package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/apps/admin/login"
	"github.com/tomyedwab/yesterday/apps/admin/state"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/wasi/guest"
	"github.com/tomyedwab/yesterday/wasi/types"
)

//go:wasmexport init
func init() {
	guest.Init("0.0.1")

	db, err := sqlx.Connect("sqlproxy", "")
	if err != nil {
		panic("cannot connect to sqlproxy")
	}

	// Internal login functionality, used by the login service
	guest.RegisterHandler("/internal/dologin", login.HandleDoLogin)
	guest.RegisterHandler("/internal/checkAccess", login.HandleCheckAccess)

	// Register event handlers
	guest.RegisterEventHandler(events.DBInitEventType, state.ApplicationsHandleInitEvent)
	guest.RegisterEventHandler(events.DBInitEventType, state.UsersHandleInitEvent)
	guest.RegisterEventHandler(state.UserAddedEventType, state.UsersHandleAddedEvent)
	guest.RegisterEventHandler(events.DBInitEventType, state.UserAccessRulesHandleInitEvent)
	
	// User management event handlers
	guest.RegisterEventHandler(state.UpdateUserPasswordEventType, state.UsersHandleUpdatePasswordEvent)
	guest.RegisterEventHandler(state.DeleteUserEventType, state.UsersHandleDeleteEvent)
	guest.RegisterEventHandler(state.UpdateUserEventType, state.UsersHandleUpdateEvent)
	
	// Application management event handlers
	guest.RegisterEventHandler(state.CreateApplicationEventType, state.ApplicationsHandleCreateEvent)
	guest.RegisterEventHandler(state.UpdateApplicationEventType, state.ApplicationsHandleUpdateEvent)
	guest.RegisterEventHandler(state.DeleteApplicationEventType, state.ApplicationsHandleDeleteEvent)
	
	// User access rules management event handlers
	guest.RegisterEventHandler(state.CreateUserAccessRuleEventType, state.UserAccessRulesHandleCreateEvent)
	guest.RegisterEventHandler(state.DeleteUserAccessRuleEventType, state.UserAccessRulesHandleDeleteEvent)

	// Register data views
	guest.RegisterHandler("/api/users", func(params types.RequestParams) types.Response {
		ret, err := state.GetUsers(db)
		return guest.CreateResponse(map[string]any{
			"users": ret,
		}, err, "Error fetching users")
	})
	
	guest.RegisterHandler("/api/applications", func(params types.RequestParams) types.Response {
		ret, err := state.GetApplications(db)
		return guest.CreateResponse(map[string]any{
			"applications": ret,
		}, err, "Error fetching applications")
	})
	
	guest.RegisterHandler("/api/user-access-rules", func(params types.RequestParams) types.Response {
		applicationId := params.Query().Get("applicationId")
		var ret []state.UserAccessRule
		var err error
		
		if applicationId != "" {
			ret, err = state.GetUserAccessRulesForApplication(db, applicationId)
		} else {
			ret, err = state.GetAllUserAccessRules(db)
		}
		
		return guest.CreateResponse(map[string]any{
			"rules": ret,
		}, err, "Error fetching user access rules")
	})
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
