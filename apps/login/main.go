package main

import (
	"log"
	"net/http"
	"time"

	"github.com/tomyedwab/yesterday/applib"
	"github.com/tomyedwab/yesterday/apps/login/handlers"
	"github.com/tomyedwab/yesterday/apps/login/sessions"
)

func main() {
	application, err := applib.Init("0.0.1")
	if err != nil {
		log.Fatal(err)
	}

	sessionManager, err := sessions.NewManager(application.GetSqliteDB(), 10*time.Minute, 1*time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	application.AddContextVar(sessions.SessionManagerKey, sessionManager)

	http.HandleFunc("/public/login", handlers.HandleLogin)
	http.HandleFunc("/public/logout", handlers.HandleLogout)
	http.HandleFunc("/internal/access_token", handlers.HandleAccessToken)

	err = application.GetDatabase().Initialize()
	if err != nil {
		panic(err)
	}

	application.Serve()
}
