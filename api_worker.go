//go:build js && wasm

package wasmsqlite

import (
	"fmt"
	"sync"
	"syscall/js"
)

// APIWorker adapts the JavaScript SQLite Worker API to work with our Go driver
type APIWorker struct {
	bridge js.Value
	mu     sync.Mutex
}

// NewAPIWorker creates a new worker API
func NewAPIWorker() (*APIWorker, error) {
	bridge := js.Global().Get("sqliteBridge")
	if bridge.IsUndefined() {
		return nil, fmt.Errorf("sqliteBridge not found - ensure sqlite-bridge.js is loaded")
	}

	_, err := createWorker(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker: %w", err)
	}

	return &APIWorker{
		bridge: bridge,
	}, nil
}

// Init initializes the SQLite bridge
func (b *APIWorker) Init() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	initMethod := b.bridge.Get("init")
	if initMethod.IsUndefined() {
		return fmt.Errorf("sqliteBridge.init method not found")
	}

	// The bridge auto-initializes on load, so we just return success
	return nil
}

// Open opens a database
func (b *APIWorker) Open(filename, vfs string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	fmt.Printf("🔍 Bridge adapter opening database: filename=%s, vfs=%s\n", filename, vfs)

	openMethod := b.bridge.Get("open")
	if openMethod.IsUndefined() {
		return "", fmt.Errorf("sqliteBridge.open method not found")
	}

	fmt.Println("🔍 Found sqliteBridge.open method, calling it...")

	// Call the open method
	result, err := callAsync(openMethod, filename, vfs)
	if err != nil {
		fmt.Printf("❌ Bridge open failed: %v\n", err)
		return "", err
	}

	fmt.Printf("🔍 Bridge open result: %v\n", result)

	// Extract VFS type from result
	vfsType := "unknown"
	if !result.IsUndefined() && !result.Get("vfsType").IsUndefined() {
		vfsType = result.Get("vfsType").String()
		fmt.Printf("✅ VFS type extracted: %s\n", vfsType)
	} else {
		fmt.Printf("⚠️ vfsType not found in result\n")
	}

	return vfsType, nil
}

// Exec executes a SQL statement
func (b *APIWorker) Exec(sql string, params []any) (int, int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	execMethod := b.bridge.Get("exec")
	if execMethod.IsUndefined() {
		return 0, 0, fmt.Errorf("sqliteBridge.exec method not found")
	}

	// Convert params to JavaScript array
	jsParams := js.Global().Get("Array").New()
	for i, param := range params {
		jsParams.SetIndex(i, toJSValue(param))
	}

	result, err := callAsync(execMethod, sql, jsParams)
	if err != nil {
		return 0, 0, err
	}

	// Extract rowsAffected and lastInsertId
	rowsAffected := 0
	lastInsertId := 0

	if !result.IsUndefined() {
		if !result.Get("rowsAffected").IsUndefined() {
			rowsAffected = result.Get("rowsAffected").Int()
		}
		if !result.Get("lastInsertId").IsUndefined() {
			lastInsertId = result.Get("lastInsertId").Int()
		}
	}

	return rowsAffected, lastInsertId, nil
}

// Query executes a query and returns results
func (b *APIWorker) Query(sql string, params []any) ([]string, [][]any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	queryMethod := b.bridge.Get("query")
	if queryMethod.IsUndefined() {
		return nil, nil, fmt.Errorf("sqliteBridge.query method not found")
	}

	// Convert params to JavaScript array
	jsParams := js.Global().Get("Array").New()
	for i, param := range params {
		jsParams.SetIndex(i, toJSValue(param))
	}

	result, err := callAsync(queryMethod, sql, jsParams)
	if err != nil {
		return nil, nil, err
	}

	// Extract columns and rows
	var columns []string
	var rows [][]any

	if !result.IsUndefined() {
		// Get columns
		columnsJS := result.Get("columns")
		if !columnsJS.IsUndefined() && columnsJS.Length() > 0 {
			columns = make([]string, columnsJS.Length())
			for i := 0; i < columnsJS.Length(); i++ {
				columns[i] = columnsJS.Index(i).String()
			}
		}

		// Get rows
		rowsJS := result.Get("rows")
		if !rowsJS.IsUndefined() && rowsJS.Length() > 0 {
			rows = make([][]any, rowsJS.Length())
			for i := 0; i < rowsJS.Length(); i++ {
				rowJS := rowsJS.Index(i)
				if rowJS.Length() > 0 {
					row := make([]any, rowJS.Length())
					for j := 0; j < rowJS.Length(); j++ {
						val := rowJS.Index(j)
						if val.IsNull() {
							row[j] = nil
						} else if val.Type() == js.TypeNumber {
							num := val.Float()
							// If it's a whole number, return as int64 to match SQLite integer types
							if num == float64(int64(num)) {
								row[j] = int64(num)
							} else {
								row[j] = num
							}
						} else if val.Type() == js.TypeString {
							row[j] = val.String()
						} else if val.Type() == js.TypeBoolean {
							row[j] = val.Bool()
						} else {
							row[j] = val.String()
						}
					}
					rows[i] = row
				}
			}
		}
	}

	return columns, rows, nil
}

// Begin starts a transaction
func (b *APIWorker) Begin() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	beginMethod := b.bridge.Get("begin")
	if beginMethod.IsUndefined() {
		return fmt.Errorf("sqliteBridge.begin method not found")
	}

	_, err := callAsync(beginMethod)
	return err
}

// Commit commits a transaction
func (b *APIWorker) Commit() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	commitMethod := b.bridge.Get("commit")
	if commitMethod.IsUndefined() {
		return fmt.Errorf("sqliteBridge.commit method not found")
	}

	_, err := callAsync(commitMethod)
	return err
}

// Rollback rolls back a transaction
func (b *APIWorker) Rollback() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	rollbackMethod := b.bridge.Get("rollback")
	if rollbackMethod.IsUndefined() {
		return fmt.Errorf("sqliteBridge.rollback method not found")
	}

	_, err := callAsync(rollbackMethod)
	return err
}

// Close closes the database connection
func (b *APIWorker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	closeMethod := b.bridge.Get("close")
	if closeMethod.IsUndefined() {
		return fmt.Errorf("sqliteBridge.close method not found")
	}

	_, err := callAsync(closeMethod)
	return err
}

// Dump exports the database as SQL statements
func (b *APIWorker) Dump() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	dumpMethod := b.bridge.Get("dump")
	if dumpMethod.IsUndefined() {
		return "", fmt.Errorf("sqliteBridge.dump method not found")
	}

	result, err := callAsync(dumpMethod)
	if err != nil {
		return "", err
	}

	// Extract dump from result
	if !result.IsUndefined() && !result.IsNull() {
		dump := result.Get("dump")
		if dump.Truthy() {
			return dump.String(), nil
		}
	}

	return "", fmt.Errorf("no dump data received")
}

// Load imports SQL statements to restore the database
func (b *APIWorker) Load(dump string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	loadMethod := b.bridge.Get("load")
	if loadMethod.IsUndefined() {
		return fmt.Errorf("sqliteBridge.load method not found")
	}

	_, err := callAsync(loadMethod, dump)
	return err
}
