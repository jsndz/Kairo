package timeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"server/internal/domain/asset"
	"server/internal/domain/record"
)

// DecodePayload validates required fields and unmarshals payload.
func DecodePayload(raw json.RawMessage, dst any, required ...string) error {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil || len(fields) < len(required) {
		return fmt.Errorf("invalid operation payload")
	}
	for _, k := range required {
		if v, ok := fields[k]; !ok || string(v) == "null" {
			return fmt.Errorf("missing payload field %s", k)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func applyRelink(t Timeline, assets map[string]asset.Asset, mediaID, replacementID string) (Timeline, error) {
	source, ok := assets[mediaID]
	target, exists := assets[replacementID]
	if !ok || !exists || source.Kind != target.Kind || target.Frames < source.Frames || mediaID == replacementID {
		return t, fmt.Errorf("choose a new source of the same type and at least the original duration")
	}

	next := t
	next.Clips = slices.Clone(t.Clips)
	for i := range next.Clips {
		if next.Clips[i].MediaID == mediaID {
			next.Clips[i].MediaID = replacementID
		}
	}
	return next, nil
}

func applyClipEdit(t Timeline, opType string, p opParams) (Timeline, error) {
	i := slices.IndexFunc(t.Clips, func(c Clip) bool { return c.ID == p.ClipID })
	if i < 0 {
		return t, fmt.Errorf("clip is not on the timeline")
	}

	next := t
	next.Clips = slices.Clone(t.Clips)
	c := &next.Clips[i]

	switch opType {
	case "move":
		c.At = p.At
		c.TrackID = p.TrackID

	case "trim":
		c.At = p.At
		c.Offset = p.Offset
		c.Length = p.Length

	case "volume":
		c.Volume = p.Volume

	case "delete":
		next.Clips = slices.Delete(next.Clips, i, i+1)

	case "split":
		if p.Frame <= c.At || p.Frame >= c.At+c.Length {
			return t, fmt.Errorf("move the playhead inside the clip to split it")
		}
		left := p.Frame - c.At
		right := *c
		right.ID = p.RightID
		right.At = p.Frame
		right.Offset = c.Offset + left
		right.Length = c.Length - left

		c.Length = left
		next.Clips = slices.Insert(next.Clips, i+1, right)
	}

	return next, nil
}

// opParams holds unmarshaled operation parameters.
type opParams struct {
	Name          string  `json:"name"`
	TrackID       string  `json:"trackId"`
	Kind          string  `json:"kind"`
	ClipID        string  `json:"clipId"`
	Clip          *Clip   `json:"clip"`
	At            int     `json:"at"`
	Offset        int     `json:"offset"`
	Length        int     `json:"length"`
	Volume        float64 `json:"volume"`
	Frame         int     `json:"frame"`
	RightID       string  `json:"rightId"`
	MediaID       string  `json:"mediaId"`
	ReplacementID string  `json:"replacementId"`
}

// requiredFields maps operation types to mandatory JSON keys.
var requiredFields = map[string][]string{
	"reset":        {},
	"rename":       {"name"},
	"add-track":    {"trackId"},
	"delete-track": {"trackId"},
	"add":          {"clip"},
	"move":         {"clipId", "trackId", "at"},
	"trim":         {"clipId", "at", "offset", "length"},
	"split":        {"clipId", "frame", "rightId"},
	"delete":       {"clipId"},
	"volume":       {"clipId", "volume"},
	"relink":       {"mediaId", "replacementId"},
}

func dispatch(t Timeline, assets map[string]asset.Asset, opType string, raw json.RawMessage, p opParams) (Timeline, error) {
	switch opType {
	case "reset":
		return Empty(), nil

	case "rename":
		next := t
		next.Name = strings.TrimSpace(p.Name)
		return next, nil

	case "add-track":
		kind := p.Kind
		if kind != "audio" {
			kind = "video"
		}
		next := t
		next.Tracks = append(slices.Clone(t.Tracks), Track{ID: p.TrackID, Kind: kind})
		return next, nil

	case "delete-track":
		if len(t.Tracks) <= 1 {
			return t, fmt.Errorf("cannot delete the only remaining track")
		}
		var found bool
		for _, tr := range t.Tracks {
			if tr.ID == p.TrackID {
				found = true
				break
			}
		}
		if !found {
			return t, fmt.Errorf("track not found: %s", p.TrackID)
		}
		next := t
		next.Tracks = slices.DeleteFunc(slices.Clone(t.Tracks), func(tr Track) bool {
			return tr.ID == p.TrackID
		})
		next.Clips = slices.DeleteFunc(slices.Clone(t.Clips), func(c Clip) bool {
			return c.TrackID == p.TrackID
		})
		return next, nil

	case "add":
		var wrapper map[string]json.RawMessage
		_ = json.Unmarshal(raw, &wrapper)
		if err := DecodePayload(wrapper["clip"], p.Clip, "id", "trackId", "mediaId", "at", "offset", "length", "volume"); err != nil {
			return t, err
		}
		next := t
		next.Clips = append(slices.Clone(t.Clips), *p.Clip)
		return next, nil

	case "relink":
		return applyRelink(t, assets, p.MediaID, p.ReplacementID)

	default:
		return applyClipEdit(t, opType, p)
	}
}

// Apply applies an operation record to the timeline.
func Apply(t Timeline, assets map[string]asset.Asset, op record.Record) (Timeline, error) {
	required, ok := requiredFields[op.Type]
	if !ok {
		return t, fmt.Errorf("unknown operation type: %s", op.Type)
	}

	var p opParams
	if err := DecodePayload(op.Payload, &p, required...); err != nil {
		return t, err
	}

	next, err := dispatch(t, assets, op.Type, op.Payload, p)
	if err != nil {
		return t, err
	}

	if err := Validate(next, assets); err != nil {
		return t, err
	}

	return next, nil
}

// ExtractClipID extracts targeted clip ID from an edit payload.
func ExtractClipID(e Edit) (string, bool) {
	switch e.Type {
	case "move", "trim", "split", "delete", "volume":
		var p struct {
			ClipID    string `json:"clipId"`
			ClipIDAlt string `json:"clip_id"`
		}
		if err := json.Unmarshal(e.Payload, &p); err == nil {
			if p.ClipID != "" {
				return p.ClipID, true
			}
			if p.ClipIDAlt != "" {
				return p.ClipIDAlt, true
			}
		}
	case "add":
		var p struct {
			Clip struct {
				ID string `json:"id"`
			} `json:"clip"`
		}
		if err := json.Unmarshal(e.Payload, &p); err == nil && p.Clip.ID != "" {
			return p.Clip.ID, true
		}
	}
	return "", false
}
