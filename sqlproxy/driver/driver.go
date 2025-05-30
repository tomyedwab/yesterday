package driver

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/tomyedwab/yesterday/sqlproxy/types"
)

// CallHost is a function provided by the WASI host environment to handle SQL proxy requests.
// This function must be set by the user of this driver in the WASI environment.
var CallHost func(requestPayload []byte) (responsePayload []byte, err error)

// SetHostHandler allows the WASI application to set the function
// used to proxy queries to the host. This must be called before
// any database operations.
func SetHostHandler(handler func(requestPayload []byte) (responsePayload []byte, err error)) {
	CallHost = handler
}

const driverName = "sqlproxy"

func init() {
	sql.Register(driverName, &Driver{})
}

// --- Driver implementation ---

// Driver is the SQL driver for the proxy.
type Driver struct{}

// Open returns a new connection to the database.
// If name is non-empty, then there is a transaction being passed from the host
// environment.
func (d *Driver) Open(name string) (driver.Conn, error) {
	if CallHost == nil {
		return nil, fmt.Errorf("sqlproxy: CallHost function is not set")
	}
	return &Conn{HostTxID: name}, nil
}

// --- Connection implementation ---

// Conn implements the driver.Conn interface.
type Conn struct {
	HostTxID    string // For transactions initiated by the host and passed via DSN
	currentTxID string // For transactions initiated by driver.Begin()
}

// Prepare returns a prepared statement, suitable for query or execution.
func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	sqlReq := types.SQLRequest{Command: "prepare", SQL: query}
	if c.currentTxID != "" {
		sqlReq.TxID = c.currentTxID
	}

	reqPayload, err := json.Marshal(sqlReq)
	if err != nil {
		return nil, fmt.Errorf("sqlproxy: failed to marshal prepare request: %w", err)
	}

	respPayload, err := CallHost(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("sqlproxy: CallHost for prepare failed: %w", err)
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		return nil, fmt.Errorf("sqlproxy: failed to unmarshal prepare response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("sqlproxy: host prepare error: %s", resp.Error)
	}
	if resp.StmtID == "" {
		return nil, fmt.Errorf("sqlproxy: host did not return a StmtID for prepare")
	}

	// Store the transaction ID with the statement if it was prepared within a transaction
	stmtTxID := ""
	if c.currentTxID != "" {
		stmtTxID = c.currentTxID
	}

	return &Stmt{conn: c, query: query, stmtID: resp.StmtID, txID: stmtTxID}, nil
}

// Close invalidates and potentially releases resources associated with the connection.
func (c *Conn) Close() error {
	reqPayload, err := json.Marshal(types.SQLRequest{Command: "close_conn"})
	if err != nil {
		return fmt.Errorf("sqlproxy: failed to marshal close_conn request: %w", err)
	}

	respPayload, err := CallHost(reqPayload)
	if err != nil {
		// Decide if error on close should be propagated or logged
		return fmt.Errorf("sqlproxy: CallHost for close_conn failed: %w", err)
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		return fmt.Errorf("sqlproxy: failed to unmarshal close_conn response: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("sqlproxy: host close_conn error: %s", resp.Error)
	}
	return nil
}

// Begin starts and returns a new transaction.
func (c *Conn) Begin() (driver.Tx, error) {
	if c.currentTxID != "" {
		return nil, fmt.Errorf("sqlproxy: transaction already active on this connection (TxID: %s)", c.currentTxID)
	}

	if c.HostTxID != "" { // If connection was opened with a HostTxID from DSN
		c.currentTxID = c.HostTxID
		return &Tx{conn: c, txID: c.HostTxID}, nil
	}

	// For BeginTx (which we are not implementing yet as per spec non-goals for context propagation),
	// we would pass TxOptions here.
	reqPayload, err := json.Marshal(types.SQLRequest{Command: "begin_tx"})
	if err != nil {
		return nil, fmt.Errorf("sqlproxy: failed to marshal begin_tx request: %w", err)
	}

	respPayload, err := CallHost(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("sqlproxy: CallHost for begin_tx failed: %w", err)
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		return nil, fmt.Errorf("sqlproxy: failed to unmarshal begin_tx response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("sqlproxy: host begin_tx error: %s", resp.Error)
	}
	if resp.TxID == "" {
		return nil, fmt.Errorf("sqlproxy: host did not return a transaction ID for begin_tx")
	}

	c.currentTxID = resp.TxID // Set current transaction ID on the connection
	return &Tx{conn: c, txID: resp.TxID}, nil
}

// --- Statement implementation ---

// Stmt implements the driver.Stmt interface.
type Stmt struct {
	conn   *Conn
	query  string // Original query, mainly for context/debugging
	stmtID string // Host-provided statement ID
	txID   string // Transaction ID if this statement was prepared within a transaction
}

// Close closes the statement.
func (s *Stmt) Close() error {
	reqPayload, err := json.Marshal(types.SQLRequest{Command: "close_stmt", StmtID: s.stmtID})
	if err != nil {
		return fmt.Errorf("sqlproxy: failed to marshal close_stmt request: %w", err)
	}

	respPayload, err := CallHost(reqPayload)
	if err != nil {
		return fmt.Errorf("sqlproxy: CallHost for close_stmt failed: %w", err)
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		return fmt.Errorf("sqlproxy: failed to unmarshal close_stmt response: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("sqlproxy: host close_stmt error: %s", resp.Error)
	}
	s.stmtID = "" // Mark as closed
	return nil
}

// NumInput returns the number of placeholder parameters.
// Returns -1 if the driver doesn't know its value.
func (s *Stmt) NumInput() int {
	return -1 // As per spec: "Can return -1 (unknown) or be determined from the host after preparation."
}

func convertDriverValues(args []driver.Value) []interface{} {
	interfaceArgs := make([]interface{}, len(args))
	for i, v := range args {
		// Handle specific types that need conversion for JSON, e.g., time.Time, []byte.
		// json.Marshal handles time.Time to RFC3339 string and []byte to base64 string by default.
		switch val := v.(type) {
		case time.Time:
			interfaceArgs[i] = val.Format(time.RFC3339Nano)
		default:
			interfaceArgs[i] = v
		}
	}
	return interfaceArgs
}

// Exec executes a prepared statement with the given arguments and returns a Result.
func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	convertedArgs := convertDriverValues(args)
	sqlReq := types.SQLRequest{Command: "exec", StmtID: s.stmtID, Args: convertedArgs}
	if s.txID != "" { // If statement is part of a transaction
		sqlReq.TxID = s.txID
	}

	reqPayload, err := json.Marshal(sqlReq)
	if err != nil {
		return nil, fmt.Errorf("sqlproxy: failed to marshal exec request: %w", err)
	}

	respPayload, err := CallHost(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("sqlproxy: CallHost for exec failed: %w", err)
	}

	var resp types.ExecResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		return nil, fmt.Errorf("sqlproxy: failed to unmarshal exec response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("sqlproxy: host exec error: %s", resp.Error)
	}

	return &sqlProxyResult{lastInsertID: resp.LastInsertID, rowsAffected: resp.RowsAffected}, nil
}

// Query executes a prepared statement with the given arguments and returns Rows.
func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	convertedArgs := convertDriverValues(args)
	sqlReq := types.SQLRequest{Command: "query", StmtID: s.stmtID, Args: convertedArgs}
	if s.txID != "" { // If statement is part of a transaction
		sqlReq.TxID = s.txID
	}

	reqPayload, err := json.Marshal(sqlReq)
	if err != nil {
		return nil, fmt.Errorf("sqlproxy: failed to marshal query request: %w", err)
	}

	respPayload, err := CallHost(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("sqlproxy: CallHost for query failed: %w", err)
	}

	var resp types.QueryResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		return nil, fmt.Errorf("sqlproxy: failed to unmarshal query response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("sqlproxy: host query error: %s", resp.Error)
	}

	return &sqlProxyRows{columns: resp.Columns, data: resp.Rows, conn: s.conn}, nil
}

// --- Transaction implementation ---

// Tx implements the driver.Tx interface.
type Tx struct {
	conn *Conn
	txID string // Host-provided transaction ID
}

// Commit commits the transaction.
func (t *Tx) Commit() error {
	if t.txID == "" {
		return fmt.Errorf("sqlproxy: transaction already committed or rolled back")
	}
	reqPayload, err := json.Marshal(types.SQLRequest{Command: "commit", TxID: t.txID})
	if err != nil {
		return fmt.Errorf("sqlproxy: failed to marshal commit request: %w", err)
	}

	respPayload, err := CallHost(reqPayload)
	if err != nil {
		return fmt.Errorf("sqlproxy: CallHost for commit failed: %w", err)
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		return fmt.Errorf("sqlproxy: failed to unmarshal commit response: %w", err)
	}

	if resp.Error != "" {
		// Even if commit fails on host, the transaction is likely in an indeterminate state.
		// For safety, we clear the currentTxID on the connection, but the Tx object itself
		// retains its txID for potential inspection, though it's effectively unusable.
		t.conn.currentTxID = "" 
		return fmt.Errorf("sqlproxy: host commit error: %s (TxID: %s)", resp.Error, t.txID)
	}

	t.conn.currentTxID = "" // Clear current transaction ID on the connection
	t.txID = ""             // Mark as completed successfully
	return nil
}

// Rollback aborts the transaction.
func (t *Tx) Rollback() error {
	if t.txID == "" {
		return fmt.Errorf("sqlproxy: transaction already committed or rolled back")
	}
	reqPayload, err := json.Marshal(types.SQLRequest{Command: "rollback", TxID: t.txID})
	if err != nil {
		return fmt.Errorf("sqlproxy: failed to marshal rollback request: %w", err)
	}

	respPayload, err := CallHost(reqPayload)
	if err != nil {
		return fmt.Errorf("sqlproxy: CallHost for rollback failed: %w", err)
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		// If unmarshal fails, the host state is unknown. Clear connection's txID.
		t.conn.currentTxID = ""
		return fmt.Errorf("sqlproxy: failed to unmarshal rollback response: %w", err)
	}

	// Regardless of host response for rollback (success or error indicating already rolled back),
	// the transaction is no longer considered active on this connection from the client's perspective.
	t.conn.currentTxID = ""
	originalTxID := t.txID // Save for error message if needed
	t.txID = ""          // Mark Tx object as completed

	if resp.Error != "" {
		// It's common for Rollback to report an error if the transaction was already aborted due to an error.
		// The sql package checks for driver.ErrBadConn if Rollback fails, to see if the connection is still usable.
		// Client should discard the Tx object.
		return fmt.Errorf("sqlproxy: host rollback error: %s (TxID: %s)", resp.Error, originalTxID)
	}

	return nil
}

// --- Result implementation ---

// sqlProxyResult implements the driver.Result interface.
type sqlProxyResult struct {
	lastInsertID int64
	rowsAffected int64
}

// LastInsertId returns the database's auto-generated ID after, for example, an INSERT into a table with primary key.
func (r *sqlProxyResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

// RowsAffected returns the number of rows affected by the query.
func (r *sqlProxyResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// --- Rows implementation ---

// sqlProxyRows implements the driver.Rows interface.
type sqlProxyRows struct {
	conn            *Conn
	columns         []string
	data            [][]interface{} // All rows data, pre-fetched
	currentRowIndex int
	// stmtID might be needed if rows are fetched incrementally from host using this ID
}

// Columns returns the names of the columns. The number of columns of the result is inferred from the length of the slice.
func (r *sqlProxyRows) Columns() []string {
	return r.columns
}

// Close closes the Rows, preventing further enumeration.
func (r *sqlProxyRows) Close() error {
	// If rows were streamed from host, this would signal host to release resources.
	// Since all rows are fetched at once in this implementation, this is a client-side cleanup.
	r.data = nil
	r.currentRowIndex = 0
	// TODO: If host needs explicit close for row sets (even if all data sent),
	// a "close_rows" command with a relevant ID (e.g., StmtID or a new RowsID) would be sent here.
	return nil
}

// Next is called to populate the next row of data into the provided slice. The provided slice will be the same size as the Columns() are wide.
// Next should return io.EOF when there are no more rows.
func (r *sqlProxyRows) Next(dest []driver.Value) error {
	if r.currentRowIndex >= len(r.data) {
		return io.EOF
	}

	rowData := r.data[r.currentRowIndex]
	if len(rowData) != len(dest) {
		return fmt.Errorf("sqlproxy: column count mismatch. Expected %d, got %d", len(dest), len(rowData))
	}

	for i, val := range rowData {
		// The host provides data as []interface{}. We need to assign to dest []driver.Value.
		// database/sql will handle further conversion from driver.Value to scan targets.
		// Need to handle time.Time and []byte specifically if they are not already in correct format.
		// JSON unmarshals numbers as float64, strings as string, bools as bool.
		// []byte would be base64 string from JSON, time.Time as string.
		// This part needs careful consideration of type mapping from JSON to driver.Value.
		// For simplicity, we assign directly. The `database/sql` layer does a lot of conversion.
		// Example: If host sends time as RFC3339 string, and Scan target is time.Time,
		// `database/sql`'s `ConvertAssign` handles it.
		// If it's a base64 encoded byte array, it should be decoded here if not handled by json.Unmarshal or host.
		dest[i] = val
	}

	r.currentRowIndex++
	return nil
}

// --- Context-aware methods (Not implemented as per spec non-goals) ---
/*
func (c *Conn) Ping(ctx context.Context) error {
	// TODO: Send a "ping" command to host
	return driver.ErrSkip // Or actual implementation
}

func (c *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	// TODO: Send "begin_tx" with options to host
	return nil, driver.ErrSkip // Or actual implementation
}

func (c *Conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
    // Similar to Prepare, but with context
    return c.Prepare(query) // Fallback or implement context handling
}

func (s *Stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
    // Convert NamedValue to Value for simplicity if host doesn't support named args
    values := make([]driver.Value, len(args))
    for i, arg := range args {
        values[i] = arg.Value
    }
    return s.Exec(values) // Fallback or implement context handling
}

func (s *Stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
    // Convert NamedValue to Value
    values := make([]driver.Value, len(args))
    for i, arg := range args {
        values[i] = arg.Value
    }
    return s.Query(values) // Fallback or implement context handling
}
*/
