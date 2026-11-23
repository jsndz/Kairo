package timeline_test

import (
	"encoding/json"
	"testing"

	"server/internal/domain/asset"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
)

func TestAddAndDeleteTrack(t *testing.T) {
	tl := timeline.Timeline{
		Name: "Project",
		Tracks: []timeline.Track{
			{ID: "track-1", Kind: "video"},
		},
		Clips: []timeline.Clip{
			{ID: "c1", TrackID: "track-1", MediaID: "m1", At: 0, Offset: 0, Length: 50, Volume: 1.0},
		},
	}
	assets := map[string]asset.Asset{
		"m1": {ID: "m1", Kind: "video", Frames: 100},
		"m2": {ID: "m2", Kind: "audio", Frames: 100},
	}

	// 1. Add audio track
	addAudioOp := record.Record{
		ID:      "op-1",
		Type:    "add-track",
		Payload: json.RawMessage(`{"trackId":"audio-1","kind":"audio"}`),
	}
	next, err := timeline.Apply(tl, assets, addAudioOp)
	if err != nil {
		t.Fatalf("Add audio track failed: %v", err)
	}
	if len(next.Tracks) != 2 || next.Tracks[1].Kind != "audio" || next.Tracks[1].ID != "audio-1" {
		t.Fatalf("Unexpected tracks after add-track: %+v", next.Tracks)
	}

	// 2. Add clip on audio track
	addClipOp := record.Record{
		ID:   "op-2",
		Type: "add",
		Payload: json.RawMessage(`{"clip":{"id":"c2","trackId":"audio-1","mediaId":"m2","at":0,"offset":0,"length":50,"volume":0.8}}`),
	}
	next, err = timeline.Apply(next, assets, addClipOp)
	if err != nil {
		t.Fatalf("Add clip failed: %v", err)
	}
	if len(next.Clips) != 2 {
		t.Fatalf("Expected 2 clips, got %d", len(next.Clips))
	}

	// 3. Delete track-1 (cascading clip c1)
	delTrackOp := record.Record{
		ID:      "op-3",
		Type:    "delete-track",
		Payload: json.RawMessage(`{"trackId":"track-1"}`),
	}
	next, err = timeline.Apply(next, assets, delTrackOp)
	if err != nil {
		t.Fatalf("Delete track failed: %v", err)
	}
	if len(next.Tracks) != 1 || next.Tracks[0].ID != "audio-1" {
		t.Fatalf("Unexpected tracks after delete-track: %+v", next.Tracks)
	}
	if len(next.Clips) != 1 || next.Clips[0].ID != "c2" {
		t.Fatalf("Expected only c2 remaining, got %+v", next.Clips)
	}

	// 4. Attempt to delete last remaining track (should fail)
	delLastOp := record.Record{
		ID:      "op-4",
		Type:    "delete-track",
		Payload: json.RawMessage(`{"trackId":"audio-1"}`),
	}
	_, err = timeline.Apply(next, assets, delLastOp)
	if err == nil {
		t.Fatalf("Expected error deleting last track, got nil")
	}
}
