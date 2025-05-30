package main

import (
	"fmt"

	"database/sql"

	"github.com/tomyedwab/yesterday/wasi"
)

func handle_echo(params wasi.RequestParams) (string, error) {
	db, err := sql.Open("sqlproxy", "")
	if err != nil {
		return "", fmt.Errorf("sql.Open failed: %v", err)
	}
	defer db.Close()

	// Use db as a standard *sql.DB object
	rows, err := db.Query("SELECT id, username FROM users_v1")
	if err != nil {
		return "", fmt.Errorf("sql.Query failed: %v", err)
	}
	defer rows.Close()

	var ret string
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return "", fmt.Errorf("sql.Scan failed: %v", err)
		}
		ret += fmt.Sprintf("User %d: %s\n", id, name)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("sql.Rows.Err failed: %v", err)
	}
	return ret, nil
}

//go:wasmexport init
func init() {
	wasi.Init()
	wasi.RegisterHandler("/api/echo", handle_echo)
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
