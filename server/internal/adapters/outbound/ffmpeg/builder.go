package ffmpeg

import (
	"fmt"
	"sort"
	"strings"

	"server/internal/domain/asset"
	"server/internal/domain/export"
	"server/internal/domain/timeline"
)

// PipelineConfig contains parameters for the FFmpeg command generation.
type PipelineConfig struct {
	Manifest       export.Manifest
	ScratchDir     string
	SourcePaths    map[string]string // assetID -> local staged path
	SourceHasAudio map[string]bool   // assetID -> whether source contains an audio stream
	SourceHasVideo map[string]bool   // assetID -> whether source contains a video stream
	OutputPath     string
	FFmpegBin      string
	ProgressSocket string // or pipe:1
}

// Stage1State holds the reconstructed state from Stage 1.
type Stage1State struct {
	TotalFrames     int
	DurationSeconds float64
	Width           int
	Height          int
	OutputFPS       int
	OrderedClips    []timeline.Clip
	VideoClips      []timeline.Clip
	AudioClips      []timeline.Clip
}

// Stage1ReconstructState parses the manifest timeline and calculates output geometry and duration.
func Stage1ReconstructState(manifest export.Manifest, hasAudio, hasVideo map[string]bool) Stage1State {
	trackOrder := make(map[string]int, len(manifest.Timeline.Tracks))
	for i, tr := range manifest.Timeline.Tracks {
		trackOrder[tr.ID] = i
	}

	ordered := make([]timeline.Clip, len(manifest.Timeline.Clips))
	copy(ordered, manifest.Timeline.Clips)
	sort.Slice(ordered, func(i, j int) bool {
		return trackOrder[ordered[i].TrackID] < trackOrder[ordered[j].TrackID]
	})

	maxFrames := 1
	var videoClips, audioClips []timeline.Clip

	for _, clip := range ordered {
		end := clip.At + clip.Length
		if end > maxFrames {
			maxFrames = end
		}
		a, ok := manifest.Assets[clip.MediaID]
		if !ok {
			continue
		}
		if a.Kind == "image" || (a.Kind == "video" && (hasVideo == nil || hasVideo[clip.MediaID])) {
			videoClips = append(videoClips, clip)
		}
		// Audio clips: either pure audio kind (if has audio), OR video clip that ACTUALLY contains audio
		hasAudioStream := true
		if hasAudio != nil {
			hasAudioStream = hasAudio[clip.MediaID]
		}
		if (a.Kind == "audio" || a.Kind == "video") && hasAudioStream && clip.Volume > 0 {
			audioClips = append(audioClips, clip)
		}
	}

	w, h := manifest.Settings.Resolution.Dimensions()
	fps := int(manifest.Settings.FrameRate)
	if fps <= 0 {
		fps = 30
	}

	durSec := float64(maxFrames) / 30.0

	return Stage1State{
		TotalFrames:     maxFrames,
		DurationSeconds: durSec,
		Width:           w,
		Height:          h,
		OutputFPS:       fps,
		OrderedClips:    ordered,
		VideoClips:      videoClips,
		AudioClips:      audioClips,
	}
}

// BuildFFmpegArgs generates the complete command line arguments for stages 3 through 9.
func BuildFFmpegArgs(cfg PipelineConfig) ([]string, error) {
	st1 := Stage1ReconstructState(cfg.Manifest, cfg.SourceHasAudio, cfg.SourceHasVideo)

	// Identify unique inputs in stable order
	uniqueAssetIDs := make([]string, 0, len(cfg.SourcePaths))
	for id := range cfg.SourcePaths {
		uniqueAssetIDs = append(uniqueAssetIDs, id)
	}
	sort.Strings(uniqueAssetIDs)

	assetIndex := make(map[string]int, len(uniqueAssetIDs))
	var inputArgs []string

	for idx, id := range uniqueAssetIDs {
		assetIndex[id] = idx
		path := cfg.SourcePaths[id]
		a := cfg.Manifest.Assets[id]

		if a.Kind == "image" {
			inputArgs = append(inputArgs, "-loop", "1", "-framerate", "30", "-i", path)
		} else {
			inputArgs = append(inputArgs, "-i", path)
		}
	}

	// Build filter_complex
	filterGraph, videoOutLabel, audioOutLabel := buildFilterComplex(st1, cfg.Manifest.Assets, assetIndex)

	var args []string
	args = append(args, "-y") // overwrite output
	args = append(args, "-hide_banner", "-nostats")
	args = append(args, "-progress", "pipe:1")
	args = append(args, inputArgs...)
	args = append(args, "-filter_complex", filterGraph)
	args = append(args, "-map", videoOutLabel)
	args = append(args, "-map", audioOutLabel)

	// Stage 8: Encoding settings
	args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-r", fmt.Sprintf("%d", st1.OutputFPS))
	if cfg.Manifest.Settings.Quality == export.QualityHigh {
		args = append(args, "-crf", "18", "-preset", "slow", "-c:a", "aac", "-b:a", "256k")
	} else {
		args = append(args, "-crf", "23", "-preset", "medium", "-c:a", "aac", "-b:a", "192k")
	}

	// Stage 9: Muxing settings
	args = append(args, "-f", "mp4", "-movflags", "+faststart")
	args = append(args, "-t", fmt.Sprintf("%.4f", st1.DurationSeconds))
	args = append(args, cfg.OutputPath)

	return args, nil
}

func buildFilterComplex(st1 Stage1State, assets map[string]asset.Asset, assetIdx map[string]int) (string, string, string) {
	var filterParts []string

	// Stage 6: Compositing base solid canvas #050607
	baseVideoLabel := "[bg_base]"
	filterParts = append(filterParts, fmt.Sprintf(
		"color=c=#050607:s=%dx%d:d=%.4f:r=%d%s",
		st1.Width, st1.Height, st1.DurationSeconds, st1.OutputFPS, baseVideoLabel,
	))

	// Stage 5: Processing & Stage 6: Compositing for Video
	currentVideoBg := baseVideoLabel
	for i, clip := range st1.VideoClips {
		idx := assetIdx[clip.MediaID]
		a := assets[clip.MediaID]

		offsetSec := float64(clip.Offset) / 30.0
		durSec := float64(clip.Length) / 30.0
		atSec := float64(clip.At) / 30.0

		clipOutLabel := fmt.Sprintf("[vclip_%d]", i)
		nextBgLabel := fmt.Sprintf("[bg_%d]", i)
		if i == len(st1.VideoClips)-1 {
			nextBgLabel = "[outv]"
		}

		if a.Kind == "image" {
			filterParts = append(filterParts, fmt.Sprintf(
				"[%d:v]trim=duration=%.4f,scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=#050607,setpts=PTS-STARTPTS+%.4f/TB%s",
				idx, durSec, st1.Width, st1.Height, st1.Width, st1.Height, atSec, clipOutLabel,
			))
		} else {
			filterParts = append(filterParts, fmt.Sprintf(
				"[%d:v]trim=start=%.4f:duration=%.4f,scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=#050607,setpts=PTS-STARTPTS+%.4f/TB%s",
				idx, offsetSec, durSec, st1.Width, st1.Height, st1.Width, st1.Height, atSec, clipOutLabel,
			))
		}

		filterParts = append(filterParts, fmt.Sprintf(
			"%s%soverlay=eof_action=pass%s",
			currentVideoBg, clipOutLabel, nextBgLabel,
		))
		currentVideoBg = nextBgLabel
	}

	videoOut := currentVideoBg
	if len(st1.VideoClips) == 0 {
		videoOut = "[outv]"
		// Connect base directly
		filterParts[0] = fmt.Sprintf(
			"color=c=#050607:s=%dx%d:d=%.4f:r=%d[outv]",
			st1.Width, st1.Height, st1.DurationSeconds, st1.OutputFPS,
		)
	}

	// Stage 7: Audio Mixing
	// Base silence source of exact timeline duration
	silenceLabel := "[base_silence]"
	filterParts = append(filterParts, fmt.Sprintf(
		"aevalsrc=0:d=%.4f:s=48000%s",
		st1.DurationSeconds, silenceLabel,
	))

	var audioLabels []string
	audioLabels = append(audioLabels, silenceLabel)

	for i, clip := range st1.AudioClips {
		idx := assetIdx[clip.MediaID]
		offsetSec := float64(clip.Offset) / 30.0
		durSec := float64(clip.Length) / 30.0
		atMs := int(float64(clip.At) / 30.0 * 1000.0)

		audioClipLabel := fmt.Sprintf("[aclip_%d]", i)
		filterParts = append(filterParts, fmt.Sprintf(
			"[%d:a]atrim=start=%.4f:duration=%.4f,asetpts=PTS-STARTPTS,adelay=%d|%d,volume=%.4f%s",
			idx, offsetSec, durSec, atMs, atMs, clip.Volume, audioClipLabel,
		))
		audioLabels = append(audioLabels, audioClipLabel)
	}

	audioOut := "[outa]"
	if len(st1.AudioClips) == 0 {
		// Pure silence
		audioOut = "[outa]"
		// Replace base silence label
		lastIdx := len(filterParts) - 1
		filterParts[lastIdx] = fmt.Sprintf("aevalsrc=0:d=%.4f:s=48000[outa]", st1.DurationSeconds)
	} else {
		// Mix all active audio with silence
		filterParts = append(filterParts, fmt.Sprintf(
			"%samix=inputs=%d:duration=first:dropout_transition=0:normalize=0%s",
			strings.Join(audioLabels, ""), len(audioLabels), audioOut,
		))
	}

	filterGraph := strings.Join(filterParts, ";\n")
	return filterGraph, videoOut, audioOut
}
