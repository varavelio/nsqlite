package db

import (
	"database/sql/driver"

	"github.com/varavelio/nsqlite/internal/nsqlite/sqlitedrv"
)

func newConnector(dbPath string, readOnly bool) driver.Connector {
	optimizations := []string{
		"PRAGMA JOURNAL_MODE = WAL;",
		"PRAGMA BUSY_TIMEOUT = 5000;",
		"PRAGMA SYNCHRONOUS = NORMAL;",
		"PRAGMA CACHE_SIZE = 10000;",
		"PRAGMA FOREIGN_KEYS = true;",
		"PRAGMA TEMP_STORE = MEMORY;",
		"PRAGMA MMAP_SIZE = 536870912;", // 512MB
	}

	if readOnly {
		optimizations = append(optimizations, "PRAGMA QUERY_ONLY = true;")
	}

	return sqlitedrv.NewConnector(
		dbPath,
		sqlitedrv.WithPostConnectQueries(optimizations),
	)
}
