package ffmpeg_test

import (
	"strings"
	"testing"
	"time"

	"server/internal/adapters/outbound/ffmpeg"
	"server/internal/domain/asset"
	"server/internal/domain/export"
	"server/internal/domain/timeline"
)

func TestBuildFFmpegArgs(t *testing.T) {
	manifest := export.Manifest{
		ProjectID: "proj-1",
		Revision:  1,
		Timeline: timeline.Timeline{
			Name: "Test Project",
			Tracks: []timeline.Track{
				{ID: "video-1", Kind: "video"},
				{ID: "audio-1", Kind: "audio"},
			},
			Clips: []timeline.Clip{
				{
					ID:      "c1",
					TrackID: "video-1",
					MediaID: "a1",
					At:      0,
					Offset:  0,
					Length:  60, // 2 seconds
					Volume:  0.8,
				},
				{
					ID:      "c2",
					TrackID: "audio-1",
					MediaID: "a2",
					At:      30, // 1 second in
					Offset:  15,
					Length:  45, // 1.5 seconds
					Volume:  0.5,
				},
			},
		},
		Assets: map[string]asset.Asset{
			"a1": {ID: "a1", Source: asset.Source{Kind: "video", Frames: 300}, Status: "completed"},
			"a2": {ID: "a2", Source: asset.Source{Kind: "audio", Frames: 300}, Status: "completed"},
		},
		Settings:  export.DefaultSettings(),
		CreatedAt: time.Now(),
	}

	cfg := ffmpeg.PipelineConfig{
		Manifest: manifest,
		SourcePaths: map[string]string{
			"a1": "/tmp/test/a1.mp4",
			"a2": "/tmp/test/a2.mp3",
		},
		OutputPath: "/tmp/test/out.mp4",
	}

	args, err := ffmpeg.BuildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fullCmd := strings.Join(args, " ")
	if !strings.Contains(fullCmd, "-filter_complex") {
		t.Fatal("expected filter_complex in args")
	}
	if !strings.Contains(fullCmd, "overlay=eof_action=pass") {
		t.Fatal("expected overlay filter")
	}
	if !strings.Contains(fullCmd, "amix=inputs=") {
		t.Fatal("expected amix filter")
	}
	if !strings.Contains(fullCmd, "-c:v libx264") || !strings.Contains(fullCmd, "-c:a aac") {
		t.Fatal("expected h264 and aac encoders")
	}
	if !strings.Contains(fullCmd, "/tmp/test/out.mp4") {
		t.Fatal("expected output path")
	}
}

func TestStage1ReconstructState(t *testing.T) {
	manifest := export.Manifest{
		Timeline: timeline.Timeline{
			Tracks: []timeline.Track{
				{ID: "v1", Kind: "video"},
				{ID: "v2", Kind: "video"},
			},
			Clips: []timeline.Clip{
				{ID: "c1", TrackID: "v2", MediaID: "a1", At: 10, Length: 50},
				{ID: "c2", TrackID: "v1", MediaID: "a1", At: 0, Length: 30},
			},
		},
		Assets: map[string]asset.Asset{
			"a1": {ID: "a1", Source: asset.Source{Kind: "video"}},
		},
		Settings: export.DefaultSettings(),
	}

	st := ffmpeg.Stage1ReconstructState(manifest, nil, nil)
	if st.TotalFrames != 60 {
		t.Fatalf("expected total frames 60, got %d", st.TotalFrames)
	}
	if st.DurationSeconds != 2.0 {
		t.Fatalf("expected duration 2.0s, got %f", st.DurationSeconds)
	}
	// Clips should be ordered by track stacking order: v1 (index 0) first, then v2 (index 1)
	if st.OrderedClips[0].TrackID != "v1" || st.OrderedClips[1].TrackID != "v2" {
		t.Fatalf("expected clips sorted by track order: %+v", st.OrderedClips)
	}
}

func TestBuildFFmpegArgs_VideoWithoutAudio(t *testing.T) {
	manifest := export.Manifest{
		Timeline: timeline.Timeline{
			Tracks: []timeline.Track{{ID: "v1", Kind: "video"}},
			Clips: []timeline.Clip{
				{ID: "c1", TrackID: "v1", MediaID: "v_silent", At: 0, Length: 60, Volume: 1.0},
			},
		},
		Assets: map[string]asset.Asset{
			"v_silent": {ID: "v_silent", Source: asset.Source{Kind: "video"}},
		},
		Settings: export.DefaultSettings(),
	}

	cfg := ffmpeg.PipelineConfig{
		Manifest: manifest,
		SourcePaths: map[string]string{
			"v_silent": "/tmp/silent.webm",
		},
		SourceHasAudio: map[string]bool{
			"v_silent": false, // No audio stream!
		},
		OutputPath: "/tmp/out.mp4",
	}

	args, err := ffmpeg.BuildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fullCmd := strings.Join(args, " ")
	// Must NOT contain [0:a] filter!
	if strings.Contains(fullCmd, "[0:a]") {
		t.Fatalf("expected no [0:a] filter for video without audio stream, got: %s", fullCmd)
	}
	// Must still generate pure silence track for output audio
	if !strings.Contains(fullCmd, "aevalsrc=0") {
		t.Fatalf("expected aevalsrc silence generator, got: %s", fullCmd)
	}
}
