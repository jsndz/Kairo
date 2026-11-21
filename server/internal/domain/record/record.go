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
