package applib

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/applib/database"
)

type Application struct {
	db          *database.Database
	contextVars map[string]any
}

var (
	ContextApplicationKey    = "application"
	ContextDatabaseKey       = "database"
	ContextSqliteDatabaseKey = "sqlite_database"
)

func NewApplication(db *database.Database) *Application {
	return &Application{
		db:          db,
		contextVars: make(map[string]any),
	}
}

func (app *Application) AddContextVar(key string, value any) {
	app.contextVars[key] = value
}

func (app *Application) Serve() {
	log.Printf("Starting server on port 80")
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
	server := &http.Server{Addr: "127.0.0.1:80", Handler: nil, BaseContext: contextFn}
	log.Fatal(server.ListenAndServe())
}

func (app *Application) GetDatabase() *database.Database {
	return app.db
}

func (app *Application) GetSqliteDB() *sqlx.DB {
	return app.db.GetDB()
}
