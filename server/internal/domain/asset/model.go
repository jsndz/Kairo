package asset

import (
	"context"
	"time"
)

// Source contains metadata of media uploaded or registered by the client.
type Source struct {
	Name     string  `json:"name"`
	Kind     string  `json:"kind"` // "video", "audio", or "image"
	MIME     string  `json:"mime"`
	Bytes    int64   `json:"bytes"`
	Frames   int     `json:"frames"`
	Duration float64 `json:"duration"` // in seconds
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
}

// Asset represents an immutable media file registered with the project.
type Asset struct {
	Source
	ID        string    `json:"id"`
	Bucket    string    `json:"bucket"`
	ObjectKey string    `json:"objectKey"`
	Status    string    `json:"status"` // "pending", "uploading", "completed", "failed"
	CreatedAt time.Time `json:"createdAt"`
	Attempt   string    `json:"attempt,omitempty"`
}

// StageKey returns the temporary MinIO staging object key for an upload attempt.
func (a Asset) StageKey() string {
	return "staging/" + a.ID + "/" + a.Attempt
}

// Upload contains the presigned URL and form fields required for direct browser-to-MinIO uploads.
type Upload struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
}

// ObjectInfo represents basic object metadata retrieved from storage.
type ObjectInfo struct {
	Size       int64
	MIME, ETag string
}

// Objects abstracts the underlying object store (e.g. MinIO) for media uploads and streaming.
type Objects interface {
	Bucket() string
	UploadPolicy(ctx context.Context, key string, src Source) (Upload, error)
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	Promote(ctx context.Context, stage, key, etag string) error
	ReadURL(ctx context.Context, key string) (string, error)
}
