package ports

import (
	"context"

	"server/internal/domain/export"
)

// SubmitRequest contains parameters for creating an export job.
type SubmitRequest struct {
	ProjectID string          `json:"projectId,omitempty"`
	RequestID string          `json:"requestId"`
	Revision  uint64          `json:"revision"`
	Settings  export.Settings `json:"settings"`
}

// ProbeStream holds stream inspection results.
type ProbeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	FrameRate string `json:"r_frame_rate,omitempty"`
}

// ProbeResult holds probed media container and streams data.
type ProbeResult struct {
	DurationSeconds float64       `json:"duration"`
	Streams         []ProbeStream `json:"streams"`
}

// JobRepository defines persistence for export jobs and immutable manifests.
type JobRepository interface {
	SaveJob(export.Job) error
	GetJob(id string) (export.Job, error)
	GetJobByRequestID(projectID, requestID string) (export.Job, error)
	ListJobs() ([]export.Job, error)
	SaveManifest(export.Manifest) error
	GetManifest(id string) (export.Manifest, error)
}

// MediaRenderer executes the rendering pipeline for an export manifest.
type MediaRenderer interface {
	Render(ctx context.Context, manifest export.Manifest, scratchDir string, onProgress func(percent float64)) (outputPath string, err error)
}

// MediaProber inspects and validates rendered output.
type MediaProber interface {
	Probe(ctx context.Context, path string) (ProbeResult, error)
}

// ExportService coordinates export job lifecycle.
type ExportService interface {
	Submit(ctx context.Context, req SubmitRequest) (export.Job, error)
	Get(ctx context.Context, id string) (export.Job, error)
	Cancel(ctx context.Context, id string) (export.Job, error)
	Retry(ctx context.Context, id string) (export.Job, error)
	DownloadURL(ctx context.Context, id string) (string, error)
}

// ExportManager aliases ExportService for interface consistency.
type ExportManager = ExportService
