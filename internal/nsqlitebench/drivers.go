package nsqlitebench

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/nsqlite/nsqlitego"
)

func createMattnDriver(tmpDir string) (*sql.DB, error) {
	dbPath := filepath.Join(tmpDir, "mattn.sqlite")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	fmt.Printf("Temporary SQLite database (mattn/go-sqlite3): %s\n", dbPath)
	return db, nil
}

func createNsqliteDriver(dsn string) (*sql.DB, error) {
	db, err := sql.Open("nsqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	fmt.Printf("Temporary NSQLite server: %s\n", dsn)
	return db, nil
}
