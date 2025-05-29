package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/sqlproxy/host"
)

type RequestContext struct {
	w http.ResponseWriter
	r *http.Request
}

type ContextKey int

const (
	ContextKeyRequest ContextKey = iota
	ContextKeyDB
	ContextKeySqliteHost
)

type RequestParams struct {
	Path     string
	RawQuery string
}

//go:embed testdata/echo.wasm
var echoWasm []byte

func readBytes(m api.Module, offset, byteCount uint32) []byte {
	buf, ok := m.Memory().Read(offset, byteCount)
	if !ok {
		log.Panicf("Memory.Read(%d, %d) out of range", offset, byteCount)
	}
	return buf
}

func writeBytes(m api.Module, data []byte) (freeFn func(), handle uint32) {
	alloc := m.ExportedFunction("alloc_bytes")
	free := m.ExportedFunction("free_bytes")
	result, err := alloc.Call(context.Background(), uint64(len(data)))
	if err != nil {
		log.Panicln(err)
	}
	handle = uint32(result[0] >> 32)
	ptr := uint32(result[0])
	freeFn = func() {
		free.Call(context.Background(), uint64(handle))
	}
	fmt.Printf("Writing %d bytes to %d on handle %d\n", len(data), ptr, handle)
	if !m.Memory().Write(ptr, data) {
		log.Panicln("Memory.Write failed")
	}
	return
}

func registerHandler(ctx context.Context, m api.Module, uriOffset, uriByteCount uint32, handlerId uint32) {
	uri := string(readBytes(m, uriOffset, uriByteCount))

	fmt.Printf("Registering handler %d for %s\n", handlerId, uri)
	http.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
		requestCtx := context.WithValue(ctx, ContextKeyRequest, RequestContext{
			w: w,
			r: r,
		})

		params := RequestParams{
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		jsonBytes, _ := json.Marshal(params)
		freeFn, handle := writeBytes(m, jsonBytes)
		defer freeFn()

		handlerFn := m.ExportedFunction("handle_request")
		if handlerFn == nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Request handler not found")
			return
		}
		_, err := handlerFn.Call(requestCtx, uint64(handle), uint64(handlerId))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Request returned an error: %v\n", err)
			return
		}
	})
}

func writeResponse(requestCtx context.Context, m api.Module, respOffset, respByteCount uint32) {
	requestContext := requestCtx.Value(ContextKeyRequest)
	if requestContext == nil {
		log.Panicln("Missing request context")
	}
	response := readBytes(m, respOffset, respByteCount)
	requestContext.(RequestContext).w.Write(response)
}

func sqliteHostHandler(requestCtx context.Context, m api.Module, reqOffset, reqByteCount uint32) uint64 {
	request := readBytes(m, reqOffset, reqByteCount)
	fmt.Printf("REQ: %s\n", string(request))

	sqliteHost := requestCtx.Value(ContextKeySqliteHost)
	if sqliteHost == nil {
		log.Panicln("Missing sqlite host in context")
	}

	response, err := sqliteHost.(*host.SQLHost).HandleRequest([]byte(request))
	if err != nil {
		log.Printf("Error handling sqlite request: %v\n", err)
		_, handle := writeBytes(m, []byte(err.Error()))
		return uint64(handle) | (1 << 32)
	}
	_, handle := writeBytes(m, response)
	return uint64(handle)
}

func main() {
	dbPath := "./users.db"

	// Define handlers for user state
	handlers := map[string]database.EventUpdateHandler{}

	// Connect to the database and initialize schema/handlers
	db, err := database.Connect("sqlite3", dbPath, "0.0.0", handlers)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqliteHost := host.NewSQLHost(db.GetDB().DB)

	// Initialize WASI runtime
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKeyDB, db)
	ctx = context.WithValue(ctx, ContextKeySqliteHost, sqliteHost)

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	_, err = r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(writeResponse).Export("write_response").
		NewFunctionBuilder().WithFunc(registerHandler).Export("register_handler").
		NewFunctionBuilder().WithFunc(sqliteHostHandler).Export("sqlite_host_handler").
		Instantiate(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Instantiate a WebAssembly module and call the `init` function
	// automatically.
	_, err = r.InstantiateWithConfig(
		ctx,
		echoWasm,
		wazero.NewModuleConfig().WithStartFunctions("_initialize"),
	)
	if err != nil {
		log.Panicln(err)
	}

	// Initialize HTTP endpoints using our event mapper
	db.InitHandlers(func(rawMessage *json.RawMessage, generic *events.GenericEvent) (events.Event, error) {
		return generic, nil
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
