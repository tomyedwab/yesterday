// Package driver implements a database/sql/driver for proxying SQL queries
// from a Go application running in a WebAssembly System Interface (WASI) module
// to a host environment.
//
// The primary goal of this driver is to allow WASI-compiled Go code to interact
// with an SQLite database (or potentially other SQL databases) managed by the host.
// It achieves this by serializing SQL commands and arguments into a defined format (JSON)
// and passing them to a host-provided function.
//
// Usage:
//
//  1. Import the driver package. This will register the driver with the name "sqlproxy".
//     import _ "path/to/your/project/sqlproxy/driver"
//
// 2. Before opening a database connection, the WASI application must set the global
// `CallHost` variable in this package. This function is responsible for the actual
// communication with the host environment.
//
//	driver.CallHost = func(requestPayload []byte) (responsePayload []byte, err error) {
//	    // ... logic to send requestPayload to host and receive responsePayload ...
//	    return hostResponse, hostError
//	}
//
//  3. Open a database connection using sql.Open:
//     db, err := sql.Open("sqlproxy", "") // DSN is currently ignored
//     if err != nil {
//     // handle error
//     }
//     defer db.Close()
//
// 4. Use the *sql.DB object as usual to execute queries, prepared statements, and transactions.
//
// Communication Protocol:
//
// The driver communicates with the host by sending JSON-encoded `SQLRequest` structs
// and expects JSON-encoded responses (`QueryResponse`, `ExecResponse`, or `GeneralResponse`)
// from the `CallHost` function.
//
// Implemented Interfaces:
//
// The driver implements the following core `database/sql/driver` interfaces:
// - driver.Driver
// - driver.Conn
// - driver.Stmt
// - driver.Tx
// - driver.Result
// - driver.Rows
//
// Limitations:
//
//   - This driver relies entirely on the host for SQL execution, connection management,
//     and transaction integrity.
//   - Context-aware methods (e.g., `BeginTx`, `ExecContext`) are not fully implemented
//     to pass context to the host but will fall back to their non-contextual counterparts
//     or return `driver.ErrSkip` where appropriate if not implemented.
//   - Advanced features like connection pooling within the driver are not supported; this
//     is typically handled by `database/sql` or the host.
package driver
