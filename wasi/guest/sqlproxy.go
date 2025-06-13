package guest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/tomyedwab/yesterday/wasi/types"
)

//go:wasmimport env sqlite_host_handler
func sqlite_host_handler(requestPayload string, destPtr *uint32) int32

var hostHandler func([]byte) ([]byte, error)

// DB represents a database connection that proxies to the host
type DB struct {
	// DB doesn't hold state since the actual connection is on the host
}

// Tx represents a database transaction
type Tx struct {
	id       string
	readOnly bool // true if this tx was passed from host and cannot be committed/rolled back
}

// Stmt represents a prepared statement
type Stmt struct {
	id string
	tx *Tx // optional transaction context
}

// Result represents the result of an Exec operation
type Result struct {
	lastInsertID int64
	rowsAffected int64
}

func (r Result) LastInsertId() (int64, error) { return r.lastInsertID, nil }
func (r Result) RowsAffected() (int64, error) { return r.rowsAffected, nil }

// InitSQLProxy initializes the SQL proxy system
func InitSQLProxy() {
	hostHandler = func(payload []byte) ([]byte, error) {
		var destPtr uint32
		destSize := sqlite_host_handler(string(payload), &destPtr)
		if destSize < 0 {
			ret := GetBytesFromPtr(destPtr, uint32(-destSize))
			return nil, fmt.Errorf("sqlite_host_handler returned error: %s", string(ret))
		}
		return GetBytesFromPtr(destPtr, uint32(destSize)), nil
	}
}

// NewDB creates a new database connection proxy
func NewDB() *DB {
	return &DB{}
}

// NewTxFromID creates a transaction object from a host-provided transaction ID
// This transaction is read-only and cannot be committed or rolled back from the guest
func NewTxFromID(txID string) *Tx {
	return &Tx{id: txID, readOnly: true}
}

// sendRequest sends a request to the host and handles the response
func sendRequest(req types.SQLRequest) ([]byte, error) {
	if hostHandler == nil {
		return nil, fmt.Errorf("SQL proxy not initialized - call InitSQLProxy() first")
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	return hostHandler(reqBytes)
}

// --- DB Methods ---

// Exec executes a query without returning any rows
func (db *DB) Exec(query string, args ...any) (Result, error) {
	req := types.SQLRequest{
		Command: "exec",
		SQL:     query,
		Args:    args,
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return Result{}, err
	}

	var resp types.ExecResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return Result{}, fmt.Errorf("failed to unmarshal exec response: %w", err)
	}

	if resp.Error != "" {
		return Result{}, fmt.Errorf("exec error: %s", resp.Error)
	}

	return Result{
		lastInsertID: resp.LastInsertID,
		rowsAffected: resp.RowsAffected,
	}, nil
}

// Get executes a query and scans the first row into dest
func (db *DB) Get(dest any, query string, args ...any) error {
	rows, err := db.query(query, args...)
	if err != nil {
		return err
	}

	if len(rows.Rows) == 0 {
		return fmt.Errorf("no rows in result set")
	}

	return scanRow(dest, rows.Columns, rows.Rows[0])
}

// Select executes a query and scans all rows into dest (which should be a slice)
func (db *DB) Select(dest any, query string, args ...any) error {
	rows, err := db.query(query, args...)
	if err != nil {
		return err
	}

	return scanRows(dest, rows.Columns, rows.Rows)
}

// Begin starts a new transaction
func (db *DB) Begin() (*Tx, error) {
	req := types.SQLRequest{Command: "begin_tx"}

	respBytes, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal begin_tx response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("begin_tx error: %s", resp.Error)
	}

	return &Tx{id: resp.TxID, readOnly: false}, nil
}

func (db *DB) UseTx(id string) *Tx {
	return &Tx{id: id, readOnly: false}
}

// Prepare creates a prepared statement
func (db *DB) Prepare(query string) (*Stmt, error) {
	req := types.SQLRequest{
		Command: "prepare",
		SQL:     query,
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prepare response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("prepare error: %s", resp.Error)
	}

	return &Stmt{id: resp.StmtID}, nil
}

func (db *DB) query(query string, args ...any) (*types.QueryResponse, error) {
	req := types.SQLRequest{
		Command: "query",
		SQL:     query,
		Args:    args,
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	var resp types.QueryResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal query response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("query error: %s", resp.Error)
	}

	return &resp, nil
}

// --- Transaction Methods ---

// Exec executes a query within the transaction
func (tx *Tx) Exec(query string, args ...any) (Result, error) {
	req := types.SQLRequest{
		Command: "exec",
		SQL:     query,
		Args:    args,
		TxID:    tx.id,
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return Result{}, err
	}

	var resp types.ExecResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return Result{}, fmt.Errorf("failed to unmarshal exec response: %w", err)
	}

	if resp.Error != "" {
		return Result{}, fmt.Errorf("exec error: %s", resp.Error)
	}

	return Result{
		lastInsertID: resp.LastInsertID,
		rowsAffected: resp.RowsAffected,
	}, nil
}

// Get executes a query within the transaction and scans the first row into dest
func (tx *Tx) Get(dest any, query string, args ...any) error {
	rows, err := tx.query(query, args...)
	if err != nil {
		return err
	}

	if len(rows.Rows) == 0 {
		return fmt.Errorf("no rows in result set")
	}

	return scanRow(dest, rows.Columns, rows.Rows[0])
}

// Select executes a query within the transaction and scans all rows into dest
func (tx *Tx) Select(dest any, query string, args ...any) error {
	rows, err := tx.query(query, args...)
	if err != nil {
		return err
	}

	return scanRows(dest, rows.Columns, rows.Rows)
}

// Prepare creates a prepared statement within the transaction
func (tx *Tx) Prepare(query string) (*Stmt, error) {
	req := types.SQLRequest{
		Command: "prepare",
		SQL:     query,
		TxID:    tx.id,
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prepare response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("prepare error: %s", resp.Error)
	}

	return &Stmt{id: resp.StmtID, tx: tx}, nil
}

// Commit commits the transaction (only if not read-only)
func (tx *Tx) Commit() error {
	if tx.readOnly {
		return fmt.Errorf("cannot commit read-only transaction")
	}

	req := types.SQLRequest{
		Command: "commit",
		TxID:    tx.id,
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return err
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal commit response: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("commit error: %s", resp.Error)
	}

	return nil
}

// Rollback rolls back the transaction (only if not read-only)
func (tx *Tx) Rollback() error {
	if tx.readOnly {
		return fmt.Errorf("cannot rollback read-only transaction")
	}

	req := types.SQLRequest{
		Command: "rollback",
		TxID:    tx.id,
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return err
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal rollback response: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("rollback error: %s", resp.Error)
	}

	return nil
}

func (tx *Tx) query(query string, args ...any) (*types.QueryResponse, error) {
	req := types.SQLRequest{
		Command: "query",
		SQL:     query,
		Args:    args,
		TxID:    tx.id,
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	var resp types.QueryResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal query response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("query error: %s", resp.Error)
	}

	return &resp, nil
}

// --- Statement Methods ---

// Exec executes the prepared statement
func (stmt *Stmt) Exec(args ...any) (Result, error) {
	req := types.SQLRequest{
		Command: "exec",
		Args:    args,
		StmtID:  stmt.id,
	}

	if stmt.tx != nil {
		req.TxID = stmt.tx.id
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return Result{}, err
	}

	var resp types.ExecResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return Result{}, fmt.Errorf("failed to unmarshal exec response: %w", err)
	}

	if resp.Error != "" {
		return Result{}, fmt.Errorf("exec error: %s", resp.Error)
	}

	return Result{
		lastInsertID: resp.LastInsertID,
		rowsAffected: resp.RowsAffected,
	}, nil
}

// Get executes the prepared statement and scans the first row into dest
func (stmt *Stmt) Get(dest any, args ...any) error {
	rows, err := stmt.query(args...)
	if err != nil {
		return err
	}

	if len(rows.Rows) == 0 {
		return fmt.Errorf("no rows in result set")
	}

	return scanRow(dest, rows.Columns, rows.Rows[0])
}

// Select executes the prepared statement and scans all rows into dest
func (stmt *Stmt) Select(dest any, args ...any) error {
	rows, err := stmt.query(args...)
	if err != nil {
		return err
	}

	return scanRows(dest, rows.Columns, rows.Rows)
}

// Close closes the prepared statement
func (stmt *Stmt) Close() error {
	req := types.SQLRequest{
		Command: "close_stmt",
		StmtID:  stmt.id,
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return err
	}

	var resp types.GeneralResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal close_stmt response: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("close_stmt error: %s", resp.Error)
	}

	return nil
}

func (stmt *Stmt) query(args ...any) (*types.QueryResponse, error) {
	req := types.SQLRequest{
		Command: "query",
		Args:    args,
		StmtID:  stmt.id,
	}

	if stmt.tx != nil {
		req.TxID = stmt.tx.id
	}

	respBytes, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	var resp types.QueryResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal query response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("query error: %s", resp.Error)
	}

	return &resp, nil
}

// --- Scanning Functions ---

// scanRow scans a single row into dest
func scanRow(dest any, columns []string, row []any) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	destValue = destValue.Elem()
	destType := destValue.Type()

	switch destType.Kind() {
	case reflect.Struct:
		return scanIntoStruct(destValue, destType, columns, row)
	case reflect.Map:
		return scanIntoMap(destValue, columns, row)
	default:
		// Single value scanning
		if len(row) != 1 {
			return fmt.Errorf("cannot scan %d columns into single value", len(row))
		}
		return scanValue(destValue, row[0])
	}
}

// scanRows scans multiple rows into dest (which should be a slice)
func scanRows(dest any, columns []string, rows [][]any) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	destValue = destValue.Elem()
	if destValue.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	elemType := destValue.Type().Elem()
	slice := reflect.MakeSlice(destValue.Type(), 0, len(rows))

	for _, row := range rows {
		elem := reflect.New(elemType).Elem()

		switch elemType.Kind() {
		case reflect.Struct:
			if err := scanIntoStruct(elem, elemType, columns, row); err != nil {
				return err
			}
		case reflect.Map:
			if err := scanIntoMap(elem, columns, row); err != nil {
				return err
			}
		default:
			if len(row) != 1 {
				return fmt.Errorf("cannot scan %d columns into single value", len(row))
			}
			if err := scanValue(elem, row[0]); err != nil {
				return err
			}
		}

		slice = reflect.Append(slice, elem)
	}

	destValue.Set(slice)
	return nil
}

// scanIntoStruct scans a row into a struct
func scanIntoStruct(destValue reflect.Value, destType reflect.Type, columns []string, row []any) error {
	columnMap := make(map[string]int)
	for i, col := range columns {
		columnMap[strings.ToLower(col)] = i
	}

	for i := range destType.NumField() {
		field := destType.Field(i)
		fieldValue := destValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Get field name from db tag or use field name
		dbName := field.Tag.Get("db")
		if dbName == "" {
			dbName = strings.ToLower(field.Name)
		}

		if dbName == "-" {
			continue
		}

		colIndex, exists := columnMap[strings.ToLower(dbName)]
		if !exists {
			continue
		}

		if err := scanValue(fieldValue, row[colIndex]); err != nil {
			return fmt.Errorf("error scanning field %s: %w", field.Name, err)
		}
	}

	return nil
}

// scanIntoMap scans a row into a map
func scanIntoMap(destValue reflect.Value, columns []string, row []any) error {
	if destValue.IsNil() {
		destValue.Set(reflect.MakeMap(destValue.Type()))
	}

	for i, col := range columns {
		key := reflect.ValueOf(col)
		val := reflect.ValueOf(row[i])
		destValue.SetMapIndex(key, val)
	}

	return nil
}

// scanValue scans a single value into a reflect.Value
func scanValue(dest reflect.Value, src any) error {
	if src == nil {
		return nil
	}

	srcValue := reflect.ValueOf(src)

	// Handle string-encoded values that need conversion
	if srcStr, ok := src.(string); ok {
		switch dest.Kind() {
		case reflect.String:
			dest.SetString(srcStr)
			return nil
		case reflect.Slice:
			if dest.Type().Elem().Kind() == reflect.Uint8 {
				// Decode base64 for []byte
				decoded, err := base64.StdEncoding.DecodeString(srcStr)
				if err != nil {
					return fmt.Errorf("failed to decode base64: %w", err)
				}
				dest.SetBytes(decoded)
				return nil
			}
		case reflect.Struct:
			if dest.Type() == reflect.TypeOf(time.Time{}) {
				t, err := time.Parse(time.RFC3339Nano, srcStr)
				if err != nil {
					return fmt.Errorf("failed to parse time: %w", err)
				}
				dest.Set(reflect.ValueOf(t))
				return nil
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := strconv.ParseInt(srcStr, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int: %w", err)
			}
			dest.SetInt(i)
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u, err := strconv.ParseUint(srcStr, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse uint: %w", err)
			}
			dest.SetUint(u)
			return nil
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(srcStr, 64)
			if err != nil {
				return fmt.Errorf("failed to parse float: %w", err)
			}
			dest.SetFloat(f)
			return nil
		case reflect.Bool:
			b, err := strconv.ParseBool(srcStr)
			if err != nil {
				return fmt.Errorf("failed to parse bool: %w", err)
			}
			dest.SetBool(b)
			return nil
		}
	}

	// Direct assignment for compatible types
	if srcValue.Type().ConvertibleTo(dest.Type()) {
		dest.Set(srcValue.Convert(dest.Type()))
		return nil
	}

	return fmt.Errorf("cannot scan %T into %s", src, dest.Type())
}
