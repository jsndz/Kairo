package sqlite_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"server/internal/adapters/outbound/storage/sqlite"
	"server/internal/domain/asset"
	"server/internal/domain/record"
)

func TestProjectDB_OperationsAndSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "project.db")

	pdb, err := sqlite.OpenProject(dbPath)
	if err != nil {
		t.Fatalf("OpenProject failed: %v", err)
	}
	defer pdb.Close()

	log := pdb.Log()

	// 1. Initial read should be empty
	records, err := log.Read()
	if err != nil {
		t.Fatalf("initial Read failed: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}

	// 2. Append operations
	for i := 1; i <= 5; i++ {
		rec := record.Record{
			ID:        "op-" + string(rune('a'+i-1)),
			ProjectID: "proj-1",
			Sequence:  uint64(i),
			Timestamp: time.Now().UTC(),
			Type:      "clip_move",
			Payload:   json.RawMessage(`{"at": 100}`),
		}
		if err := log.Append(rec); err != nil {
			t.Fatalf("Append failed for seq %d: %v", i, err)
		}
	}

	// 3. Read all operations
	records, err = log.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(records) != 5 {
		t.Fatalf("expected 5 records, got %d", len(records))
	}
	if records[0].Sequence != 1 || records[4].Sequence != 5 {
		t.Fatalf("unexpected sequence order: %d ... %d", records[0].Sequence, records[4].Sequence)
	}

	// 4. Read delta operations (sinceSeq = 3)
	deltas, err := log.ReadSince(3)
	if err != nil {
		t.Fatalf("ReadSince failed: %v", err)
	}
	if len(deltas) != 2 {
		t.Fatalf("expected 2 delta records, got %d", len(deltas))
	}
	if deltas[0].Sequence != 4 || deltas[1].Sequence != 5 {
		t.Fatalf("expected deltas 4 and 5, got %d and %d", deltas[0].Sequence, deltas[1].Sequence)
	}

	// 5. Snapshot handling
	snap, err := log.GetLatestSnapshot()
	if err != nil {
		t.Fatalf("GetLatestSnapshot failed: %v", err)
	}
	if snap != nil {
		t.Fatalf("expected nil snapshot initially, got %v", snap)
	}

	statePayload := []byte(`{"revision": 5, "timeline": {"name": "Test"}}`)
	if err := log.SaveSnapshot(5, statePayload); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	snap, err = log.GetLatestSnapshot()
	if err != nil {
		t.Fatalf("GetLatestSnapshot after save failed: %v", err)
	}
	if snap == nil || snap.Sequence != 5 {
		t.Fatalf("expected snapshot at sequence 5, got %v", snap)
	}
}

func TestProjectDB_AssetCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "project.db")

	pdb, err := sqlite.OpenProject(dbPath)
	if err != nil {
		t.Fatalf("OpenProject failed: %v", err)
	}
	defer pdb.Close()

	cat := pdb.Catalog()

	// Initial read should return os.ErrNotExist
	_, err = cat.Read()
	if err != os.ErrNotExist {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}

	now := time.Now().UTC()
	meta := record.Metadata{
		ID: "proj-123",
		Assets: map[string]asset.Asset{
			"a1": {
				ID: "a1",
				Source: asset.Source{
					Name:     "video.mp4",
					Kind:     "video",
					MIME:     "video/mp4",
					Bytes:    1048576,
					Duration: 10.5,
					Frames:   315,
					Width:    1920,
					Height:   1080,
				},
				ObjectKey: "raw/a1.mp4",
				Bucket:    "media",
				Status:    "completed",
				CreatedAt: now,
			},
		},
	}

	if err := cat.Save(meta); err != nil {
		t.Fatalf("Save metadata failed: %v", err)
	}

	readMeta, err := cat.Read()
	if err != nil {
		t.Fatalf("Read metadata failed: %v", err)
	}
	if readMeta.ID != "proj-123" {
		t.Fatalf("expected id proj-123, got %s", readMeta.ID)
	}
	if len(readMeta.Assets) != 1 || readMeta.Assets["a1"].Name != "video.mp4" {
		t.Fatalf("unexpected assets: %+v", readMeta.Assets)
	}
}

func TestIndexDB(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "index.db")

	idx, err := sqlite.OpenIndex(indexPath)
	if err != nil {
		t.Fatalf("OpenIndex failed: %v", err)
	}
	defer idx.Close()

	now := time.Now().UTC()
	s1 := record.ProjectMetadata{
		ID:        "p1",
		Name:      "Project One",
		Revision:  10,
		Clips:     3,
		Assets:    2,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s2 := record.ProjectMetadata{
		ID:        "p2",
		Name:      "Project Two",
		Revision:  5,
		Clips:     1,
		Assets:    1,
		CreatedAt: now,
		UpdatedAt: now.Add(time.Minute),
	}

	if err := idx.Save([]record.ProjectMetadata{s1, s2}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	summaries, err := idx.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	// Ordered by updated_at DESC, so p2 comes first
	if summaries[0].ID != "p2" || summaries[1].ID != "p1" {
		t.Fatalf("expected [p2, p1], got [%s, %s]", summaries[0].ID, summaries[1].ID)
	}

	// Upsert update
	s1.Name = "Project One Updated"
	s1.UpdatedAt = now.Add(2 * time.Minute)
	if err := idx.Upsert(s1); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	summaries, err = idx.Read()
	if err != nil {
		t.Fatalf("Read after upsert failed: %v", err)
	}
	if summaries[0].ID != "p1" || summaries[0].Name != "Project One Updated" {
		t.Fatalf("expected p1 updated first, got %+v", summaries[0])
	}

	// Delete
	if err := idx.Delete("p1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	summaries, err = idx.Read()
	if err != nil {
		t.Fatalf("Read after delete failed: %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != "p2" {
		t.Fatalf("expected only p2, got %+v", summaries)
	}
}
