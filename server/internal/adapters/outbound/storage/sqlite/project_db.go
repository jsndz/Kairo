package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"server/internal/domain/asset"
	"server/internal/domain/record"
	"server/internal/ports"
)

var (
	_ ports.OperationLogRepository = (*OperationLog)(nil)
	_ ports.SnapshotRepository     = (*OperationLog)(nil)
	_ record.DeltaLogReader        = (*OperationLog)(nil)
	_ ports.CatalogRepository      = (*AssetCatalog)(nil)
)

// ProjectDB manages the embedded SQLite database for a single project.
type ProjectDB struct {
	db   *sql.DB
	path string
	log  *OperationLog
	cat  *AssetCatalog
}

// OpenProject opens or initializes a project database at dbPath.
func OpenProject(dbPath string) (*ProjectDB, error) {
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	p := &ProjectDB{
		db:   db,
		path: dbPath,
	}
	p.log = &OperationLog{db: db}
	p.cat = &AssetCatalog{db: db}

	if err := p.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init project schema: %w", err)
	}

	return p, nil
}

func (p *ProjectDB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS meta (
		key         TEXT PRIMARY KEY,
		value       TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS assets (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		kind        TEXT NOT NULL,
		mime        TEXT NOT NULL,
		bytes       INTEGER NOT NULL,
		duration    REAL NOT NULL,
		frames      INTEGER NOT NULL,
		width       INTEGER NOT NULL,
		height      INTEGER NOT NULL,
		object_key  TEXT NOT NULL,
		bucket      TEXT NOT NULL,
		status      TEXT NOT NULL,
		created_at  DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS operations (
		sequence    INTEGER PRIMARY KEY,
		id          TEXT NOT NULL,
		type        TEXT NOT NULL,
		payload     TEXT NOT NULL,
		created_at  DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS snapshots (
		sequence    INTEGER PRIMARY KEY,
		state       TEXT NOT NULL,
		created_at  DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_ops_seq ON operations (sequence);
	`
	_, err := p.db.Exec(schema)
	return err
}

// DB returns the underlying *sql.DB handle.
func (p *ProjectDB) DB() *sql.DB {
	return p.db
}

// Log returns the operation log and snapshot adapter.
func (p *ProjectDB) Log() *OperationLog {
	return p.log
}

// Catalog returns the project metadata and asset catalog adapter.
func (p *ProjectDB) Catalog() *AssetCatalog {
	return p.cat
}

// Close flushes the WAL checkpoint and closes the database connection pool.
func (p *ProjectDB) Close() error {
	if p.db == nil {
		return nil
	}
	// Attempt non-blocking WAL truncate checkpoint to keep disk compact
	_, _ = p.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
	err := p.db.Close()
	p.db = nil
	return err
}

// OperationLog provides append-only operations and snapshot storage.
type OperationLog struct {
	db *sql.DB
}

// Read loads all historical operations in sequence order.
func (l *OperationLog) Read() ([]record.Record, error) {
	rows, err := l.db.Query("SELECT sequence, id, type, payload, created_at FROM operations ORDER BY sequence ASC")
	if err != nil {
		return nil, fmt.Errorf("query operations: %w", err)
	}
	defer rows.Close()

	var records []record.Record
	for rows.Next() {
		var rec record.Record
		var payloadStr string
		var createdAt time.Time
		if err := rows.Scan(&rec.Sequence, &rec.ID, &rec.Type, &payloadStr, &createdAt); err != nil {
			return nil, fmt.Errorf("scan operation: %w", err)
		}
		rec.Payload = json.RawMessage(payloadStr)
		rec.Timestamp = createdAt
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ReadSince loads operations with sequence strictly greater than sinceSeq.
func (l *OperationLog) ReadSince(sinceSeq uint64) ([]record.Record, error) {
	rows, err := l.db.Query(
		"SELECT sequence, id, type, payload, created_at FROM operations WHERE sequence > ? ORDER BY sequence ASC",
		sinceSeq,
	)
	if err != nil {
		return nil, fmt.Errorf("query delta operations: %w", err)
	}
	defer rows.Close()

	var records []record.Record
	for rows.Next() {
		var rec record.Record
		var payloadStr string
		var createdAt time.Time
		if err := rows.Scan(&rec.Sequence, &rec.ID, &rec.Type, &payloadStr, &createdAt); err != nil {
			return nil, fmt.Errorf("scan delta operation: %w", err)
		}
		rec.Payload = json.RawMessage(payloadStr)
		rec.Timestamp = createdAt
		records = append(records, rec)
	}
	return records, rows.Err()
}

// Append persists a new operation record.
func (l *OperationLog) Append(rec record.Record) error {
	ts := rec.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	_, err := l.db.Exec(
		"INSERT INTO operations (sequence, id, type, payload, created_at) VALUES (?, ?, ?, ?, ?)",
		rec.Sequence, rec.ID, rec.Type, string(rec.Payload), ts,
	)
	if err != nil {
		return fmt.Errorf("insert operation %d: %w", rec.Sequence, err)
	}
	return nil
}

// GetLatestSnapshot fetches the highest sequence snapshot, if one exists.
func (l *OperationLog) GetLatestSnapshot() (*record.Snapshot, error) {
	row := l.db.QueryRow("SELECT sequence, state, created_at FROM snapshots ORDER BY sequence DESC LIMIT 1")
	var seq uint64
	var stateStr string
	var createdAt time.Time
	if err := row.Scan(&seq, &stateStr, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest snapshot: %w", err)
	}

	return &record.Snapshot{
		Sequence:  seq,
		State:     json.RawMessage(stateStr),
		Timestamp: createdAt,
	}, nil
}

// SaveSnapshot persists a materialized state checkpoint.
func (l *OperationLog) SaveSnapshot(sequence uint64, state []byte) error {
	_, err := l.db.Exec(
		"INSERT OR REPLACE INTO snapshots (sequence, state, created_at) VALUES (?, ?, ?)",
		sequence, string(state), time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("save snapshot %d: %w", sequence, err)
	}
	return nil
}

// AssetCatalog manages project metadata and media assets in SQLite.
type AssetCatalog struct {
	db *sql.DB
}

// Read retrieves project metadata and all registered media assets.
func (c *AssetCatalog) Read() (record.Metadata, error) {
	var id string
	err := c.db.QueryRow("SELECT value FROM meta WHERE key = 'id'").Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return record.Metadata{}, os.ErrNotExist
		}
		return record.Metadata{}, fmt.Errorf("query project id: %w", err)
	}

	rows, err := c.db.Query(
		"SELECT id, name, kind, mime, bytes, duration, frames, width, height, object_key, bucket, status, created_at FROM assets",
	)
	if err != nil {
		return record.Metadata{}, fmt.Errorf("query assets: %w", err)
	}
	defer rows.Close()

	assets := make(map[string]asset.Asset)
	for rows.Next() {
		var a asset.Asset
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Kind, &a.MIME, &a.Bytes, &a.Duration, &a.Frames,
			&a.Width, &a.Height, &a.ObjectKey, &a.Bucket, &a.Status, &a.CreatedAt,
		); err != nil {
			return record.Metadata{}, fmt.Errorf("scan asset: %w", err)
		}
		assets[a.ID] = a
	}

	return record.Metadata{
		ID:     id,
		Assets: assets,
	}, rows.Err()
}

// Save updates project metadata and the asset catalog in a single transaction.
func (c *AssetCatalog) Save(meta record.Metadata) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("begin asset tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if meta.ID != "" {
		_, err = tx.Exec(
			"INSERT INTO meta (key, value) VALUES ('id', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
			meta.ID,
		)
		if err != nil {
			return fmt.Errorf("upsert meta id: %w", err)
		}
	}

	for _, a := range meta.Assets {
		status := a.Status
		if status == "" {
			status = "completed"
		}
		_, err = tx.Exec(`
			INSERT INTO assets (id, name, kind, mime, bytes, duration, frames, width, height, object_key, bucket, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				name = excluded.name,
				kind = excluded.kind,
				mime = excluded.mime,
				bytes = excluded.bytes,
				duration = excluded.duration,
				frames = excluded.frames,
				width = excluded.width,
				height = excluded.height,
				object_key = excluded.object_key,
				bucket = excluded.bucket,
				status = excluded.status,
				created_at = excluded.created_at
		`, a.ID, a.Name, a.Kind, a.MIME, a.Bytes, a.Duration, a.Frames, a.Width, a.Height, a.ObjectKey, a.Bucket, status, a.CreatedAt)
		if err != nil {
			return fmt.Errorf("upsert asset %s: %w", a.ID, err)
		}
	}

	return tx.Commit()
}
