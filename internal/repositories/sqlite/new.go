// Package sqlite implements models.ExamRepository on top of SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openkku/cp-examseat-backend/internal/models"

	_ "modernc.org/sqlite"
)

// Database is the SQLite-backed exam repository.
type Database struct {
	db *sql.DB
}

var _ models.ExamRepository = (*Database)(nil)

// New opens (creating if needed) the database at dbPath and migrates its schema.
func New(dbPath string) (*Database, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	// exams.db holds student data: keep its directory private to the owner and group.
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Connection settings go in the DSN so they apply to every pooled
	// connection, not just the one that happened to run a PRAGMA: foreign keys
	// (ON DELETE CASCADE), waiting on locks instead of failing, and
	// IMMEDIATE transactions so concurrent writers queue instead of erroring
	// with SQLITE_BUSY when upgrading a read lock.
	dsn := "file:" + dbPath +
		"?_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=temp_store(MEMORY)" +
		"&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{db: db}
	if err := d.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

// Close releases the database handle.
func (d *Database) Close() error {
	return d.db.Close()
}

// RawQueryRow runs a read-only diagnostic query (used by tests and admin tooling).
func (d *Database) RawQueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}
