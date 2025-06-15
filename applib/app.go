package applib

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/applib/database"
)

type Application struct {
	serverVersion string
	serverPort    int
	db            *database.Database
	contextVars   map[string]any
}

var (
	ContextApplicationKey    = "application"
	ContextDatabaseKey       = "database"
	ContextSqliteDatabaseKey = "sqlite_database"
)

func NewApplication(serverVersion string, serverPort int, db *database.Database) *Application {
	return &Application{
		serverVersion: serverVersion,
		serverPort:    serverPort,
		db:            db,
		contextVars:   make(map[string]any),
	}
}

func (app *Application) AddContextVar(key string, value any) {
	app.contextVars[key] = value
}

func (app *Application) Serve() {
	listenAddr := fmt.Sprintf(":%d", app.serverPort)
	log.Printf("Starting server on %s", listenAddr)
	contextFn := func(net.Listener) context.Context {
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextApplicationKey, app)
		ctx = context.WithValue(ctx, ContextDatabaseKey, app.db)
		ctx = context.WithValue(ctx, ContextSqliteDatabaseKey, app.db.GetDB())
		for key, value := range app.contextVars {
			ctx = context.WithValue(ctx, key, value)
		}
		return ctx
	}
	server := &http.Server{Addr: listenAddr, Handler: nil, BaseContext: contextFn}
	log.Fatal(server.ListenAndServe())
}

func (app *Application) GetDatabase() *database.Database {
	return app.db
}

func (app *Application) GetSqliteDB() *sqlx.DB {
	return app.db.GetDB()
}
