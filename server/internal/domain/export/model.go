package export

import (
	"time"

	"server/internal/domain/asset"
	"server/internal/domain/timeline"
)

// Resolution defines the output video resolution.
type Resolution string

const (
	Resolution720p  Resolution = "720p"
	Resolution1080p Resolution = "1080p"
	Resolution4K    Resolution = "4k"
)

// Dimensions returns width and height for a resolution.
func (r Resolution) Dimensions() (int, int) {
	switch r {
	case Resolution720p:
		return 1280, 720
	case Resolution4K:
		return 3840, 2160
	case Resolution1080p:
		fallthrough
	default:
		return 1920, 1080
	}
}

// FrameRate defines the output video frame rate.
type FrameRate int

const (
	FPS24 FrameRate = 24
	FPS30 FrameRate = 30
	FPS60 FrameRate = 60
)

// Quality defines output encoding quality settings.
type Quality string

const (
	QualityStandard Quality = "standard"
	QualityHigh     Quality = "high"
)

// Format defines the container format.
type Format string

const (
	FormatMP4 Format = "mp4"
)

// Method defines the rendering method.
type Method string

const (
	MethodServer Method = "server"
)

// Settings holds all export configurations.
type Settings struct {
	Format     Format     `json:"format"`
	Filename   string     `json:"filename"`
	Resolution Resolution `json:"resolution"`
	FrameRate  FrameRate  `json:"frameRate"`
	Quality    Quality    `json:"quality"`
	Method     Method     `json:"method"`
}

// DefaultSettings returns Stage 1 default export configuration.
func DefaultSettings() Settings {
	return Settings{
		Format:     FormatMP4,
		Filename:   "export.mp4",
		Resolution: Resolution1080p,
		FrameRate:  FPS30,
		Quality:    QualityStandard,
		Method:     MethodServer,
	}
}

// Manifest is an immutable snapshot of project state and settings for an export job.
type Manifest struct {
	ProjectID string                 `json:"projectId"`
	Revision  uint64                 `json:"revision"`
	Timeline  timeline.Timeline      `json:"timeline"`
	Assets    map[string]asset.Asset `json:"assets"`
	Settings  Settings               `json:"settings"`
	CreatedAt time.Time              `json:"createdAt"`
}

// JobStatus represents the current lifecycle state of an export job.
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRendering JobStatus = "rendering"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// Job represents an export task record.
type Job struct {
	ID        string     `json:"id"`
	RequestID string     `json:"requestId"`
	ProjectID string     `json:"projectId"`
	Revision  uint64     `json:"revision"`
	Status    JobStatus  `json:"status"`
	Progress  float64    `json:"progress"` // 0.0 to 100.0
	Attempts  int        `json:"attempts"`
	Error     string     `json:"error,omitempty"`
	OutputKey string     `json:"outputKey,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}
