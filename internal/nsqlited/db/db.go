// Package db provides the SQLite integration for NSQLite.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nsqlite/nsqlite/internal/nsqlited/log"
	"github.com/nsqlite/nsqlite/internal/nsqlited/sqlitec"
	"github.com/nsqlite/nsqlite/internal/nsqlited/sqlitedrv"
	"github.com/nsqlite/nsqlite/internal/nsqlited/stats"
	"github.com/nsqlite/nsqlite/internal/util/syncutil"
	"github.com/orsinium-labs/enum"
)

var (
	ErrTxNotFound   = errors.New("transaction not found or timed out, check your settings")
	ErrTxNotMatch   = errors.New("transaction ID does not match the currently active transaction")
	ErrTxWithinTx   = errors.New("cannot start a transaction within a transaction")
	ErrTxIdRequired = errors.New("transaction ID is required for this operation")
)

// Config represents the configuration for a DB instance.
type Config struct {
	// Logger is the shared NSQLite logger.
	Logger log.Logger
	// DBStats is an instance of dbstats.DBStats.
	DBStats *stats.DBStats
	// DataDir is the directory where the database files are stored.
	DataDir string
	// TxIdleTimeout if a transaction is not active for this duration, it
	// will be rolled back.
	TxIdleTimeout time.Duration
}

// DB represents the SQLite integration for NSQLite.
type DB struct {
	Config
	isInitialized     bool
	readWritePool     *sql.DB
	readOnlyPool      *sql.DB
	txId              syncutil.AtomicString
	txIdLastUsed      syncutil.AtomicTime
	txIdleMonitorStop chan any
	txMu              sync.Mutex
	writeMu           sync.Mutex
	closeWg           sync.WaitGroup
}

// Query represents a query to be executed.
type Query struct {
	TxID   string
	Query  string
	Params []sqlitec.QueryParam
}

// queryType represents the type of a given SQLite query.
type queryType enum.Member[string]

var (
	QueryTypeUnknown  = queryType{Value: "unknown"}
	QueryTypeRead     = queryType{Value: "read"}
	QueryTypeWrite    = queryType{Value: "write"}
	QueryTypeBegin    = queryType{Value: "begin"}
	QueryTypeCommit   = queryType{Value: "commit"}
	QueryTypeRollback = queryType{Value: "rollback"}
)

// QueryResult represents the result of a query.
type QueryResult struct {
	// For all queries
	Type queryType

	// For begin queries
	TxID string

	// For write queries
	LastInsertID int64
	RowsAffected int64

	// For read and write queries that return rows
	Columns []string
	Types   []string
	Rows    [][]any
}

// NewDB creates a new DB instance.
func NewDB(config Config) (*DB, error) {
	if !config.Logger.IsInitialized() {
		return nil, errors.New("logger is required")
	}
	if config.DataDir == "" {
		return nil, errors.New("database directory is required")
	}
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	if config.TxIdleTimeout <= 0 {
		return nil, errors.New("transaction idle timeout must be provided")
	}

	databasePath := path.Join(config.DataDir, "database.sqlite")
	readWriteConnector := newConnector(databasePath, false)
	readOnlyConnector := newConnector(databasePath, true)

	readWritePool := sql.OpenDB(readWriteConnector)
	if err := readWritePool.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping write connection: %w", err)
	}
	readWritePool.SetConnMaxIdleTime(0)
	readWritePool.SetConnMaxLifetime(0)
	readWritePool.SetMaxIdleConns(1)
	readWritePool.SetMaxOpenConns(1)

	readOnlyPool := sql.OpenDB(readOnlyConnector)
	if err := readOnlyPool.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping read connection: %w", err)
	}
	readOnlyPool.SetConnMaxIdleTime(0)
	readOnlyPool.SetConnMaxLifetime(0)
	readOnlyPool.SetMaxIdleConns(100)

	db := &DB{
		Config:            config,
		isInitialized:     true,
		readWritePool:     readWritePool,
		readOnlyPool:      readOnlyPool,
		txId:              *syncutil.NewAtomicString(""),
		txIdLastUsed:      *syncutil.NewAtomicTime(time.Now()),
		txIdleMonitorStop: make(chan any),
		txMu:              sync.Mutex{},
		writeMu:           sync.Mutex{},
		closeWg:           sync.WaitGroup{},
	}

	db.closeWg.Add(1)
	go db.txIdleMonitor(config.TxIdleTimeout)

	config.Logger.InfoNs(log.NsDatabase, "database started")
	return db, nil
}

// getRawConn returns a raw connection from *sql.DB and a function to return
// it to the pool.
func (db *DB) getRawConn(ctx context.Context, dbPool *sql.DB) (*sqlitec.Conn, func() error, error) {
	poolConn, err := dbPool.Conn(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get connection from pool: %w", err)
	}

	var rawConn *sqlitec.Conn
	err = poolConn.Raw(func(driverConn any) error {
		dc, ok := driverConn.(*sqlitedrv.Conn)
		if !ok {
			return fmt.Errorf("failed to cast driver connection")
		}
		rawConn = dc.RawConn()
		return nil
	})
	if err != nil {
		poolConn.Close()
		return nil, nil, fmt.Errorf("failed to get raw connection: %w", err)
	}

	return rawConn, poolConn.Close, nil
}

// getReadWriteRawConn returns the read-write connection and a function to
// return it to the pool.
func (db *DB) getReadWriteRawConn(ctx context.Context) (*sqlitec.Conn, func() error, error) {
	return db.getRawConn(ctx, db.readWritePool)
}

// getReadOnlyRawConn returns the read-only connection and a function to return it
// to the pool.
func (db *DB) getReadOnlyRawConn(ctx context.Context) (*sqlitec.Conn, func() error, error) {
	return db.getRawConn(ctx, db.readOnlyPool)
}

// IsInitialized returns whether the DB instance is initialized.
func (db *DB) IsInitialized() bool {
	return db.isInitialized
}

// txIdleMonitor rolls back the current transaction if not used within the timeout.
func (db *DB) txIdleMonitor(timeout time.Duration) {
	defer db.closeWg.Done()
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case <-db.txIdleMonitorStop:
			return
		case <-ticker.C:
			txId := db.txId.Load()
			if txId == "" {
				continue
			}
			if time.Since(db.txIdLastUsed.Load()) > timeout {
				_, _ = db.executeRollbackQuery(context.Background(), txId)
				db.Logger.InfoNs(log.NsDatabase, "transaction rolled back due to idle timeout", log.KV{
					"txId":    txId,
					"timeout": timeout.String(),
				})
			}
		}
	}
}

// Close attempts a graceful shutdown of everything this DB manages.
func (db *DB) Close() error {
	close(db.txIdleMonitorStop)
	db.closeWg.Wait()

	txId := db.txId.Load()
	if txId != "" {
		_, _ = db.executeRollbackQuery(context.Background(), txId)
	}

	if db.readWritePool != nil {
		if err := db.readWritePool.Close(); err != nil {
			return fmt.Errorf("failed to close write connection: %w", err)
		}
	}

	if db.readOnlyPool != nil {
		if err := db.readOnlyPool.Close(); err != nil {
			return fmt.Errorf("failed to close read connections: %w", err)
		}
	}

	return nil
}

// detectQueryType detects the type of query between read, write, begin, commit,
// and rollback.
func (db *DB) detectQueryType(ctx context.Context, query string) (queryType, error) {
	trimmed := strings.ToLower(strings.TrimSpace(query))

	switch {
	case strings.HasPrefix(trimmed, "begin"):
		return QueryTypeBegin, nil
	case strings.HasPrefix(trimmed, "commit"):
		return QueryTypeCommit, nil
	case strings.HasPrefix(trimmed, "rollback"), strings.HasPrefix(trimmed, "end transaction"):
		return QueryTypeRollback, nil
	}

	conn, returnConn, err := db.getReadOnlyRawConn(ctx)
	if err != nil {
		return QueryTypeUnknown, fmt.Errorf("failed to get connection: %w", err)
	}
	defer func() { _ = returnConn() }()

	stmt, err := conn.Prepare(query)
	if err != nil {
		return QueryTypeUnknown, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Finalize() }()

	if stmt.ReadOnly() {
		return QueryTypeRead, nil
	}
	return QueryTypeWrite, nil
}

// Query executes an SQLite query.
func (db *DB) Query(ctx context.Context, query Query) (QueryResult, error) {
	res, err := db.query(ctx, query)
	if err != nil {
		db.DBStats.IncErrors()
	}

	return res, err
}

// query is the underlying logic for Query.
func (db *DB) query(ctx context.Context, query Query) (QueryResult, error) {
	typeOfQuery, err := db.detectQueryType(ctx, query.Query)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to detect query type: %w", err)
	}

	if query.TxID != "" {
		currentTxID := db.txId.Load()
		if currentTxID == "" {
			return QueryResult{}, ErrTxNotFound
		}
		if query.TxID != currentTxID {
			return QueryResult{}, ErrTxNotMatch
		}
		db.txIdLastUsed.Store(time.Now())
	}

	switch typeOfQuery {
	case QueryTypeBegin:
		return db.executeBeginQuery(ctx, query.TxID)
	case QueryTypeCommit:
		return db.executeCommitQuery(ctx, query.TxID)
	case QueryTypeRollback:
		return db.executeRollbackQuery(ctx, query.TxID)
	case QueryTypeRead:
		return db.executeReadQuery(ctx, query)
	case QueryTypeWrite:
		return db.executeWriteQuery(ctx, query)
	}

	return QueryResult{}, fmt.Errorf("unknown query type: %s", typeOfQuery.Value)
}

// executeBeginQuery executes a begin query using the read-write connection.
func (db *DB) executeBeginQuery(ctx context.Context, queryTxID string) (QueryResult, error) {
	if queryTxID != "" {
		return QueryResult{}, ErrTxWithinTx
	}

	db.DBStats.IncQueuedBegins()
	defer db.DBStats.DecQueuedBegins()

	// We need to lock the transaction mutex to ensure that we don't start
	// a new transaction while another transaction is in progress.
	//
	// The unlock is done either in the commit or rollback functions.
	db.txMu.Lock()

	conn, returnConn, err := db.getReadWriteRawConn(ctx)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to get read-write connection from pool: %w", err)
	}
	defer func() { _ = returnConn() }()

	if _, err = conn.Query("BEGIN TRANSACTION", nil); err != nil {
		return QueryResult{}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	txId := uuid.NewString()
	db.txId.Store(txId)
	db.txIdLastUsed.Store(time.Now())
	db.DBStats.IncBegins()

	return QueryResult{
		Type: QueryTypeBegin,
		TxID: txId,
	}, nil
}

// executeCommitQuery commits the existing transaction with the given ID.
func (db *DB) executeCommitQuery(ctx context.Context, queryTxID string) (QueryResult, error) {
	if queryTxID == "" {
		return QueryResult{}, ErrTxIdRequired
	}

	conn, returnConn, err := db.getReadWriteRawConn(ctx)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to get read-write connection from pool: %w", err)
	}
	defer func() { _ = returnConn() }()

	if _, err = conn.Query("COMMIT", nil); err != nil {
		return QueryResult{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	db.txId.Store("")
	db.txIdLastUsed.Store(time.Now())
	db.DBStats.IncCommits()
	db.txMu.Unlock()

	return QueryResult{
		Type: QueryTypeCommit,
	}, nil
}

// executeRollbackQuery rolls back an existing transaction.
func (db *DB) executeRollbackQuery(ctx context.Context, queryTxID string) (QueryResult, error) {
	if queryTxID == "" {
		return QueryResult{}, ErrTxIdRequired
	}

	conn, returnConn, err := db.getReadWriteRawConn(ctx)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to get read-write connection from pool: %w", err)
	}
	defer func() { _ = returnConn() }()

	if _, err = conn.Query("ROLLBACK", nil); err != nil {
		return QueryResult{}, fmt.Errorf("failed to rollback transaction: %w", err)
	}

	db.txId.Store("")
	db.txIdLastUsed.Store(time.Now())
	db.DBStats.IncRollbacks()
	db.txMu.Unlock()

	return QueryResult{
		Type: QueryTypeRollback,
	}, nil
}

// executeWriteQuery increments the write queue count, sends the task,
// waits for a response, and then decrements the counter.
func (db *DB) executeWriteQuery(ctx context.Context, query Query) (QueryResult, error) {
	db.DBStats.IncQueuedWrites()
	defer db.DBStats.DecQueuedWrites()

	db.writeMu.Lock()
	defer db.writeMu.Unlock()

	conn, returnConn, err := db.getReadWriteRawConn(ctx)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to get read-write connection from pool: %w", err)
	}
	defer func() { _ = returnConn() }()

	res, err := conn.Query(query.Query, query.Params)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to execute write query: %w", err)
	}

	db.DBStats.IncWrites()
	return QueryResult{
		Type:         QueryTypeWrite,
		LastInsertID: res.LastInsertID,
		RowsAffected: res.RowsAffected,
		Columns:      res.Columns,
		Types:        res.Types,
		Rows:         res.Rows,
	}, nil
}

// executeReadQuery executes a read query. If no transaction ID is provided, it
// will use the read-only connection. If a transaction ID is provided, it will
// use the read-write connection.
func (db *DB) executeReadQuery(ctx context.Context, query Query) (QueryResult, error) {
	var conn *sqlitec.Conn
	var returnConn func() error
	var err error

	if query.TxID == "" {
		conn, returnConn, err = db.getReadOnlyRawConn(ctx)
		if err != nil {
			return QueryResult{}, fmt.Errorf("failed to get read-only connection: %w", err)
		}
		defer func() { _ = returnConn() }()
	} else {
		conn, returnConn, err = db.getReadWriteRawConn(ctx)
		if err != nil {
			return QueryResult{}, fmt.Errorf("failed to get read-write connection: %w", err)
		}
		defer func() { _ = returnConn() }()
	}

	res, err := conn.Query(query.Query, query.Params)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to execute read query: %w", err)
	}

	db.DBStats.IncReads()
	return QueryResult{
		Type:    QueryTypeRead,
		Columns: res.Columns,
		Types:   res.Types,
		Rows:    res.Rows,
	}, nil
}
