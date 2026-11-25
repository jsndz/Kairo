package ports

import (
	"server/internal/domain/asset"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
)

// State is the project snapshot exposed to clients.
type State struct {
	ID       string `json:"id"`
	Revision uint64 `json:"revision"`
	Project  struct {
		timeline.Timeline
		Assets map[string]asset.Asset `json:"assets"`
	} `json:"project"`
	CanUndo bool `json:"canUndo"`
	CanRedo bool `json:"canRedo"`
}

// OwnershipChecker verifies clip edit permissions.
type OwnershipChecker interface {
	IsAllowed(clientID, clipID string) bool
}

// ProjectReader provides read access to project state and historical operations.
type ProjectReader interface {
	Get() State
	GetOpsSince(seq uint64) ([]record.Record, error)
	ProjectID() string
	Summary() record.ProjectSummary
}

// OperationCommitter validates and applies timeline mutations.
type OperationCommitter interface {
	Commit(edit timeline.Edit) (State, error)
	CommitAuthor(edit timeline.Edit, authorID string, checker OwnershipChecker) (State, error)
}

// ProjectService combines project reading and operation committing.
type ProjectService interface {
	ProjectReader
	OperationCommitter
}

// OperationService aliases OperationCommitter for interface segregation.
type OperationService = OperationCommitter
