package timeline

import (
	"fmt"
	"math"
	"sort"
	"unicode/utf8"

	"server/internal/domain/asset"
)

// Validate checks all structural and semantic invariants of a timeline:
// 1. Project name length <= 80 runes.
// 2. Track IDs are unique and non-empty; track kinds are "video" or "audio".
// 3. Each clip references a valid registered asset.
// 4. Clip IDs are unique and non-empty.
// 5. Clip media kind matches the assigned track kind ("image" behaves as "video").
// 6. Clip bounds: At and Offset are >= 0, Length >= 1, stays within MaxFrames and source Frames.
// 7. Volume is between 0.0 and 1.0 (not NaN).
// 8. Clips on the same track do not overlap.
func Validate(t Timeline, assets map[string]asset.Asset) error {
	if t.Name == "" || utf8.RuneCountInString(t.Name) > 80 {
		return fmt.Errorf("enter a project name of up to 80 characters")
	}

	tracks := make(map[string]string, len(t.Tracks))
	for _, tr := range t.Tracks {
		if tr.ID == "" || tracks[tr.ID] != "" || (tr.Kind != "video" && tr.Kind != "audio") {
			return fmt.Errorf("invalid or duplicate track")
		}
		tracks[tr.ID] = tr.Kind
	}

	ids := make(map[string]bool, len(t.Clips))
	lanes := make(map[string][]Clip, len(t.Tracks))

	for _, c := range t.Clips {
		a, ok := assets[c.MediaID]
		if !ok {
			return fmt.Errorf("asset is not registered")
		}

		kind := a.Kind
		if kind == "image" {
			kind = "video"
		}

		if c.ID == "" || ids[c.ID] {
			return fmt.Errorf("invalid or duplicate clip ID")
		}
		ids[c.ID] = true

		if tracks[c.TrackID] != kind {
			return fmt.Errorf("track does not match the media type")
		}

		if c.At < 0 || c.At >= MaxFrames ||
			c.Offset < 0 || c.Offset >= a.Frames ||
			c.Length < 1 ||
			c.Length > MaxFrames-c.At ||
			c.Length > a.Frames-c.Offset {
			return fmt.Errorf("edit exceeds the source or timeline limits")
		}

		if math.IsNaN(c.Volume) || c.Volume < 0 || c.Volume > 1 {
			return fmt.Errorf("volume must be between 0%% and 100%%")
		}

		lanes[c.TrackID] = append(lanes[c.TrackID], c)
	}

	// Verify no two clips on the same track overlap
	for _, tr := range t.Tracks {
		lane := lanes[tr.ID]
		if len(lane) <= 1 {
			continue
		}
		sort.Slice(lane, func(i, j int) bool { return lane[i].At < lane[j].At })
		for i := 1; i < len(lane); i++ {
			if lane[i].At < lane[i-1].At+lane[i-1].Length {
				return fmt.Errorf("clips on the same track cannot overlap")
			}
		}
	}

	return nil
}
