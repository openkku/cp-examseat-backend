// Package database is the engine-neutral side of persistence: a handle to an
// open SQL database and the row helpers shared by the slices' database
// adapters. Engine specifics (driver, connection settings, schema) live in a
// subpackage such as database/sqlite.
package database

import (
	"context"
	"database/sql"
)

// DB is an open, migrated database.
type DB struct {
	db *sql.DB
}

// New wraps an open *sql.DB whose schema is already migrated.
func New(db *sql.DB) *DB {
	return &DB{db: db}
}

// SQL returns the underlying connection pool.
func (d *DB) SQL() *sql.DB {
	return d.db
}

// Close releases the database handle.
func (d *DB) Close() error {
	return d.db.Close()
}

// RawQueryRow runs a read-only diagnostic query (used by tests).
func (d *DB) RawQueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}
