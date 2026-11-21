package timeline

import "encoding/json"

// MaxFrames defines the maximum allowable timeline frame count (30 FPS * 60s * 60min = 1 hour).
const MaxFrames = 30 * 60 * 60

// Track represents an audio or video track lane on the timeline.
type Track struct {
	ID   string `json:"id"`
	Kind string `json:"kind"` // "video" or "audio"
}

// Clip represents a media segment placed on a specific track and timeline position.
type Clip struct {
	ID      string  `json:"id"`
	TrackID string  `json:"trackId"`
	MediaID string  `json:"mediaId"`
	At      int     `json:"at"`     // Timeline start frame
	Offset  int     `json:"offset"` // Source start frame within the original asset
	Length  int     `json:"length"` // Duration in frames
	Volume  float64 `json:"volume"` // Audio volume from 0.0 to 1.0
}

// Timeline encapsulates the project name, tracks, and placed clips.
type Timeline struct {
	Name   string  `json:"name"`
	Tracks []Track `json:"tracks"`
	Clips  []Clip  `json:"clips"`
}

// Empty returns a default project timeline with one video track and one audio track.
func Empty() Timeline {
	return Timeline{
		Name: "Untitled project",
		Tracks: []Track{
			{ID: "video-1", Kind: "video"},
			{ID: "audio-1", Kind: "audio"},
		},
		Clips: []Clip{},
	}
}

// Edit represents an incoming editing command received from the client.
type Edit struct {
	ID               string          `json:"id"`
	Type             string          `json:"type"`
	Payload          json.RawMessage `json:"payload"`
	ExpectedRevision *uint64         `json:"expectedRevision"`
}
