package sqlite

import (
	"context"
	"database/sql"
)

// DB is an open, migrated SQLite database. It implements the repository port
// of every slice (see ports.go).
type DB struct {
	db *sql.DB
}

// Close releases the database handle.
func (d *DB) Close() error {
	return d.db.Close()
}

// RawQueryRow runs a read-only diagnostic query (used by tests).
func (d *DB) RawQueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}
