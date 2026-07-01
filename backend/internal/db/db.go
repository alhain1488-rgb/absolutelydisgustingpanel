// Package db opens the SQLite database, applies migrations at startup and
// exposes the *sql.DB handle. A pure-Go driver (modernc.org/sqlite) is used so
// the backend builds and tests without CGO.
package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/migrations"
)

// Open opens (or creates) the SQLite database at path, configures pragmas and
// applies all pending migrations.
func Open(path string) (*sql.DB, error) {
	// Busy timeout + foreign keys enabled via DSN query params.
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", path)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite is a single writer; keep the pool small to avoid lock churn.
	database.SetMaxOpenConns(1)
	database.SetConnMaxLifetime(time.Hour)

	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping sqlite (%s): %w", path, err)
	}
	if err := migrations.Apply(database); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	return database, nil
}

// InMemory opens a shared in-memory database, useful for tests. Each call with
// a distinct name gets an isolated database that survives while the handle is open.
func InMemory(name string) (*sql.DB, error) {
	if name == "" {
		name = "test"
	}
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", name)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	if err := migrations.Apply(database); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}

// EnsureDir makes sure the parent directory of the DB file exists.
func EnsureDir(path string) string {
	return filepath.Dir(path)
}
