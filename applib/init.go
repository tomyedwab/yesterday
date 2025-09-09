package applib

import (
	"fmt"

	"github.com/tomyedwab/yesterday/applib/database"
)

func Init() (*Application, error) {
	db, err := database.Connect("sqlite3", "/db/app.sqlite")
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database: %v", err)
	}

	return NewApplication(db), nil
}
