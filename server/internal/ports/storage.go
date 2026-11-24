package ports

import (
	"context"

	"server/internal/domain/asset"
	"server/internal/domain/record"
)

// CatalogRepository defines persistence for project metadata and asset records.
type CatalogRepository interface {
	Read() (record.Metadata, error)
	Save(record.Metadata) error
}

// CatalogRepo aliases CatalogRepository.
type CatalogRepo = CatalogRepository

// OperationLogRepository defines append-only storage for ordered operation records.
type OperationLogRepository interface {
	Read() ([]record.Record, error)
	Append(record.Record) error
}

// OpLogRepo aliases OperationLogRepository.
type OpLogRepo = OperationLogRepository

// ObjectStore defines object storage operations for media uploads and streaming.
type ObjectStore interface {
	Bucket() string
	UploadPolicy(ctx context.Context, key string, src asset.Source) (asset.Upload, error)
	Stat(ctx context.Context, key string) (asset.ObjectInfo, error)
	Promote(ctx context.Context, stage, key, etag string) error
	ReadURL(ctx context.Context, key string) (string, error)
	Download(ctx context.Context, key string, targetPath string) error
	Upload(ctx context.Context, key string, sourcePath string, contentType string) error
}
