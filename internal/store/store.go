// Package store is module 2: it persists captured records into SQLite and
// retains raw bytes on disk. It implements sink.Sink, so the collection and
// normalize layers from module 1 are unchanged — only the sink is swapped.
package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// SchemaVersion is the current on-disk schema revision.
const SchemaVersion = "1"

// Store owns the SQLite database and the raw-files directory.
type Store struct {
	db      *sql.DB
	dataDir string
	rawDir  string
	mu      sync.Mutex // serializes writes (one capture at a time is plenty)
}

// Open initializes the data directory, opens (creating if needed) the database,
// applies the schema, and ensures the raw/ directory exists.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	rawDir := filepath.Join(dataDir, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return nil, fmt.Errorf("create raw dir: %w", err)
	}

	dsn := "file:" + filepath.Join(dataDir, "tracelab.db") +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if _, err := db.Exec(
		`INSERT INTO schema_meta(key,value) VALUES('version',?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`, SchemaVersion); err != nil {
		db.Close()
		return nil, fmt.Errorf("record schema version: %w", err)
	}
	return &Store{db: db, dataDir: dataDir, rawDir: rawDir}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the underlying handle for read queries and tests.
func (s *Store) DB() *sql.DB { return s.db }
