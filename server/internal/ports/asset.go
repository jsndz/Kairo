package ports

import (
	"context"

	"server/internal/domain/asset"
)

// AssetManager defines the contract for asset registration, upload coordination, and content retrieval.
type AssetManager interface {
	Register(src asset.Source) (asset.Asset, error)
	Begin(ctx context.Context, id string) (asset.Asset, asset.Upload, error)
	Complete(ctx context.Context, id, attempt string) (asset.Asset, error)
	Fail(id, attempt string) (asset.Asset, error)
	Content(ctx context.Context, id string) (string, error)
}

// AssetService aliases AssetManager.
type AssetService = AssetManager
