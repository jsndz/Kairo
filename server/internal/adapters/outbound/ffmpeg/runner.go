package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"server/internal/domain/export"
	"server/internal/ports"
)

var _ ports.MediaRenderer = (*Runner)(nil)

// Runner manages FFmpeg pipeline execution and progress monitoring.
type Runner struct {
	ffmpegPath  string
	prober      ports.MediaProber
	objectStore ports.ObjectStore
}

// NewRunner initializes an FFmpeg media renderer.
func NewRunner(ffmpegPath string, prober ports.MediaProber, store ports.ObjectStore) *Runner {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &Runner{
		ffmpegPath:  ffmpegPath,
		prober:      prober,
		objectStore: store,
	}
}

// Render downloads source assets to scratchDir and runs stages 3 through 9 via FFmpeg.
func (r *Runner) Render(ctx context.Context, manifest export.Manifest, scratchDir string, onProgress func(percent float64)) (string, error) {
	sourcesDir := filepath.Join(scratchDir, "sources")
	if err := os.MkdirAll(sourcesDir, 0700); err != nil {
		return "", fmt.Errorf("create sources dir: %w", err)
	}

	// Stage 2: Asset Retrieval
	sourcePaths := make(map[string]string)
	sourceHasAudio := make(map[string]bool)
	sourceHasVideo := make(map[string]bool)

	for id, a := range manifest.Assets {
		ext := filepath.Ext(a.Name)
		if ext == "" {
			ext = ".media"
		}
		localPath := filepath.Join(sourcesDir, id+ext)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			if err := r.objectStore.Download(ctx, a.ObjectKey, localPath); err != nil {
				return "", fmt.Errorf("retrieve asset %s: %w", id, err)
			}
		}
		sourcePaths[id] = localPath

		// Probe the local file to check for audio and video streams
		if r.prober != nil {
			probeRes, err := r.prober.Probe(ctx, localPath)
			if err == nil {
				for _, stream := range probeRes.Streams {
					if stream.CodecType == "audio" {
						sourceHasAudio[id] = true
					}
					if stream.CodecType == "video" {
						sourceHasVideo[id] = true
					}
				}
			} else {
				sourceHasAudio[id] = (a.Kind == "audio" || a.Kind == "video")
				sourceHasVideo[id] = (a.Kind == "video" || a.Kind == "image")
			}
		} else {
			sourceHasAudio[id] = (a.Kind == "audio" || a.Kind == "video")
			sourceHasVideo[id] = (a.Kind == "video" || a.Kind == "image")
		}
	}

	outputPath := filepath.Join(scratchDir, manifest.Settings.Filename)

	cfg := PipelineConfig{
		Manifest:       manifest,
		ScratchDir:     scratchDir,
		SourcePaths:    sourcePaths,
		SourceHasAudio: sourceHasAudio,
		SourceHasVideo: sourceHasVideo,
		OutputPath:     outputPath,
		FFmpegBin:      r.ffmpegPath,
	}

	args, err := BuildFFmpegArgs(cfg)
	if err != nil {
		return "", fmt.Errorf("build ffmpeg args: %w", err)
	}

	cmd := exec.CommandContext(ctx, r.ffmpegPath, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start ffmpeg: %w", err)
	}

	st1 := Stage1ReconstructState(manifest, sourceHasAudio, sourceHasVideo)
	totalDurationUs := int64(st1.DurationSeconds * 1_000_000)
	if totalDurationUs <= 0 {
		totalDurationUs = 1
	}

	// Read progress in background
	scanner := bufio.NewScanner(stdoutPipe)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "out_time_us=") {
				valStr := strings.TrimPrefix(line, "out_time_us=")
				if us, err := strconv.ParseInt(valStr, 10, 64); err == nil && totalDurationUs > 0 {
					p := float64(us) / float64(totalDurationUs) * 100.0
					if p > 100.0 {
						p = 100.0
					}
					if p < 0.0 {
						p = 0.0
					}
					if onProgress != nil {
						onProgress(p)
					}
				}
			}
		}
	}()

	var stderrBuf strings.Builder
	go func() {
		errScanner := bufio.NewScanner(stderrPipe)
		for errScanner.Scan() {
			stderrBuf.WriteString(errScanner.Text() + "\n")
		}
	}()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("ffmpeg execution failed: %w (details: %s)", err, stderrBuf.String())
	}

	if onProgress != nil {
		onProgress(100.0)
	}

	return outputPath, nil
}
