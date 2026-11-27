package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// openDB opens a SQLite database at the given path with standard high-performance WAL pragmas.
func openDB(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create db directory %s: %w", dir, err)
	}

	// modernc.org/sqlite DSN format: file:path?_pragma=val&...
	// We configure WAL mode, synchronous=NORMAL, busy_timeout=5000, foreign_keys=ON, cache_size=-2000 (2MB)
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=temp_store(MEMORY)&_pragma=cache_size(-2000)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db %s: %w", path, err)
	}

	// For SQLite, a connection pool of up to 10 connections works well with WAL mode
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite db %s: %w", path, err)
	}

	return db, nil
}
