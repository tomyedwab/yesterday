package types

// --- JSON structures for host communication ---

// SQLRequest defines the structure for requests sent to the host.
type SQLRequest struct {
	Command string        `json:"command"`
	SQL     string        `json:"sql,omitempty"`
	Args    []interface{} `json:"args,omitempty"` // Processed driver.Value
	StmtID  string        `json:"stmt_id,omitempty"`
	TxID    string        `json:"tx_id,omitempty"`
}

// GeneralResponse is used for commands that don't return rows or specific exec results (e.g., prepare, commit, rollback, close).
type GeneralResponse struct {
	StmtID string `json:"stmt_id,omitempty"` // For 'prepare' command, host returns a statement ID
	TxID   string `json:"tx_id,omitempty"`   // For 'begin_tx' command, host returns a transaction ID
	Error  string `json:"error,omitempty"`
}

// QueryResponse defines the structure for responses from 'query' commands.
type QueryResponse struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"` // Host should ensure types are JSON-compatible (e.g., time.Time as string, []byte as base64)
	Error   string          `json:"error,omitempty"`
}

// ExecResponse defines the structure for responses from 'exec' commands.
type ExecResponse struct {
	LastInsertID int64  `json:"last_insert_id"`
	RowsAffected int64  `json:"rows_affected"`
	Error        string `json:"error,omitempty"`
}
