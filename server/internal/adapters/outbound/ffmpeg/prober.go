package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"server/internal/ports"
)

var _ ports.MediaProber = (*Prober)(nil)

// Prober inspects media containers and streams using ffprobe.
type Prober struct {
	ffprobePath string
}

// NewProber initializes an ffprobe inspector.
func NewProber(ffprobePath string) *Prober {
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	return &Prober{ffprobePath: ffprobePath}
}

type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		FrameRate string `json:"r_frame_rate"`
	} `json:"streams"`
}

// Probe runs ffprobe to inspect video and audio streams and duration.
func (p *Prober) Probe(ctx context.Context, path string) (ports.ProbeResult, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration:stream=codec_type,codec_name,width,height,r_frame_rate",
		"-of", "json",
		path,
	}

	cmd := exec.CommandContext(ctx, p.ffprobePath, args...)
	out, err := cmd.Output()
	if err != nil {
		return ports.ProbeResult{}, fmt.Errorf("ffprobe probe failed: %w", err)
	}

	var parsed ffprobeOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return ports.ProbeResult{}, fmt.Errorf("parse ffprobe json: %w", err)
	}

	dur, _ := strconv.ParseFloat(parsed.Format.Duration, 64)

	res := ports.ProbeResult{
		DurationSeconds: dur,
	}

	for _, s := range parsed.Streams {
		res.Streams = append(res.Streams, ports.ProbeStream{
			CodecType: s.CodecType,
			CodecName: s.CodecName,
			Width:     s.Width,
			Height:    s.Height,
			FrameRate: s.FrameRate,
		})
	}

	return res, nil
}
