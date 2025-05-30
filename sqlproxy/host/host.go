package host

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/tomyedwab/yesterday/sqlproxy/types"
)

// SQLHost handles proxy requests for an SQLite database.
// It manages prepared statements and transactions.
type SQLHost struct {
	db    *sql.DB
	stmts map[string]*sql.Stmt
	txs   map[string]*sql.Tx
	mu    sync.Mutex
}

// NewSQLHost creates a new SQLHost instance.
// The provided db must be an active connection to an SQLite database.
func NewSQLHost(db *sql.DB) *SQLHost {
	return &SQLHost{
		db:    db,
		stmts: make(map[string]*sql.Stmt),
		txs:   make(map[string]*sql.Tx),
	}
}

// HandleRequest processes a raw SQL request payload and returns a raw response payload.
// This is the main entry point for host-side SQL proxying logic.
func (h *SQLHost) HandleRequest(requestPayload []byte) ([]byte, error) {
	var req types.SQLRequest
	if err := json.Unmarshal(requestPayload, &req); err != nil {
		return marshalErrorResponse(fmt.Sprintf("failed to unmarshal request: %v", err))
	}

	var responseData interface{}
	var opErr error

	switch req.Command {
	case "prepare":
		responseData, opErr = h.handlePrepare(&req)
	case "query":
		responseData, opErr = h.handleQuery(&req)
	case "exec":
		responseData, opErr = h.handleExec(&req)
	case "begin_tx":
		responseData, opErr = h.handleBeginTx(&req)
	case "commit":
		responseData, opErr = h.handleCommit(&req)
	case "rollback":
		responseData, opErr = h.handleRollback(&req)
	case "close_stmt":
		responseData, opErr = h.handleCloseStmt(&req)
	case "close_conn":
		responseData, opErr = h.handleCloseConn(&req)
	default:
		opErr = fmt.Errorf("unknown command: %s", req.Command)
	}

	if opErr != nil {
		return marshalErrorResponse(opErr.Error())
	}

	return json.Marshal(responseData)
}

func marshalErrorResponse(errMsg string) ([]byte, error) {
	resp := types.GeneralResponse{Error: errMsg}
	payload, err := json.Marshal(resp)
	if err != nil {
		// This is a critical failure: can't even marshal the error response.
		// Return a hardcoded JSON string and the marshalling error.
		return []byte(fmt.Sprintf(`{"error":"critical: failed to marshal error response for: %s"}`, errMsg)),
			fmt.Errorf("failed to marshal error response for '%s': %w", errMsg, err)
	}
	// The error for HandleRequest itself is nil here, as the operational error is packaged in the payload.
	return payload, nil
}

func (h *SQLHost) handlePrepare(req *types.SQLRequest) (types.GeneralResponse, error) {
	h.mu.Lock()
	tx, txExists := h.txs[req.TxID]
	defer h.mu.Unlock()

	var stmt *sql.Stmt
	var err error
	if req.TxID != "" {
		if !txExists {
			return types.GeneralResponse{}, fmt.Errorf("transaction not found for statement execution: %s", req.TxID)
		}
		stmt, err = tx.Prepare(req.SQL)
		if err != nil {
			return types.GeneralResponse{}, fmt.Errorf("prepare failed: %w", err)
		}
	} else {
		stmt, err = h.db.Prepare(req.SQL)
		if err != nil {
			return types.GeneralResponse{}, fmt.Errorf("prepare failed: %w", err)
		}
	}

	stmtID := uuid.NewString()
	h.stmts[stmtID] = stmt
	return types.GeneralResponse{StmtID: stmtID}, nil
}

func (h *SQLHost) handleExec(req *types.SQLRequest) (types.ExecResponse, error) {
	h.mu.Lock()
	tx, txExists := h.txs[req.TxID]
	stmt, stmtExists := h.stmts[req.StmtID]
	h.mu.Unlock() // Unlock early before potentially long DB operations

	var res sql.Result
	var err error

	if req.StmtID != "" {
		if !stmtExists {
			return types.ExecResponse{}, fmt.Errorf("statement not found: %s", req.StmtID)
		}
		if req.TxID != "" {
			if !txExists {
				return types.ExecResponse{}, fmt.Errorf("transaction not found for statement execution: %s", req.TxID)
			}
			txStmt := tx.Stmt(stmt)
			res, err = txStmt.Exec(req.Args...)
			_ = txStmt.Close() // Close the transaction-specific statement
		} else {
			res, err = stmt.Exec(req.Args...)
		}
	} else {
		if req.TxID != "" {
			if !txExists {
				return types.ExecResponse{}, fmt.Errorf("transaction not found for direct execution: %s", req.TxID)
			}
			res, err = tx.Exec(req.SQL, req.Args...)
		} else {
			res, err = h.db.Exec(req.SQL, req.Args...)
		}
	}

	if err != nil {
		return types.ExecResponse{}, fmt.Errorf("exec failed: %w", err)
	}

	lastInsertID, lerr := res.LastInsertId()
	rowsAffected, rerr := res.RowsAffected()

	if lerr != nil {
		// Log or handle error if LastInsertId is not supported/applicable but an error occurred
	}
	if rerr != nil {
		// Log or handle error if RowsAffected is not supported/applicable but an error occurred
	}

	return types.ExecResponse{LastInsertID: lastInsertID, RowsAffected: rowsAffected}, nil
}

func (h *SQLHost) handleQuery(req *types.SQLRequest) (types.QueryResponse, error) {
	h.mu.Lock()
	tx, txExists := h.txs[req.TxID]
	stmt, stmtExists := h.stmts[req.StmtID]
	h.mu.Unlock()

	var rows *sql.Rows
	var err error

	if req.StmtID != "" {
		if !stmtExists {
			return types.QueryResponse{}, fmt.Errorf("statement not found: %s", req.StmtID)
		}
		if req.TxID != "" {
			if !txExists {
				return types.QueryResponse{}, fmt.Errorf("transaction not found for statement query: %s", req.TxID)
			}
			txStmt := tx.Stmt(stmt)
			rows, err = txStmt.Query(req.Args...)
			_ = txStmt.Close()
		} else {
			rows, err = stmt.Query(req.Args...)
		}
	} else {
		if req.TxID != "" {
			if !txExists {
				return types.QueryResponse{}, fmt.Errorf("transaction not found for direct query: %s", req.TxID)
			}
			rows, err = tx.Query(req.SQL, req.Args...)
		} else {
			rows, err = h.db.Query(req.SQL, req.Args...)
		}
	}

	if err != nil {
		return types.QueryResponse{}, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return types.QueryResponse{}, fmt.Errorf("failed to get columns: %w", err)
	}

	var results [][]interface{}
	scanArgs := make([]interface{}, len(columns))
	scanPtrs := make([]interface{}, len(columns))
	for i := range scanArgs {
		scanPtrs[i] = &scanArgs[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			return types.QueryResponse{}, fmt.Errorf("failed to scan row: %w", err)
		}
		processedRow, pErr := processRowValues(scanArgs)
		if pErr != nil {
			return types.QueryResponse{}, fmt.Errorf("failed to process row values: %w", pErr)
		}
		results = append(results, processedRow)
	}

	if err := rows.Err(); err != nil {
		return types.QueryResponse{}, fmt.Errorf("error iterating rows: %w", err)
	}

	return types.QueryResponse{Columns: columns, Rows: results}, nil
}

func processRowValues(rawRow []interface{}) ([]interface{}, error) {
	processedRow := make([]interface{}, len(rawRow))
	for i, val := range rawRow {
		if val == nil {
			processedRow[i] = nil
			continue
		}
		switch v := val.(type) {
		case []byte:
			processedRow[i] = base64.StdEncoding.EncodeToString(v)
		case time.Time:
			processedRow[i] = v.Format(time.RFC3339Nano)
		default:
			processedRow[i] = v
		}
	}
	return processedRow, nil
}

func (h *SQLHost) RegisterTx(tx *sql.Tx) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	txID := uuid.NewString()
	h.txs[txID] = tx
	return txID
}

func (h *SQLHost) handleBeginTx(req *types.SQLRequest) (types.GeneralResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Note: The spec mentions TxOptions for BeginTx, but the driver implements Begin().
	// For simplicity, using Begin() without options.
	tx, err := h.db.Begin()
	if err != nil {
		return types.GeneralResponse{}, fmt.Errorf("begin transaction failed: %w", err)
	}

	txID := uuid.NewString()
	h.txs[txID] = tx
	return types.GeneralResponse{TxID: txID}, nil
}

func (h *SQLHost) handleCommit(req *types.SQLRequest) (types.GeneralResponse, error) {
	h.mu.Lock()
	tx, exists := h.txs[req.TxID]
	if exists {
		delete(h.txs, req.TxID)
	}
	h.mu.Unlock()

	if !exists {
		return types.GeneralResponse{}, fmt.Errorf("transaction not found or already closed: %s", req.TxID)
	}

	if err := tx.Commit(); err != nil {
		return types.GeneralResponse{}, fmt.Errorf("commit failed: %w", err)
	}
	return types.GeneralResponse{}, nil
}

func (h *SQLHost) handleRollback(req *types.SQLRequest) (types.GeneralResponse, error) {
	h.mu.Lock()
	tx, exists := h.txs[req.TxID]
	if exists {
		delete(h.txs, req.TxID)
	}
	h.mu.Unlock()

	if !exists {
		return types.GeneralResponse{}, fmt.Errorf("transaction not found or already closed: %s", req.TxID)
	}

	if err := tx.Rollback(); err != nil {
		return types.GeneralResponse{}, fmt.Errorf("rollback failed: %w", err)
	}
	return types.GeneralResponse{}, nil
}

func (h *SQLHost) handleCloseStmt(req *types.SQLRequest) (types.GeneralResponse, error) {
	h.mu.Lock()
	stmt, exists := h.stmts[req.StmtID]
	if exists {
		delete(h.stmts, req.StmtID)
	}
	h.mu.Unlock()

	if !exists {
		// Closing a non-existent or already closed statement is often not an error for clients.
		// However, for strictness or debugging, we can indicate it.
		// For now, let's return success as per typical driver behavior (idempotent close).
		return types.GeneralResponse{}, nil
	}

	if err := stmt.Close(); err != nil {
		return types.GeneralResponse{}, fmt.Errorf("close statement failed: %w", err)
	}
	return types.GeneralResponse{}, nil
}

func (h *SQLHost) handleCloseConn(req *types.SQLRequest) (types.GeneralResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all open statements associated with this host instance
	for id, stmt := range h.stmts {
		_ = stmt.Close() // Ignore error, best effort
		delete(h.stmts, id)
	}

	// Rollback any pending transactions associated with this host instance
	for id, tx := range h.txs {
		_ = tx.Rollback() // Ignore error, best effort
		delete(h.txs, id)
	}

	// The underlying h.db is managed externally, so we don't close it here.
	// This command effectively resets the host's internal state.
	return types.GeneralResponse{}, nil
}
