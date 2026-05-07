package db

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/varavelio/nsqlite/internal/sqlitedrv"
)

func newConnector(
	dbPath string,
	readOnly bool,
	cacheSizeKB int,
	busyTimeout time.Duration,
) driver.Connector {
	optimizations := []string{
		// How long SQLite waits when the database is locked by another writer.
		fmt.Sprintf("PRAGMA BUSY_TIMEOUT = %d;", busyTimeout.Milliseconds()),
		// Negative value = kilobytes, giving predictable RAM usage regardless of page size.
		fmt.Sprintf("PRAGMA CACHE_SIZE = -%d;", max(cacheSizeKB, -cacheSizeKB)),

		"PRAGMA JOURNAL_MODE = WAL;",
		"PRAGMA SYNCHRONOUS = NORMAL;",
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
