# Technical Specification: Golang SQL Proxy Driver for WASI

**Version:** 1.0
**Date:** 2025-05-28
**Author:** Cascade AI

## 1. Abstract

This document outlines the design for a new Golang module that implements a `database/sql` driver. This driver is specifically designed to operate within a WebAssembly System Interface (WASI) module. Its primary function is to proxy SQL queries received by the WASI module to a handler function. This handler function, in turn, passes the queries to the host environment, which executes them against an SQLite database.

## 2. Background

Modern applications increasingly leverage WebAssembly (Wasm) for its portability and security benefits. When a Golang application compiled to WASI needs to interact with a database, it cannot directly access system resources like a traditional database connection.

The standard Go `database/sql` package provides a generic interface for database access, with specific drivers implementing the details for different database systems. This project aims to create such a driver that bridges the gap between a WASI Go module and a host-managed SQLite database.

The core idea is to delegate query execution to the host environment. The WASI module, via this driver, will serialize query requests, send them to the host through a pre-defined handler function (an exported host function callable from WASI, or an imported WASI function provided by the host), and then deserialize the results returned by the host.

## 3. Goals

*   Implement a fully compliant Golang `database/sql/driver` interface.
*   Enable Golang code running in a WASI module to execute SQL queries against an SQLite database managed by the host.
*   Define a clear and efficient mechanism for proxying queries and results between the WASI module and the host.
*   Ensure the driver is lightweight and introduces minimal overhead.
*   Provide robust error handling, propagating errors from the host/SQLite back to the WASI module.

## 4. Non-Goals

*   Implement a full SQLite engine within the WASI module.
*   Support database systems other than SQLite (though the proxy mechanism could be generalized in the future).
*   Provide advanced connection pooling or management features within the WASI driver itself (this might be handled by the host or the `database/sql` package).
*   Directly manage database files or connections from within the WASI module.

## 5. Design

### 5.1. Overview

The `sqlproxy` driver will consist of several components that implement the `database/sql/driver` interfaces. When a Go application in WASI uses `database/sql` with this driver, the driver will:
1.  Receive SQL queries and arguments.
2.  Serialize this information into a defined format (e.g., JSON).
3.  Call a host-provided handler function, passing the serialized query.
4.  The host handler function will:
    a.  Deserialize the query.
    b.  Execute it against its local SQLite database.
    c.  Serialize the results (or error).
    d.  Return the serialized response to the WASI module.
5.  The driver in the WASI module will deserialize the response and return it to the application through the `database/sql` interfaces.

### 5.2. Driver Implementation (`database/sql/driver` interface)

The following interfaces from `database/sql/driver` will be implemented:

*   **`driver.Driver`**:
    *   `Open(name string) (driver.Conn, error)`: The `name` string could be used to pass configuration or identify the specific host handler if multiple are available. For simplicity, it might be ignored initially if only one proxy channel is assumed. Returns a `sqlproxy.Conn`.

*   **`driver.Conn`**:
    *   `Prepare(query string) (driver.Stmt, error)`: Serializes the query string and sends a "prepare" command to the host. The host could optionally pre-compile the statement. Returns a `sqlproxy.Stmt`.
    *   `Close() error`: Sends a "close connection" command to the host or simply cleans up local resources.
    *   `Begin() (driver.Tx, error)`: Sends a "begin transaction" command to the host. Returns a `sqlproxy.Tx`.
    *   `BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error)`: For context-aware transaction initiation.
    *   `ExecerContext`, `QueryerContext`, `ConnPrepareContext`, `ConnBeginTx`: For context propagation. (Not implemented)
    *   `Pinger`: To check the liveness of the host connection. (Not implemented)
    *   `SessionResetter`: To reset session state. (Not implemented)
    *   `Validator`: To check if a connection is still valid. (Not implemented)

*   **`driver.Stmt`**:
    *   `Close() error`: Sends a "close statement" command to the host. (Not implemented)
    *   `NumInput() int`: Can return -1 (unknown) or be determined from the host after preparation.
    *   `Exec(args []driver.Value) (driver.Result, error)`: Serializes arguments, sends an "execute statement" command with args to the host. Deserializes the host's `driver.Result`.
    *   `Query(args []driver.Value) (driver.Rows, error)`: Serializes arguments, sends a "query statement" command with args to the host. Deserializes the host's `driver.Rows`.
    *   `ColumnConverter`**: If type conversions are needed. (Not implemented)
    *   `ExecerContext`, `QueryerContext`**: For context propagation. (Not implemented)

*   **`driver.Tx`**:
    *   `Commit() error`: Sends a "commit transaction" command to the host. (Not implemented)
    *   `Rollback() error`: Sends a "rollback transaction" command to the host. (Not implemented)

*   **`driver.Result`**:
    *   `LastInsertId() (int64, error)`: Value provided by the host.
    *   `RowsAffected() (int64, error)`: Value provided by the host.

*   **`driver.Rows`**:
    *   `Columns() []string`: Column names provided by the host.
    *   `Close() error`: Signals to the host that these rows are no longer needed.
    *   `Next(dest []driver.Value) error`: Fetches the next row data from the host and populates `dest`. Returns `io.EOF` when no more rows.
    *   `RowsColumnTypeDatabaseTypeName`, `RowsColumnTypeScanType`, etc.**: For detailed column type information. (Not implemented)

### 5.3. WASI Integration & Handler Function

Communication between the WASI module and the host will occur via a handler function. This function must be provided by the host environment and made callable from the WASI module.

**Handler Function Signature (Host Side):**
The host will need to expose a function that the WASI module can call. For example, if using `wapc-go` or a similar mechanism, this would be a registered host binding.
A conceptual signature for the function called by the WASI module (implemented by the host):
```go
// This function is conceptually what the WASI module calls.
// The actual implementation depends on the WASI host and guest SDK.
// It might involve shared memory and explicit calls to exported WASI functions
// or imported host functions.
func handleSQLProxyRequest(requestPayload []byte) (responsePayload []byte, err error)
```

The `requestPayload` will contain the serialized command (e.g., "query", "exec", "begin_tx"), SQL string, arguments, and any other necessary metadata.
The `responsePayload` will contain the serialized result set, last insert ID, rows affected, or error information.

### 5.4. Query Proxying and Data Exchange Format

A clear and efficient data exchange format is crucial. JSON is a strong candidate due to its widespread support in Go and ease of use, though more performant binary formats like Protocol Buffers or MessagePack could be considered for optimization.

**Example Request Structure (JSON):**
```json
{
  "command": "query", // "exec", "prepare", "commit", "rollback", "close_stmt", "close_conn"
  "sql": "SELECT id, name FROM users WHERE id = ?",
  "args": [1], // driver.Value types need to be handled (int64, float64, bool, []byte, string, time.Time)
  "stmt_id": "optional_statement_id_from_prepare", // For prepared statements
  "tx_id": "optional_transaction_id" // For operations within a transaction
}
```

**Example Response Structure (JSON for `driver.Rows`):**
For `Query` command:
```json
{
  "columns": ["id", "name"],
  "rows": [ // Or a mechanism to stream rows
    [1, "Alice"],
    [2, "Bob"]
  ],
  "error": null // or error message string
}
```
For `Exec` command:
```json
{
  "last_insert_id": 0, // If applicable
  "rows_affected": 1,
  "error": null // or error message string
}
```

**Data Type Handling:**
`driver.Value` can be `int64`, `float64`, `bool`, `[]byte`, `string`, `time.Time`. These types must be serializable to JSON (or chosen format) and correctly deserialized by the host for SQLite, and vice-versa for results. `time.Time` might be serialized as ISO 8601 strings or Unix timestamps. `[]byte` as base64 encoded strings.

### 5.5. Error Handling

Errors originating from the host (SQLite errors, serialization errors, etc.) must be propagated back to the WASI module. The response structure should include an error field. The driver will convert these error messages/codes into Go `error` types.

## 6. API

### 6.1. Driver Registration

Users of the driver in a WASI module will register it as usual:
```go
import (
	"database/sql"
	_ "github.com/tomyedwab/yesterday/sqlproxy/driver"
)

// HostHandler is a function provided by the WASI host environment.
// Its actual signature and how it's obtained will depend on the specific
// WASI SDK (e.g., TinyGo's wasi imports, or a library like wazero's host modules).
// For this spec, we assume a function `ProxyQueryToHost` exists.
// This function needs to be set for the driver.
//
// Example:
// sqlproxy.SetHostHandler(func(payload []byte) ([]byte, error) {
//     return host.Call("sqlite_handler", payload)
// })


func main() {
    // The DSN might be used to pass info to the host, or could be a dummy string
    // if the host interaction is configured globally via SetHostHandler.
	db, err := sql.Open("sqlproxy", "dsn_if_needed")
	if err != nil {
		// handle error
	}
	defer db.Close()

	// Use db as a standard *sql.DB object
	rows, err := db.Query("SELECT * FROM my_table")
	// ...
}
```
The driver package (`github.com/tomyedwab/yesterday/sqlproxy/driver`) will have an `init()` function that calls `sql.Register("sqlproxy", &SQLProxyDriver{})`.

A mechanism to provide the actual host communication function to the driver will be needed. This could be a package-level function:
```go
package driver

// HostQueryHandler is the type for the function that communicates with the host.
type HostQueryHandler func(requestPayload []byte) (responsePayload []byte, err error)

var hostHandler HostQueryHandler

// SetHostHandler allows the WASI application to set the function
// used to proxy queries to the host. This must be called before
// any database operations.
func SetHostHandler(handler HostQueryHandler) {
	hostHandler = handler
}
```
The `driver.Driver.Open` method would then check if `hostHandler` is set.

## 7. Security Considerations

*   **Input Sanitization**: While SQLite itself handles SQL injection if parameterized queries are used correctly, the host-side handler must ensure it's using the provided arguments as parameters and not constructing SQL strings directly from potentially unsafe input from the WASI module. The driver should always encourage/enforce parameterized queries.
*   **Data Exposure**: The host controls access to the SQLite database. The handler function acts as a gatekeeper. Ensure the host environment properly sandboxes and restricts the capabilities of the WASI module.
*   **Resource Limits**: The host should consider imposing limits on query complexity, result size, or execution time to prevent a misbehaving WASI module from consuming excessive resources.

## 8. Future Considerations

*   **Performance**: Investigate more performant serialization formats (e.g., Protocol Buffers, MessagePack) if JSON proves to be a bottleneck.
*   **Streaming Rows**: For large result sets, instead of serializing all rows in one response, implement a mechanism to stream rows one by one or in batches to reduce memory usage. This would require changes to the `sqlproxy.Rows.Next` method and the host communication protocol.
*   **Advanced Host Configuration**: Allow more detailed configuration to be passed via the DSN string during `sql.Open`.
*   **Named Parameters**: Support for named parameters if the underlying SQLite usage on the host supports it well.
*   **Support for other WASI targets**: While focused on Golang, the proxying principles could be adapted for other languages running in WASI.
*   **Connection Pooling on Host**: The host handler could implement connection pooling to the SQLite database for better performance if multiple "connections" are opened by the WASI module.

