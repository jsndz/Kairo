package storage_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"server/internal/domain/asset"
	"server/internal/domain/record"
	"server/internal/adapters/outbound/storage/catalog"
	"server/internal/adapters/outbound/storage/fs"
	"server/internal/adapters/outbound/storage/jsonl"
)

func TestJSONLAppendAndRead(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "operations.jsonl")
	l := jsonl.JSONL{Path: logPath}

	records, err := l.Read()
	if err != nil {
		t.Fatalf("expected nil error on empty log, got %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}

	r1 := record.Record{
		ID:        "op-1",
		ProjectID: "proj-1",
		Sequence:  1,
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		Type:      "rename",
		Payload:   []byte(`{"name":"New Name"}`),
	}
	r2 := record.Record{
		ID:        "op-2",
		ProjectID: "proj-1",
		Sequence:  2,
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		Type:      "reset",
		Payload:   []byte(`{}`),
	}

	if err := l.Append(r1); err != nil {
		t.Fatalf("append r1 failed: %v", err)
	}
	if err := l.Append(r2); err != nil {
		t.Fatalf("append r2 failed: %v", err)
	}

	records, err = l.Read()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Trailing corrupt row truncation check
	f, _ := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0600)
	_, _ = f.WriteString("{\"id\":\"corrupt-half-line\"")
	f.Close()

	records, err = l.Read()
	if err != nil {
		t.Fatalf("expected recovery, got %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records after repair, got %d", len(records))
	}
}

func TestCatalogSaveAndRead(t *testing.T) {
	dir := t.TempDir()
	catPath := filepath.Join(dir, "metadata.json")
	cat := catalog.JSONCatalog{Path: catPath}

	meta := record.Metadata{
		ID: "proj-1",
		Assets: map[string]asset.Asset{
			"a-1": {
				ID:        "a-1",
				Bucket:    "bucket",
				ObjectKey: "key",
				Status:    "completed",
				Source: asset.Source{
					Name:     "clip.mp4",
					Kind:     "video",
					MIME:     "video/mp4",
					Bytes:    1024,
					Frames:   30,
					Duration: 1.0,
				},
			},
		},
	}

	if err := cat.Save(meta); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	read, err := cat.Read()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if read.ID != meta.ID {
		t.Fatalf("id mismatch: %q vs %q", read.ID, meta.ID)
	}
	if !reflect.DeepEqual(read.Assets["a-1"].Source, meta.Assets["a-1"].Source) {
		t.Fatalf("source mismatch")
	}

	if err := fs.SyncDir(dir); err != nil {
		t.Fatalf("SyncDir failed: %v", err)
	}
}
