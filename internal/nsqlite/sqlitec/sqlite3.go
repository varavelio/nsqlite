// Package sqlitec provides a lightweight wrapper for the SQLite C library.
// It allows direct interaction with SQLite's low-level API.
//
//   - https://www.sqlite.org/cintro.html
//   - https://www.sqlite.org/c3ref/intro.html
package sqlitec

// #include "sqlite3.c"
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

// getResCodeStr returns the string representation of the SQLite result code
// in format "code: description".
//
// https://www.sqlite.org/c3ref/errcode.html
func getResCodeStr(resCode C.int) string {
	return fmt.Sprintf("%v: %s", resCode, C.GoString(C.sqlite3_errstr(resCode)))
}

// Conn represents a high-level connection to a SQLite database.
//
// https://www.sqlite.org/c3ref/sqlite3.html
type Conn struct {
	cDB *C.sqlite3
}

// Stmt represents a prepared statement in SQLite.
//
// https://www.sqlite.org/c3ref/stmt.html
type Stmt struct {
	conn  *Conn
	cStmt *C.sqlite3_stmt
}

// getLastError returns the last error message from the SQLite database.
func (conn *Conn) getLastError() error {
	if conn.cDB == nil {
		return errors.New("failed to get last error: database connection is nil")
	}
	return errors.New(C.GoString(C.sqlite3_errmsg(conn.cDB)))
}

// Open opens a new SQLite database connection using the given path.
//
// https://www.sqlite.org/c3ref/open.html
func Open(filePath string) (*Conn, error) {
	cFilePath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cFilePath))

	var db *C.sqlite3
	resCode := C.sqlite3_open(cFilePath, &db)
	if resCode != C.SQLITE_OK {
		errMsg := (&Conn{cDB: db}).getLastError()
		_ = C.sqlite3_close(db)
		return nil, fmt.Errorf("failed to open database: %s: %s", getResCodeStr(resCode), errMsg)
	}

	return &Conn{cDB: db}, nil
}

// Close finalizes the connection to the SQLite database.
//
// https://www.sqlite.org/c3ref/close.html
func (conn *Conn) Close() error {
	if conn.cDB == nil {
		return nil
	}

	// The sqlite3_close_v2() interface is intended for use with host
	// languages that are garbage collected, and where the order in which
	// destructors are called is arbitrary.
	resCode := C.sqlite3_close_v2(conn.cDB)
	if resCode != C.SQLITE_OK {
		return fmt.Errorf(
			"failed to close database: %s: %s",
			getResCodeStr(resCode),
			conn.getLastError(),
		)
	}
	conn.cDB = nil

	return nil
}

// LastInsertRowID returns the row ID of the most recent successful INSERT
// into the database from the current connection.
//
// https://www.sqlite.org/c3ref/last_insert_rowid.html
func (conn *Conn) LastInsertRowID() int64 {
	return int64(C.sqlite3_last_insert_rowid(conn.cDB))
}

// RowsAffected returns the number of rows modified, inserted, or deleted by
// the most recent successful INSERT, UPDATE, or DELETE statement from the
// current connection.
//
// https://www.sqlite.org/c3ref/changes.html
func (conn *Conn) RowsAffected() int64 {
	return int64(C.sqlite3_changes(conn.cDB))
}

// QueryParam represents a named (?NNN, :VVV, @VVV, $VVV) or nameless (?) parameter in a SQL query.
type QueryParam struct {
	Name  string `json:"name,omitempty"`
	Value any    `json:"value"`
}

// QueryResult represents the result for Query.
type QueryResult struct {
	LastInsertID int64
	RowsAffected int64
	Columns      []string
	Types        []string
	Rows         [][]any
}

// Query executes the given SQL query on the SQLite database connection
// from start to finish, returning the result of the query for both
// write and read operations.
func (conn *Conn) Query(query string, parameters []QueryParam) (*QueryResult, error) {
	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer func() {
		_ = stmt.Finalize()
	}()

	var lastInsertID, rowsAffected int64
	var columns []string
	var types []string
	var rows [][]any
	columnCount := stmt.ColumnCount()

	for i, param := range parameters {
		if param.Name == "" {
			if err := stmt.BindDynamic(i+1, param.Value); err != nil {
				return nil, fmt.Errorf("failed to bind nameless parameter: %w", err)
			}
		}

		if param.Name != "" {
			index := stmt.BindParameterIndexSafe(param.Name)
			if index == 0 {
				return nil, fmt.Errorf("failed to find named parameter index: %s", param.Name)
			}
			if err := stmt.BindDynamic(index, param.Value); err != nil {
				return nil, fmt.Errorf("failed to bind named parameter: %w", err)
			}
		}
	}

	if columnCount == 0 {
		hasNext := true
		err = nil
		for {
			hasNext, err = stmt.Step()
			if err != nil {
				return nil, fmt.Errorf("failed to step statement: %w", err)
			}
			if !hasNext {
				break
			}
		}

		lastInsertID = conn.LastInsertRowID()
		rowsAffected = conn.RowsAffected()
	}

	if columnCount > 0 {
		columns = make([]string, columnCount)
		types = make([]string, columnCount)
		rows = make([][]any, 0)

		for i := 0; i < columnCount; i++ {
			columns[i] = stmt.ColumnName(i)
			types[i] = stmt.ColumnDecltype(i)
		}

		isFirstIter := true
		hasNext := true
		err = nil
		for {
			hasNext, err = stmt.Step()
			if err != nil {
				return nil, fmt.Errorf("failed to step statement: %w", err)
			}
			if !hasNext {
				break
			}

			row := make([]any, columnCount)
			for i := 0; i < columnCount; i++ {
				col, err := stmt.ColumnDynamic(i)
				if err != nil {
					return nil, fmt.Errorf("failed to get column value: %w", err)
				}
				row[i] = col

				if isFirstIter && types[i] == "" {
					types[i] = stmt.ColumnValueType(col)
				}
			}

			isFirstIter = false
			rows = append(rows, row)
		}
	}

	return &QueryResult{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
		Columns:      columns,
		Types:        types,
		Rows:         rows,
	}, nil
}

// Prepare compiles the given SQL query into a prepared statement.
//
// https://www.sqlite.org/c3ref/prepare.html
func (conn *Conn) Prepare(query string) (*Stmt, error) {
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	var cStmt *C.sqlite3_stmt
	resCode := C.sqlite3_prepare_v2(conn.cDB, cQuery, C.int(-1), &cStmt, nil)
	if resCode != C.SQLITE_OK {
		return nil, fmt.Errorf(
			"failed to prepare statement: %s: %s",
			getResCodeStr(resCode),
			conn.getLastError(),
		)
	}

	return &Stmt{conn: conn, cStmt: cStmt}, nil
}

// ReadOnly returns true if the given SQL query is read-only.
//
// https://www.sqlite.org/c3ref/stmt_readonly.html
func (stmt *Stmt) ReadOnly() bool {
	return C.sqlite3_stmt_readonly(stmt.cStmt) != 0
}

// BindParameterCount returns the number of parameters in the prepared statement.
//
// https://www.sqlite.org/c3ref/bind_parameter_count.html
func (stmt *Stmt) BindParameterCount() int {
	return int(C.sqlite3_bind_parameter_count(stmt.cStmt))
}

// BindParameterName returns the name of the parameter at the given index.
//
// https://www.sqlite.org/c3ref/bind_parameter_name.html
func (stmt *Stmt) BindParameterName(index int) string {
	return C.GoString(C.sqlite3_bind_parameter_name(stmt.cStmt, C.int(index)))
}

// BindParameterIndex returns the index of the parameter with the given name.
//
// https://www.sqlite.org/c3ref/bind_parameter_index.html
func (stmt *Stmt) BindParameterIndex(name string) int {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	return int(C.sqlite3_bind_parameter_index(stmt.cStmt, cName))
}

// BindParameterIndexSafe tries to find the index of the parameter with the given name
// using all prefixes (?, :, @, $) if no one is provided.
func (stmt *Stmt) BindParameterIndexSafe(name string) int {
	if name == "" {
		return 0
	}

	prefixes := []string{":", "@", "$", "?"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return stmt.BindParameterIndex(name)
		}
	}

	for _, prefix := range prefixes {
		index := stmt.BindParameterIndex(prefix + name)
		if index != 0 {
			return index
		}
	}

	return 0
}

// BindDynamic binds a parameter at the given index depending on the type of the value.
func (stmt *Stmt) BindDynamic(index int, value any) error {
	switch v := value.(type) {
	case bool:
		if v {
			return stmt.BindInt(index, 1)
		}
		return stmt.BindInt(index, 0)
	case int8:
		return stmt.BindInt(index, int32(v))
	case uint8:
		return stmt.BindInt(index, int32(v))
	case int16:
		return stmt.BindInt(index, int32(v))
	case uint16:
		return stmt.BindInt(index, int32(v))
	case int32:
		return stmt.BindInt(index, v)
	case uint32:
		return stmt.BindInt(index, int32(v))
	case int:
		return stmt.BindInt64(index, int64(v))
	case uint:
		return stmt.BindInt64(index, int64(v))
	case int64:
		return stmt.BindInt64(index, v)
	case uint64:
		return stmt.BindInt64(index, int64(v))
	case float64:
		return stmt.BindDouble(index, v)
	case float32:
		return stmt.BindDouble(index, float64(v))
	case string:
		return stmt.BindText(index, v)
	case []byte:
		return stmt.BindBlob(index, v)
	case nil:
		return stmt.BindNull(index)
	default:
		return fmt.Errorf("unsupported bind %T type: %v", value, value)
	}
}

// BindInt binds an int parameter at the given index.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindInt(index int, value int32) error {
	if stmt.cStmt == nil {
		return fmt.Errorf("cannot bind to a nil statement")
	}

	resCode := C.sqlite3_bind_int(stmt.cStmt, C.int(index), C.int(value))
	if resCode != C.SQLITE_OK {
		return fmt.Errorf("failed to bind int: %s", getResCodeStr(resCode))
	}
	return nil
}

// BindInt64 binds an int64 parameter at the given index.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindInt64(index int, value int64) error {
	if stmt.cStmt == nil {
		return fmt.Errorf("cannot bind to a nil statement")
	}

	resCode := C.sqlite3_bind_int64(stmt.cStmt, C.int(index), C.sqlite3_int64(value))
	if resCode != C.SQLITE_OK {
		return fmt.Errorf("failed to bind int64: %s", getResCodeStr(resCode))
	}
	return nil
}

// BindDouble binds a float64 parameter at the given index.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindDouble(index int, value float64) error {
	if stmt.cStmt == nil {
		return fmt.Errorf("cannot bind to a nil statement")
	}

	resCode := C.sqlite3_bind_double(stmt.cStmt, C.int(index), C.double(value))
	if resCode != C.SQLITE_OK {
		return fmt.Errorf("failed to bind float64: %s", getResCodeStr(resCode))
	}
	return nil
}

// BindText binds a string parameter at the given index.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindText(index int, value string) error {
	if stmt.cStmt == nil {
		return fmt.Errorf("cannot bind to a nil statement")
	}
	cStr := C.CString(value)
	defer C.free(unsafe.Pointer(cStr))

	resCode := C.cust_sqlite3_bind_text(stmt.cStmt, C.int(index), cStr, C.int(-1))
	if resCode != C.SQLITE_OK {
		return fmt.Errorf("failed to bind text: %s", getResCodeStr(resCode))
	}
	return nil
}

// BindBlob binds a byte slice parameter at the given index.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindBlob(index int, data []byte) error {
	if stmt.cStmt == nil {
		return fmt.Errorf("cannot bind to a nil statement")
	}
	if len(data) == 0 {
		return stmt.BindNull(index)
	}

	resCode := C.cust_sqlite3_bind_blob(
		stmt.cStmt,
		C.int(index),
		unsafe.Pointer(&data[0]),
		C.int(len(data)),
	)
	if resCode != C.SQLITE_OK {
		return fmt.Errorf("failed to bind blob: %s", getResCodeStr(resCode))
	}
	return nil
}

// BindNull binds a NULL value at the given index.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindNull(index int) error {
	if stmt.cStmt == nil {
		return fmt.Errorf("cannot bind to a nil statement")
	}

	resCode := C.sqlite3_bind_null(stmt.cStmt, C.int(index))
	if resCode != C.SQLITE_OK {
		return fmt.Errorf("failed to bind null: %s", getResCodeStr(resCode))
	}
	return nil
}

// Step advances the statement to the next row of data, returning true if a new row
// is available, or false if there are no more rows. If an error occurs, it is returned.
//
// https://www.sqlite.org/c3ref/step.html
func (stmt *Stmt) Step() (bool, error) {
	resCode := C.sqlite3_step(stmt.cStmt)

	if resCode == C.SQLITE_DONE {
		return false, nil
	}

	if resCode == C.SQLITE_ROW {
		return true, nil
	}

	return false, fmt.Errorf("failed to step statement: %s", getResCodeStr(resCode))
}

// ColumnCount returns the number of columns in the current result row.
//
// https://www.sqlite.org/c3ref/column_count.html
func (stmt *Stmt) ColumnCount() int {
	return int(C.sqlite3_column_count(stmt.cStmt))
}

// ColumnName returns the name of the column at the given index.
//
// https://www.sqlite.org/c3ref/column_name.html
func (stmt *Stmt) ColumnName(colIndex int) string {
	return C.GoString(C.sqlite3_column_name(stmt.cStmt, C.int(colIndex)))
}

// ColumnNames returns the names of all columns in the current result row.
func (stmt *Stmt) ColumnNames() []string {
	count := stmt.ColumnCount()
	if count == 0 {
		return nil
	}

	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = stmt.ColumnName(i)
	}

	return names
}

// ColumnDecltype returns the declared type of the column at the given index.
//
// https://www.sqlite.org/c3ref/column_decltype.html
func (stmt *Stmt) ColumnDecltype(colIndex int) string {
	return strings.ToUpper(C.GoString(C.sqlite3_column_decltype(stmt.cStmt, C.int(colIndex))))
}

// ColumnValueType returns the inferred type of the given value.
// It returns one of the five storage classes of SQLite
//
// https://www.sqlite.org/datatype3.html#storage_classes_and_datatypes
func (stmt *Stmt) ColumnValueType(value any) string {
	switch value.(type) {
	case nil:
		return "NULL"
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "INTEGER"
	case float32, float64:
		return "REAL"
	case string:
		return "TEXT"
	case []byte:
		return "BLOB"
	default:
		return ""
	}
}

type ColumnType int

const (
	ColumnTypeInteger = ColumnType(C.SQLITE_INTEGER)
	ColumnTypeFloat   = ColumnType(C.SQLITE_FLOAT)
	ColumnTypeText    = ColumnType(C.SQLITE_TEXT)
	ColumnTypeBlob    = ColumnType(C.SQLITE_BLOB)
	ColumnTypeNull    = ColumnType(C.SQLITE_NULL)
)

// ColumnType returns the type of the column at the given index.
// The return value can be used to decide which of interfaces
// should be used to extract the column value.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnType(colIndex int) ColumnType {
	return ColumnType(C.sqlite3_column_type(stmt.cStmt, C.int(colIndex)))
}

// ColumnDynamic returns the column value at the given index depending on the
// type of the column.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnDynamic(colIndex int) (any, error) {
	columnType := stmt.ColumnType(colIndex)
	switch columnType {
	case ColumnTypeInteger:
		return stmt.ColumnInt(colIndex), nil
	case ColumnTypeFloat:
		return stmt.ColumnFloat64(colIndex), nil
	case ColumnTypeText:
		return stmt.ColumnText(colIndex), nil
	case ColumnTypeBlob:
		return stmt.ColumnBlob(colIndex), nil
	case ColumnTypeNull:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported column type: %d", columnType)
	}
}

// ColumnBytes returns the number of bytes in the column value at the given index.
// Useful for extracting the size of a blob or text column.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnBytes(colIndex int) int {
	return int(C.sqlite3_column_bytes(stmt.cStmt, C.int(colIndex)))
}

// ColumnInt returns the column value at the given index as int.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnInt(colIndex int) int {
	return int(C.sqlite3_column_int(stmt.cStmt, C.int(colIndex)))
}

// ColumnInt64 returns the column value at the given index as int64.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnInt64(colIndex int) int64 {
	return int64(C.sqlite3_column_int64(stmt.cStmt, C.int(colIndex)))
}

// ColumnFloat64 returns the column value at the given index as float64.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnFloat64(colIndex int) float64 {
	return float64(C.sqlite3_column_double(stmt.cStmt, C.int(colIndex)))
}

// ColumnText returns the column value at the given index as a string.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnText(colIndex int) string {
	size := C.sqlite3_column_bytes(stmt.cStmt, C.int(colIndex))
	if size <= 0 {
		return ""
	}

	text := (*C.char)(unsafe.Pointer(C.sqlite3_column_text(stmt.cStmt, C.int(colIndex))))
	if text == nil {
		return ""
	}

	return C.GoStringN(text, size)
}

// ColumnBlob returns the column value at the given index as a byte slice.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnBlob(colIndex int) []byte {
	size := C.sqlite3_column_bytes(stmt.cStmt, C.int(colIndex))
	if size <= 0 {
		return nil
	}

	dataPtr := C.sqlite3_column_blob(stmt.cStmt, C.int(colIndex))
	if dataPtr == nil {
		return nil
	}

	return C.GoBytes(dataPtr, size)
}

// Finalize frees the resources associated with this statement.
//
// https://www.sqlite.org/c3ref/finalize.html
func (stmt *Stmt) Finalize() error {
	if stmt.cStmt == nil {
		return nil
	}

	resCode := C.sqlite3_finalize(stmt.cStmt)
	if resCode != C.SQLITE_OK {
		return fmt.Errorf(
			"failed to finalize statement: %s: %s",
			getResCodeStr(resCode),
			C.GoString(C.sqlite3_errmsg(stmt.conn.cDB)),
		)
	}
	stmt.cStmt = nil

	return nil
}
