package applib

import (
	"flag"
	"fmt"

	"github.com/tomyedwab/yesterday/applib/database"
)

func Init(serverVersion string) (*Application, error) {
	// TODO(tom): More flexible configuration sharing from nexushub
	//
	dbPath := flag.String("dbPath", "", "Path to the SQLite database file")
	port := flag.Int("port", 8080, "Port for the HTTP server")
	flag.Parse()

	if *dbPath == "" {
		return nil, fmt.Errorf("Database path must be provided via -dbPath flag")
	}

	db, err := database.Connect("sqlite3", *dbPath, serverVersion)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database: %v", err)
	}

	return NewApplication(serverVersion, *port, db), nil
}
