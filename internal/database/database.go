package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	Driver string
}

// NewConnection initializes a database connection pool and runs schema migrations.
func NewConnection(driver, connStr string) (*DB, error) {
	if driver == "" {
		driver = "postgres"
	}
	if driver == "sqlite3" {
		driver = "sqlite"
	}

	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database (%s): %w", driver, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database (%s): %w", driver, err)
	}

	// Set pool limits
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	dbConn := &DB{
		DB:     db,
		Driver: driver,
	}

	if err := dbConn.Migrate(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to execute migrations: %w", err)
	}

	return dbConn, nil
}

// Migrate executes schema migrations for database tables.
func (db *DB) Migrate(ctx context.Context) error {
	var createTablesQuery string

	if db.Driver == "sqlite" {
		createTablesQuery = `
		CREATE TABLE IF NOT EXISTS totp_secrets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_name TEXT NOT NULL UNIQUE,
			secret TEXT NOT NULL,
			algorithm TEXT NOT NULL DEFAULT 'SHA1',
			digits INTEGER NOT NULL DEFAULT 6,
			period INTEGER NOT NULL DEFAULT 30,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS backup_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_name TEXT NOT NULL,
			code_hash TEXT NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			used_at DATETIME
		);

		CREATE INDEX IF NOT EXISTS idx_backup_codes_account ON backup_codes(account_name);
		`
	} else {
		// PostgreSQL schema syntax
		createTablesQuery = `
		CREATE TABLE IF NOT EXISTS totp_secrets (
			id SERIAL PRIMARY KEY,
			account_name VARCHAR(255) NOT NULL UNIQUE,
			secret VARCHAR(255) NOT NULL,
			algorithm VARCHAR(50) NOT NULL DEFAULT 'SHA1',
			digits INTEGER NOT NULL DEFAULT 6,
			period INTEGER NOT NULL DEFAULT 30,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);

		CREATE TABLE IF NOT EXISTS backup_codes (
			id SERIAL PRIMARY KEY,
			account_name VARCHAR(255) NOT NULL,
			code_hash VARCHAR(255) NOT NULL,
			used BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL,
			used_at TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_backup_codes_account ON backup_codes(account_name);
		`
	}

	_, err := db.ExecContext(ctx, createTablesQuery)
	return err
}
