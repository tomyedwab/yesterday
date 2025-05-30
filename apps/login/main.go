package main

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/tomyedwab/yesterday/apps/admin/state"
	"github.com/tomyedwab/yesterday/wasi"
)

func handle_appinfo(params wasi.RequestParams) (string, error) {
	app := params.Query().Get("app")
	if app == "" {
		return "", fmt.Errorf("Application name is required")
	}

	db, err := sqlx.Connect("sqlproxy", "")
	if err != nil {
		return "", fmt.Errorf("sqlx.Connect failed: %v", err)
	}
	defer db.Close()

	application, err := state.GetApplication(db, app)
	if err != nil {
		return "", fmt.Errorf("state.GetApplication failed: %v", err)
	}

	return application.HostName, nil
}

//go:wasmexport init
func init() {
	wasi.Init()
	wasi.RegisterHandler("/api/appinfo", handle_appinfo)
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
