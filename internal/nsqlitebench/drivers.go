package nsqlitebench

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/nsqlite/nsqlitego"
	_ "modernc.org/sqlite"
)

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

func createMattnDriver(tmpDir string) (*sql.DB, error) {
	queryParams := url.Values{}
	queryParams.Add("_foreign_keys", "1")
	queryParams.Add("_journal_mode", "WAL")
	queryParams.Add("_busy_timeout", "5000")

	dbPath := filepath.Join(tmpDir, "mattn.sqlite?"+queryParams.Encode())

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

func createModerncDriver(tmpDir string) (*sql.DB, error) {
	queryParams := url.Values{}
	queryParams.Add("_pragma", "foreign_keys(1)")
	queryParams.Add("_pragma", "journal_mode(WAL)")
	queryParams.Add("_pragma", "busy_timeout(5000)")

	dbPath := filepath.Join(tmpDir, "modernc.sqlite?"+queryParams.Encode())
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	fmt.Printf("Temporary SQLite database (modernc.org/sqlite): %s\n", dbPath)
	return db, nil
}
