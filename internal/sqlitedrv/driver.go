// Package sqlitedrv provides a basic database/sql/driver implementation for
// the custom SQLite C API wrapper of this project.
//
// This package is used to take advantage of the internal connection pooling
// that is provided by the database/sql and it should provide a way to access
// the underlying SQLite C API wrapper and should not be used directly.
package sqlitedrv

import (
	"context"
	"database/sql/driver"
	"fmt"

	"github.com/varavelio/nsqlite/internal/sqlitec"
)

var (
	_ driver.Driver          = (*Driver)(nil)
	_ driver.Conn            = (*Conn)(nil)
	_ driver.Validator       = (*Conn)(nil)
	_ driver.SessionResetter = (*Conn)(nil)
	_ driver.Connector       = (*Connector)(nil)
)

// Driver implements the database/sql/driver interface.
type Driver struct{}

// Open creates a new connection to the SQLite database.
func (driver *Driver) Open(dsn string) (driver.Conn, error) {
	connector := NewConnector(dsn)
	return connector.Connect(context.Background())
}

type connectorOption func(*Connector)

// WithPostConnectQueries sets a slice of queries to be executed after a
// connection is established.
func WithPostConnectQueries(queries []string) connectorOption {
	return func(connector *Connector) {
		connector.postConnectQueries = queries
	}
}

// Connector implements the database/sql/driver.Connector interface.
type Connector struct {
	dsn                string
	postConnectQueries []string
}

// NewConnector creates a new connector to the SQLite database.
func NewConnector(dsn string, options ...connectorOption) driver.Connector {
	connector := &Connector{
		dsn: dsn,
	}

	for _, option := range options {
		option(connector)
	}

	return connector
}

// Connect creates a new connection to the SQLite database.
func (connector *Connector) Connect(_ context.Context) (driver.Conn, error) {
	return newConn(connector.dsn, connector.postConnectQueries)
}

// Driver returns the driver.
func (connector *Connector) Driver() driver.Driver {
	return &Driver{}
}

// Conn implements the database/sql/driver.Conn interface.
type Conn struct {
	conn *sqlitec.Conn
}

// newConn creates a new connection to the SQLite database.
func newConn(dsn string, postConnectQueries []string) (driver.Conn, error) {
	conn, err := sqlitec.Open(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	for _, query := range postConnectQueries {
		if _, err := conn.Query(query, nil); err != nil {
			return nil, fmt.Errorf(`failed to execute "%s" post-connect query: %w`, query, err)
		}
	}

	return &Conn{
		conn: conn,
	}, nil
}

// RawConn returns the underlying SQLite C API connection.
func (conn *Conn) RawConn() *sqlitec.Conn {
	return conn.conn
}

// Close closes the connection to the SQLite database.
func (conn *Conn) Close() error {
	if err := conn.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}
	return nil
}

// Prepare is no-op.
func (conn *Conn) Prepare(query string) (driver.Stmt, error) {
	return nil, nil
}

// Begin is no-op.
func (conn *Conn) Begin() (driver.Tx, error) {
	return nil, nil
}

// TODO: Correctly implement the SessionResetter and Validator interfaces

// ResetSession is no-op.
func (conn *Conn) ResetSession(_ context.Context) error {
	return nil
}

// IsValid is no-op.
func (conn *Conn) IsValid() bool {
	return true
}
