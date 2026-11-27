package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"server/internal/domain/record"
	"server/internal/ports"
)

var _ ports.ProjectIndexRepository = (*IndexDB)(nil)

// IndexDB manages the global project catalog stored in a central SQLite database.
type IndexDB struct {
	db   *sql.DB
	path string
}

// OpenIndex opens or creates the project index SQLite database at dbPath.
func OpenIndex(dbPath string) (*IndexDB, error) {
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	idx := &IndexDB{
		db:   db,
		path: dbPath,
	}

	if err := idx.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init index schema: %w", err)
	}

	return idx, nil
}

func (i *IndexDB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		revision    INTEGER NOT NULL DEFAULT 0,
		clips       INTEGER NOT NULL DEFAULT 0,
		assets      INTEGER NOT NULL DEFAULT 0,
		created_at  DATETIME NOT NULL,
		updated_at  DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_projects_updated ON projects (updated_at DESC);
	`
	_, err := i.db.Exec(schema)
	return err
}

// Read returns all project metadata ordered by latest update first.
func (i *IndexDB) Read() ([]record.ProjectMetadata, error) {
	rows, err := i.db.Query(
		"SELECT id, name, revision, clips, assets, created_at, updated_at FROM projects ORDER BY updated_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query project metadata: %w", err)
	}
	defer rows.Close()

	var list []record.ProjectMetadata
	for rows.Next() {
		var s record.ProjectMetadata
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&s.ID, &s.Name, &s.Revision, &s.Clips, &s.Assets, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan project metadata: %w", err)
		}
		s.CreatedAt = createdAt
		s.UpdatedAt = updatedAt
		list = append(list, s)
	}
	if list == nil {
		list = []record.ProjectMetadata{}
	}
	return list, rows.Err()
}

// Save replaces or batch-saves a list of project metadata records.
func (i *IndexDB) Save(list []record.ProjectMetadata) error {
	tx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("begin save tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO projects (id, name, revision, clips, assets, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			revision = excluded.revision,
			clips = excluded.clips,
			assets = excluded.assets,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return fmt.Errorf("prepare upsert stmt: %w", err)
	}
	defer stmt.Close()

	for _, s := range list {
		if _, err := stmt.Exec(s.ID, s.Name, s.Revision, s.Clips, s.Assets, s.CreatedAt, s.UpdatedAt); err != nil {
			return fmt.Errorf("exec upsert %s: %w", s.ID, err)
		}
	}

	return tx.Commit()
}

// Upsert inserts or updates a single project metadata record.
func (i *IndexDB) Upsert(s record.ProjectMetadata) error {
	now := s.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	created := s.CreatedAt
	if created.IsZero() {
		created = now
	}

	_, err := i.db.Exec(`
		INSERT INTO projects (id, name, revision, clips, assets, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			revision = excluded.revision,
			clips = excluded.clips,
			assets = excluded.assets,
			updated_at = excluded.updated_at
	`, s.ID, s.Name, s.Revision, s.Clips, s.Assets, created, now)

	if err != nil {
		return fmt.Errorf("upsert project %s: %w", s.ID, err)
	}
	return nil
}

// Delete removes a project from the global index by ID.
func (i *IndexDB) Delete(id string) error {
	_, err := i.db.Exec("DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete project %s from index: %w", id, err)
	}
	return nil
}

// Close flushes and releases the SQLite index database connection.
func (i *IndexDB) Close() error {
	if i.db == nil {
		return nil
	}
	_, _ = i.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
	err := i.db.Close()
	i.db = nil
	return err
}
