package ffmpeg_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"server/internal/adapters/outbound/ffmpeg"
	"server/internal/domain/asset"
	"server/internal/domain/export"
	"server/internal/domain/timeline"
)

type diskStore struct{}

func (diskStore) Bucket() string { return "test" }
func (diskStore) UploadPolicy(_ context.Context, _ string, _ asset.Source) (asset.Upload, error) {
	return asset.Upload{}, nil
}
func (diskStore) Stat(_ context.Context, _ string) (asset.ObjectInfo, error) {
	return asset.ObjectInfo{}, nil
}
func (diskStore) Promote(_ context.Context, _, _, _ string) error { return nil }
func (diskStore) ReadURL(_ context.Context, _ string) (string, error) { return "", nil }
func (diskStore) Download(_ context.Context, key string, targetPath string) error {
	// Copy from key (local file path) to targetPath
	data, err := os.ReadFile(key)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0600)
}
func (diskStore) Upload(_ context.Context, _, _, _ string) error { return nil }

func TestE2E_FFmpegAndProbe(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg binary not available on host")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe binary not available on host")
	}

	tempDir := t.TempDir()

	// 1. Generate a 1-second sample video file (red canvas with sine audio)
	video1Path := filepath.Join(tempDir, "source_v1.mp4")
	genCmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "testsrc=size=640x360:rate=30:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		video1Path,
	)
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test video: %v, output: %s", err, string(out))
	}

	// 2. Generate a sample image file
	imgPath := filepath.Join(tempDir, "source_img.png")
	imgCmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "color=c=blue:size=300x300:duration=1",
		"-vframes", "1",
		imgPath,
	)
	if out, err := imgCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test image: %v, output: %s", err, string(out))
	}

	// 3. Construct manifest with multi-track layout:
	// - Track video-1: clip c1 (video1, at frame 0, length 30 frames = 1s, vol 0.8)
	// - Track video-2: clip c2 (img, at frame 15, length 30 frames = 1s)
	// - Total timeline duration: 45 frames = 1.5 seconds
	manifest := export.Manifest{
		ProjectID: "proj-e2e",
		Revision:  1,
		Timeline: timeline.Timeline{
			Tracks: []timeline.Track{
				{ID: "video-1", Kind: "video"},
				{ID: "video-2", Kind: "video"},
			},
			Clips: []timeline.Clip{
				{ID: "c1", TrackID: "video-1", MediaID: "m-video", At: 0, Offset: 0, Length: 30, Volume: 0.8},
				{ID: "c2", TrackID: "video-2", MediaID: "m-img", At: 15, Offset: 0, Length: 30, Volume: 0.0},
			},
		},
		Assets: map[string]asset.Asset{
			"m-video": {
				ID:        "m-video",
				Source:    asset.Source{Kind: "video", Name: "source_v1.mp4", Frames: 60},
				ObjectKey: video1Path,
				Status:    "completed",
			},
			"m-img": {
				ID:        "m-img",
				Source:    asset.Source{Kind: "image", Name: "source_img.png", Frames: 300},
				ObjectKey: imgPath,
				Status:    "completed",
			},
		},
		Settings: export.Settings{
			Format:     export.FormatMP4,
			Filename:   "rendered.mp4",
			Resolution: export.Resolution720p,
			FrameRate:  export.FPS30,
			Quality:    export.QualityStandard,
			Method:     export.MethodServer,
		},
		CreatedAt: time.Now(),
	}

	scratchDir := filepath.Join(tempDir, "scratch")
	prober := ffmpeg.NewProber("ffprobe")
	runner := ffmpeg.NewRunner("ffmpeg", prober, diskStore{})

	var progressReports []float64
	outPath, err := runner.Render(context.Background(), manifest, scratchDir, func(p float64) {
		progressReports = append(progressReports, p)
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if len(progressReports) == 0 {
		t.Fatal("expected at least one progress report")
	}

	// 4. Verify output file with Prober
	probeResult, err := prober.Probe(context.Background(), outPath)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}

	if len(probeResult.Streams) < 2 {
		t.Fatalf("expected at least 2 streams (video and audio), got %d", len(probeResult.Streams))
	}

	hasV := false
	hasA := false
	for _, s := range probeResult.Streams {
		if s.CodecType == "video" {
			hasV = true
			if s.CodecName != "h264" {
				t.Fatalf("expected h264 video, got %s", s.CodecName)
			}
			if s.Width != 1280 || s.Height != 720 {
				t.Fatalf("expected 1280x720, got %dx%d", s.Width, s.Height)
			}
		}
		if s.CodecType == "audio" {
			hasA = true
			if s.CodecName != "aac" {
				t.Fatalf("expected aac audio, got %s", s.CodecName)
			}
		}
	}

	if !hasV || !hasA {
		t.Fatalf("missing video (%v) or audio (%v)", hasV, hasA)
	}

	// Duration should be approx 1.5s
	if probeResult.DurationSeconds < 1.3 || probeResult.DurationSeconds > 1.8 {
		t.Fatalf("expected duration ~1.5s, got %f", probeResult.DurationSeconds)
	}
}
