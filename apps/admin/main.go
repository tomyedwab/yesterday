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

	// Register data views
	guest.RegisterHandler("/api/users", func(params types.RequestParams) types.Response {
		ret, err := state.GetUsers(db)
		return guest.CreateResponse(map[string]any{
			"users": ret,
		}, err, "Error fetching users")
	})
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
