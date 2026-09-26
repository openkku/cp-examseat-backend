// Package sqlite implements models.ExamRepository on top of SQLite.
package sqlite

import (
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{db: db}
	d.migrate()
	return d, nil
}

// Close releases the database handle.
func (d *Database) Close() error {
	return d.db.Close()
}
