package main

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"database/sql"

	sqlproxy "github.com/tomyedwab/yesterday/sqlproxy/driver"
)

type RequestParams struct {
	Path     string
	RawQuery string
}

//go:wasmimport env write_response
func write_response(message string)

//go:wasmimport env register_handler
func register_handler(uri string, handlerId uint32)

//go:wasmimport env sqlite_host_handler
func sqlite_host_handler(requestPayload string) (responseHandle uint64)

var BYTE_HANDLES map[uint32][]byte
var NEXT_BYTE_HANDLE uint32

//go:wasmexport alloc_bytes
func allocBytes(size uint32) uint64 {
	bytes := make([]byte, size)
	handle := NEXT_BYTE_HANDLE
	BYTE_HANDLES[handle] = bytes
	NEXT_BYTE_HANDLE++
	return uint64(uint32(handle))<<32 | uint64(uintptr(unsafe.Pointer(&bytes[0])))
}

//go:wasmexport free_bytes
func freeBytes(handle uint32) {
	delete(BYTE_HANDLES, handle)
}

//go:wasmexport handle_request
func handle_request(byteHandle uint32, handlerId uint32) int32 {
	var params RequestParams
	err := json.Unmarshal(BYTE_HANDLES[byteHandle], &params)
	if err != nil {
		write_response("Internal error: json decoding")
		return 0
	}

	sqlproxy.SetHostHandler(func(payload []byte) ([]byte, error) {
		responseHandle := sqlite_host_handler(string(payload))
		ret := BYTE_HANDLES[uint32(responseHandle)]
		freeBytes(uint32(responseHandle))
		if responseHandle>>32 != 0 {
			return nil, fmt.Errorf("sqlite_host_handler returned error: %s", string(ret))
		}
		return ret, nil
	})

	db, err := sql.Open("sqlproxy", "")
	if err != nil {
		write_response(fmt.Sprintf("Internal error: sql.Open failed: %v", err))
		return 0
	}
	defer db.Close()

	// Use db as a standard *sql.DB object
	rows, err := db.Query("SELECT id, username FROM users_v1")
	if err != nil {
		write_response(fmt.Sprintf("Internal error: sql.Query failed: %v", err))
		return 0
	}
	defer rows.Close()

	var ret string
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			write_response(fmt.Sprintf("Internal error: sql.Scan failed: %v", err))
			return 0
		}
		ret += fmt.Sprintf("User %d: %s\n", id, name)
	}
	if err := rows.Err(); err != nil {
		write_response(fmt.Sprintf("Internal error: sql.Rows.Err failed: %v", err))
		return 0
	}
	write_response(ret)
	return 0
}

//go:wasmexport init
func init() {
	BYTE_HANDLES = make(map[uint32][]byte)
	NEXT_BYTE_HANDLE = 1
	register_handler("/api/echo", 1)
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
