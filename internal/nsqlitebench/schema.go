package nsqlitebench

import (
	"context"
	"database/sql"
)

// recreateSchema drops all tables and recreates them.
func recreateSchema(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,

		`DROP TABLE IF EXISTS comments`,
		`DROP TABLE IF EXISTS articles`,
		`DROP TABLE IF EXISTS users`,

		`CREATE TABLE users (
			id INTEGER PRIMARY KEY NOT NULL,
			created INTEGER NOT NULL,
			email TEXT NOT NULL,
			active INTEGER NOT NULL
		)`,
		`CREATE INDEX users_created ON users(created)`,

		`CREATE TABLE articles (
			id INTEGER PRIMARY KEY NOT NULL,
			created INTEGER NOT NULL,
			userId INTEGER NOT NULL REFERENCES users(id),
			text TEXT NOT NULL
		)`,
		`CREATE INDEX articles_created ON articles(created)`,
		`CREATE INDEX articles_userId ON articles(userId)`,

		`CREATE TABLE comments (
			id INTEGER PRIMARY KEY NOT NULL,
			created INTEGER NOT NULL,
			articleId INTEGER NOT NULL REFERENCES articles(id),
			text TEXT NOT NULL
		)`,
		`CREATE INDEX comments_created ON comments(created)`,
		`CREATE INDEX comments_articleId ON comments(articleId)`,
	}

	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return err
		}
	}

	return nil
}
