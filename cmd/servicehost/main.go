package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/sqlproxy/host"
	wasihost "github.com/tomyedwab/yesterday/wasi/host"
	"github.com/tomyedwab/yesterday/wasi/types"
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
	ContextKeyAllocator
)

func writeSlice(ctx context.Context, m api.Module, destPtr uint32, data []byte) uint32 {
	alloc := ctx.Value(ContextKeyAllocator).(*wasihost.Allocator)
	mem, err := alloc.Alloc(ctx, m, uint32(len(data)))
	if err != nil {
		fmt.Printf("Failed to allocate memory: %v\n", err)
		return 0
	}
	m.Memory().Write(mem, data)
	m.Memory().WriteUint32Le(destPtr, mem)
	return uint32(len(data))
}

func writeSliceRet(ctx context.Context, m api.Module, data []byte) (uint32, uint32) {
	alloc := ctx.Value(ContextKeyAllocator).(*wasihost.Allocator)
	mem, err := alloc.Alloc(ctx, m, uint32(len(data)))
	if err != nil {
		fmt.Printf("Failed to allocate memory: %v\n", err)
		return 0, 0
	}
	m.Memory().Write(mem, data)
	return mem, uint32(len(data))
}

func readBytes(m api.Module, offset, byteCount uint32) []byte {
	buf, ok := m.Memory().Read(offset, byteCount)
	if !ok {
		log.Panicf("Memory.Read(%d, %d) out of range", offset, byteCount)
	}
	return buf
}

func initModule(ctx context.Context, m api.Module, versionOffset, versionByteCount uint32) {
	db := ctx.Value(ContextKeyDB).(*database.Database)
	if db == nil {
		log.Panicln("Missing database in context")
	}
	db.SetVersion(string(readBytes(m, versionOffset, versionByteCount)))
}

func getEnv(ctx context.Context, m api.Module, keyOffset, keyByteCount, destPtr uint32) uint32 {
	key := string(readBytes(m, keyOffset, keyByteCount))
	value := os.Getenv(key)
	return writeSlice(ctx, m, destPtr, []byte(value))
}

func getTime(ctx context.Context, m api.Module) uint64 {
	now := time.Now().Unix()
	return uint64(now)
}

func writeLog(ctx context.Context, m api.Module, logOffset, logByteCount uint32) {
	fmt.Println(string(readBytes(m, logOffset, logByteCount)))
}

func createUUID(ctx context.Context, m api.Module, destPtr uint32) uint32 {
	newID := uuid.New().String()
	return writeSlice(ctx, m, destPtr, []byte(newID))
}

func registerHandler(ctx context.Context, m api.Module, uriOffset, uriByteCount uint32, handlerId uint32) {
	alloc := ctx.Value(ContextKeyAllocator).(*wasihost.Allocator)
	uri := string(readBytes(m, uriOffset, uriByteCount))

	fmt.Printf("Registering handler %d for %s\n", handlerId, uri)
	http.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
		requestCtx := context.WithValue(ctx, ContextKeyRequest, RequestContext{
			w: w,
			r: r,
		})

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Failed to read request body: %v\n", err)
			return
		}

		cookieMap := make(map[string]string)
		for _, cookie := range r.Cookies() {
			cookieMap[cookie.Name] = cookie.Value
		}

		params := types.RequestParams{
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
			Body:     string(bodyBytes),
			Cookies:  cookieMap,
		}
		jsonBytes, _ := json.Marshal(params)
		mem, size := writeSliceRet(ctx, m, jsonBytes)
		if size == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Failed to allocate memory")
			return
		}
		defer alloc.Free(mem)

		handlerFn := m.ExportedFunction("handle_request")
		if handlerFn == nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Request handler not found")
			return
		}
		_, err = handlerFn.Call(requestCtx, uint64(mem), uint64(size), uint64(handlerId))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Request returned an error: %v\n", err)
			return
		}
	})
}

func reportEventError(ctx context.Context, m api.Module, errorOffset, errorByteCount uint32) {
	error := string(readBytes(m, errorOffset, errorByteCount))
	fmt.Printf("WASI client reported event error: %s\n", error)
}

func registerEventHandler(ctx context.Context, m api.Module, eventTypeOffset, eventTypeByteCount uint32, handlerId uint32) {
	alloc := ctx.Value(ContextKeyAllocator).(*wasihost.Allocator)
	eventType := string(readBytes(m, eventTypeOffset, eventTypeByteCount))
	fmt.Printf("Registering event handler %d for %s\n", handlerId, eventType)

	db := ctx.Value(ContextKeyDB).(*database.Database)
	if db == nil {
		log.Panicln("Missing database in context")
	}

	host := ctx.Value(ContextKeySqliteHost).(*host.SQLHost)
	if host == nil {
		log.Panicln("Missing sqlite host in context")
	}
	database.AddGenericEventHandler(db, eventType, func(tx *sqlx.Tx, eventJson []byte) (bool, error) {
		eventMem, eventSize := writeSliceRet(ctx, m, eventJson)
		if eventSize == 0 {
			return false, fmt.Errorf("error allocating memory")
		}
		defer alloc.Free(eventMem)

		txMem, txSize := writeSliceRet(ctx, m, []byte(host.RegisterTx(tx.Tx)))
		if txSize == 0 {
			return false, fmt.Errorf("error allocating memory")
		}
		defer alloc.Free(txMem)

		handlerFn := m.ExportedFunction("handle_event")
		if handlerFn == nil {
			return false, fmt.Errorf("event handler not found")
		}
		value, err := handlerFn.Call(ctx, uint64(eventMem), uint64(eventSize), uint64(txMem), uint64(txSize), uint64(handlerId))
		if err != nil {
			return false, fmt.Errorf("event handler returned an error: %v", err)
		}
		if int32(value[0]) == -1 {
			return false, fmt.Errorf("unknown error in wasi event handler")
		}
		if int32(value[0]) == 1 {
			return true, nil
		}
		return false, nil
	})
}

func writeResponse(requestCtx context.Context, m api.Module, respOffset, respByteCount uint32) {
	requestContext := requestCtx.Value(ContextKeyRequest)
	if requestContext == nil {
		log.Panicln("Missing request context")
	}
	responseJson := readBytes(m, respOffset, respByteCount)
	var response types.Response
	err := json.Unmarshal(responseJson, &response)
	w := requestContext.(RequestContext).w
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	for k, v := range response.Headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(response.Status)
	w.Write([]byte(response.Body))
}

func sqliteHostHandler(ctx context.Context, m api.Module, reqOffset, reqByteCount, destPtr uint32) uint32 {
	request := readBytes(m, reqOffset, reqByteCount)
	fmt.Printf("REQ: %s\n", string(request))

	sqliteHost := ctx.Value(ContextKeySqliteHost)
	if sqliteHost == nil {
		log.Panicln("Missing sqlite host in context")
	}

	response, err := sqliteHost.(*host.SQLHost).HandleRequest([]byte(request))
	if err != nil {
		log.Printf("Error handling sqlite request: %v\n", err)
		return writeSlice(ctx, m, destPtr, []byte(err.Error()))
	}
	return writeSlice(ctx, m, destPtr, response)
}

func crossServiceRequest(ctx context.Context, m api.Module, reqOffset, reqByteCount, destPtr uint32) uint32 {
	var response types.CrossServiceResponse
	request := readBytes(m, reqOffset, reqByteCount)
	var crossServiceRequest types.CrossServiceRequest
	err := json.Unmarshal(request, &crossServiceRequest)
	if err != nil {
		response.Status = http.StatusInternalServerError
		response.Body = "Error unmarshaling cross service request: " + err.Error()
		responseJson, _ := json.Marshal(response)
		return writeSlice(ctx, m, destPtr, responseJson)
	}

	csReq := http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "internal.yesterday.localhost:8443", Path: crossServiceRequest.Path},
		Header: http.Header{
			"Content-Type":     []string{"application/json"},
			"X-Application-Id": []string{crossServiceRequest.ApplicationID},
			"Authorization":    []string{"Bearer " + os.Getenv("INTERNAL_SECRET")},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(crossServiceRequest.Body))),
	}
	// TODO(tom) Hopefully we can come up with a better solution for
	// certificates that doesn't require disabling verification.
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(&csReq)
	if err != nil {
		response.Status = http.StatusInternalServerError
		response.Body = "Error making cross service request: " + err.Error()
		responseJson, _ := json.Marshal(response)
		return writeSlice(ctx, m, destPtr, responseJson)
	}
	defer resp.Body.Close()

	response.Status = resp.StatusCode
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		response.Status = http.StatusInternalServerError
		response.Body = err.Error()
		responseJson, _ := json.Marshal(response)
		return writeSlice(ctx, m, destPtr, responseJson)
	}
	response.Body = string(bodyBytes)
	responseJson, _ := json.Marshal(response)
	return writeSlice(ctx, m, destPtr, responseJson)
}

func main() {
	// TODO(tom): More flexible configuration sharing from nexushub
	wasmFile := flag.String("wasm", "", "Path to the WASM file to load")
	dbPathFlag := flag.String("dbPath", "", "Path to the SQLite database file")
	port := flag.Int("port", 8080, "Port for the HTTP server")
	flag.Parse()

	if *wasmFile == "" {
		log.Fatal("WASM file path must be provided via -wasm flag")
	}

	if *dbPathFlag == "" {
		log.Fatal("Database path must be provided via -dbPath flag")
	}

	wasmBytes, err := os.ReadFile(*wasmFile)
	if err != nil {
		log.Fatalf("Failed to read WASM file %s: %v", *wasmFile, err)
	}

	dbPath := *dbPathFlag

	db, err := database.Connect("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqliteHost := host.NewSQLHost(db.GetDB().DB)

	allocator := wasihost.NewAllocator()

	// Initialize WASI runtime
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKeyDB, db)
	ctx = context.WithValue(ctx, ContextKeySqliteHost, sqliteHost)
	ctx = context.WithValue(ctx, ContextKeyAllocator, allocator)

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	_, err = r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(initModule).Export("init_module").
		NewFunctionBuilder().WithFunc(getEnv).Export("get_env").
		NewFunctionBuilder().WithFunc(getTime).Export("get_time").
		NewFunctionBuilder().WithFunc(writeLog).Export("write_log").
		NewFunctionBuilder().WithFunc(createUUID).Export("create_uuid").
		NewFunctionBuilder().WithFunc(writeResponse).Export("write_response").
		NewFunctionBuilder().WithFunc(registerHandler).Export("register_handler").
		NewFunctionBuilder().WithFunc(registerEventHandler).Export("register_event_handler").
		NewFunctionBuilder().WithFunc(reportEventError).Export("report_event_error").
		NewFunctionBuilder().WithFunc(sqliteHostHandler).Export("sqlite_host_handler").
		NewFunctionBuilder().WithFunc(crossServiceRequest).Export("cross_service_request").
		Instantiate(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Instantiate a WebAssembly module and call the `init` function
	// automatically.
	_, err = r.InstantiateWithConfig(
		ctx,
		wasmBytes,
		wazero.NewModuleConfig().WithStartFunctions("_initialize"),
	)
	if err != nil {
		log.Panicln(err)
	}

	err = db.Initialize()
	if err != nil {
		log.Panicln(err)
	}

	listenAddr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
