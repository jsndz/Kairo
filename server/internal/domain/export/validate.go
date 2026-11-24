package export

import (
	"fmt"
	"path/filepath"
	"strings"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	"server/internal/domain/timeline"
)

// ValidateSettings validates and normalizes export settings according to SD1.
func ValidateSettings(s *Settings) error {
	if s.Format == "" {
		s.Format = FormatMP4
	}
	if s.Format != FormatMP4 {
		return fmt.Errorf("%w: format %q not supported; only mp4 is supported in Stage 1", domErrors.ErrInvalid, s.Format)
	}

	if s.Resolution == "" {
		s.Resolution = Resolution1080p
	}
	switch s.Resolution {
	case Resolution720p, Resolution1080p, Resolution4K:
	default:
		return fmt.Errorf("%w: invalid resolution %q; allowed: 720p, 1080p, 4k", domErrors.ErrInvalid, s.Resolution)
	}

	if s.FrameRate == 0 {
		s.FrameRate = FPS30
	}
	switch s.FrameRate {
	case FPS24, FPS30, FPS60:
	default:
		return fmt.Errorf("%w: invalid frame rate %d; allowed: 24, 30, 60", domErrors.ErrInvalid, s.FrameRate)
	}

	if s.Quality == "" {
		s.Quality = QualityStandard
	}
	switch s.Quality {
	case QualityStandard, QualityHigh:
	default:
		return fmt.Errorf("%w: invalid quality %q; allowed: standard, high", domErrors.ErrInvalid, s.Quality)
	}

	if s.Method == "" {
		s.Method = MethodServer
	}
	if s.Method != MethodServer {
		return fmt.Errorf("%w: method %q not supported; only server export is supported in Stage 1", domErrors.ErrInvalid, s.Method)
	}

	s.Filename = SanitizeFilename(s.Filename)
	return nil
}

// SanitizeFilename ensures safe, clean filename ending in .mp4.
func SanitizeFilename(name string) string {
	clean := filepath.Base(strings.TrimSpace(name))
	if clean == "" || clean == "." || clean == "/" {
		clean = "export.mp4"
	}
	if !strings.HasSuffix(strings.ToLower(clean), ".mp4") {
		clean += ".mp4"
	}
	return clean
}

// ValidateTimelineForExport verifies that timeline is non-empty and all referenced assets are completed.
func ValidateTimelineForExport(t timeline.Timeline, assets map[string]asset.Asset) (map[string]asset.Asset, error) {
	if len(t.Clips) == 0 {
		return nil, fmt.Errorf("%w: timeline has no clips to export", domErrors.ErrInvalid)
	}

	usedAssets := make(map[string]asset.Asset)
	for _, clip := range t.Clips {
		a, ok := assets[clip.MediaID]
		if !ok {
			return nil, fmt.Errorf("%w: asset %q referenced by clip %q not found", domErrors.ErrInvalid, clip.MediaID, clip.ID)
		}
		if a.Status != "completed" {
			return nil, fmt.Errorf("%w: asset %q is not fully uploaded (status: %s)", domErrors.ErrInvalid, a.ID, a.Status)
		}
		usedAssets[a.ID] = a
	}

	return usedAssets, nil
}
