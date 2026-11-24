package export_test

import (
	"testing"

	"server/internal/domain/asset"
	"server/internal/domain/export"
	"server/internal/domain/timeline"
)

func TestValidateSettings(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		s := export.Settings{}
		if err := export.ValidateSettings(&s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Format != export.FormatMP4 || s.Resolution != export.Resolution1080p ||
			s.FrameRate != export.FPS30 || s.Quality != export.QualityStandard ||
			s.Method != export.MethodServer || s.Filename != "export.mp4" {
			t.Fatalf("defaults not set correctly: %+v", s)
		}
	})

	t.Run("custom valid settings", func(t *testing.T) {
		s := export.Settings{
			Format:     export.FormatMP4,
			Filename:   "video_final",
			Resolution: export.Resolution4K,
			FrameRate:  export.FPS60,
			Quality:    export.QualityHigh,
			Method:     export.MethodServer,
		}
		if err := export.ValidateSettings(&s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Filename != "video_final.mp4" {
			t.Fatalf("expected .mp4 suffix, got %q", s.Filename)
		}
	})

	t.Run("invalid resolution", func(t *testing.T) {
		s := export.Settings{Resolution: "480p"}
		if err := export.ValidateSettings(&s); err == nil {
			t.Fatal("expected error for invalid resolution")
		}
	})

	t.Run("invalid frame rate", func(t *testing.T) {
		s := export.Settings{FrameRate: 120}
		if err := export.ValidateSettings(&s); err == nil {
			t.Fatal("expected error for invalid frame rate")
		}
	})

	t.Run("invalid method", func(t *testing.T) {
		s := export.Settings{Method: "local"}
		if err := export.ValidateSettings(&s); err == nil {
			t.Fatal("expected error for local method")
		}
	})
}

func TestValidateTimelineForExport(t *testing.T) {
	t.Run("empty timeline rejected", func(t *testing.T) {
		tl := timeline.Empty()
		_, err := export.ValidateTimelineForExport(tl, nil)
		if err == nil {
			t.Fatal("expected error for empty timeline")
		}
	})

	t.Run("asset incomplete rejected", func(t *testing.T) {
		tl := timeline.Empty()
		tl.Clips = append(tl.Clips, timeline.Clip{
			ID:      "c1",
			TrackID: "video-1",
			MediaID: "a1",
			At:      0,
			Offset:  0,
			Length:  30,
			Volume:  1.0,
		})
		assets := map[string]asset.Asset{
			"a1": {ID: "a1", Status: "uploading"},
		}
		_, err := export.ValidateTimelineForExport(tl, assets)
		if err == nil {
			t.Fatal("expected error for incomplete asset")
		}
	})

	t.Run("valid timeline accepted", func(t *testing.T) {
		tl := timeline.Empty()
		tl.Clips = append(tl.Clips, timeline.Clip{
			ID:      "c1",
			TrackID: "video-1",
			MediaID: "a1",
			At:      0,
			Offset:  0,
			Length:  30,
			Volume:  1.0,
		})
		assets := map[string]asset.Asset{
			"a1": {ID: "a1", Status: "completed"},
			"a2": {ID: "a2", Status: "pending"}, // unused asset
		}
		used, err := export.ValidateTimelineForExport(tl, assets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(used) != 1 || used["a1"].ID != "a1" {
			t.Fatalf("expected only used asset a1, got %+v", used)
		}
	})
}
