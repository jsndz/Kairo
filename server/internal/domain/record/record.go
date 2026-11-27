package record

import (
	"encoding/json"
	"time"

	"server/internal/domain/asset"
)

// Record is an immutable operation log entry persisted in append-only storage.
type Record struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"projectId"`
	Sequence  uint64          `json:"sequence"`
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// Log defines the storage interface for ordered operation records.
type Log interface {
	Read() ([]Record, error)
	Append(Record) error
}

// Metadata holds project-level properties and the registered assets catalog.
type Metadata struct {
	ID     string                 `json:"id"`
	Assets map[string]asset.Asset `json:"assets"`
}

// Catalog defines the persistence interface for project and asset metadata.
type Catalog interface {
	Read() (Metadata, error)
	Save(Metadata) error
}

// Snapshot is a materialized timeline state checkpoint at a specific sequence.
type Snapshot struct {
	Sequence  uint64          `json:"sequence"`
	State     json.RawMessage `json:"state"`
	Timestamp time.Time       `json:"timestamp"`
}

// SnapshotStore defines persistence for materialized project state checkpoints.
type SnapshotStore interface {
	GetLatestSnapshot() (*Snapshot, error)
	SaveSnapshot(sequence uint64, state json.RawMessage) error
}

// DeltaLogReader extends operation log with range queries.
type DeltaLogReader interface {
	ReadSince(sinceSeq uint64) ([]Record, error)
}
