package ports

import (
	"context"

	"server/internal/domain/record"
)

// ProjectInstance combines a project's timeline state, operations, asset management, and lifecycle.
type ProjectInstance interface {
	ProjectService
	AssetManager
	Close() error
}

// ProjectManager coordinates project creation, lookup, listing, and lifecycle management.
type ProjectManager interface {
	List(ctx context.Context) ([]record.ProjectMetadata, error)
	Create(ctx context.Context, name string) (record.ProjectMetadata, ProjectInstance, error)
	Get(ctx context.Context, id string) (ProjectInstance, error)
	Delete(ctx context.Context, id string) error
	Touch(id string)
}

// ProjectResolver resolves a ProjectReader by project ID (e.g. for ExportService).
type ProjectResolver interface {
	ResolveReader(ctx context.Context, projectID string) (ProjectReader, error)
}
