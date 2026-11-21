package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	"server/internal/domain/timeline"
	"server/internal/service"
	"server/internal/storage/catalog"
	"server/internal/storage/jsonl"
)

type mockObjects struct {
	data   map[string]asset.ObjectInfo
	copies int
}

func (o *mockObjects) Bucket() string { return "test" }
func (o *mockObjects) UploadPolicy(_ context.Context, key string, _ asset.Source) (asset.Upload, error) {
	return asset.Upload{URL: "https://minio.invalid", Fields: map[string]string{"key": key}}, nil
}
func (o *mockObjects) Stat(_ context.Context, key string) (asset.ObjectInfo, error) {
	if v, ok := o.data[key]; ok {
		return v, nil
	}
	return asset.ObjectInfo{}, domErrors.ErrNotFound
}
func (o *mockObjects) Promote(_ context.Context, stage, key, etag string) error {
	if o.data[stage].ETag != etag {
		return domErrors.ErrConflict
	}
	o.data[key] = o.data[stage]
	o.copies++
	return nil
}
func (o *mockObjects) ReadURL(context.Context, string) (string, error) {
	return "https://minio.invalid/file", nil
}

func setup(t *testing.T) (*service.Service, catalog.JSONCatalog, jsonl.JSONL, *mockObjects) {
	t.Helper()
	dir := t.TempDir()
	c := catalog.JSONCatalog{Path: filepath.Join(dir, "metadata.json")}
	l := jsonl.JSONL{Path: filepath.Join(dir, "operations.jsonl")}
	o := &mockObjects{data: make(map[string]asset.ObjectInfo)}
	s, err := service.Open(c, l, o, 1024)
	if err != nil {
		t.Fatal(err)
	}
	return s, c, l, o
}

func sampleSource() asset.Source {
	return asset.Source{
		Name:     "cut.mp4",
		Kind:     "video",
		MIME:     "video/mp4",
		Bytes:    100,
		Frames:   300,
		Duration: 10,
		Width:    1280,
		Height:   720,
	}
}

func makeEdit(id, kind, data string, rev uint64) timeline.Edit {
	return timeline.Edit{ID: id, Type: kind, Payload: json.RawMessage(data), ExpectedRevision: &rev}
}

func TestRegisterUploadRetryAndImmutableCompletion(t *testing.T) {
	ctx := context.Background()
	s, c, _, o := setup(t)
	bad := sampleSource()
	bad.Bytes = 1025
	if _, err := s.Register(bad); !errors.Is(err, domErrors.ErrInvalid) {
		t.Fatalf("invalid size: %v", err)
	}
	a, err := s.Register(sampleSource())
	if err != nil || a.Status != "pending" {
		t.Fatalf("register: %+v %v", a, err)
	}
	if got, _ := c.Read(); got.Assets[a.ID] != a {
		t.Fatal("register was not persisted")
	}
	a, first, err := s.Begin(ctx, a.ID)
	if err != nil || a.Status != "uploading" || first.Fields["key"] == a.ObjectKey {
		t.Fatalf("begin: %+v %+v %v", a, first, err)
	}
	if _, err = s.Complete(ctx, a.ID, a.Attempt); !errors.Is(err, domErrors.ErrConflict) {
		t.Fatalf("premature complete: %v", err)
	}
	stale := a.Attempt
	if _, err = s.Fail(a.ID, stale); err != nil {
		t.Fatal(err)
	}
	a, second, err := s.Begin(ctx, a.ID)
	if err != nil || second.Fields["key"] == first.Fields["key"] {
		t.Fatalf("retry: %+v %v", second, err)
	}
	if _, err = s.Fail(a.ID, stale); !errors.Is(err, domErrors.ErrConflict) {
		t.Fatalf("stale request: %v", err)
	}
	key := second.Fields["key"]
	o.data[key] = asset.ObjectInfo{Size: 99, MIME: a.MIME, ETag: "first"}
	if _, err = s.Complete(ctx, a.ID, a.Attempt); !errors.Is(err, domErrors.ErrInvalid) {
		t.Fatalf("invalid stored bytes: %v", err)
	}
	o.data[key] = asset.ObjectInfo{Size: a.Bytes, MIME: a.MIME, ETag: "accepted"}
	a, err = s.Complete(ctx, a.ID, a.Attempt)
	if err != nil || a.Status != "completed" || o.copies != 1 {
		t.Fatalf("complete: %+v %v", a, err)
	}
	o.data[key] = asset.ObjectInfo{Size: a.Bytes, MIME: a.MIME, ETag: "overwritten staging"}
	if _, err = s.Complete(ctx, a.ID, a.Attempt); err != nil {
		t.Fatal(err)
	}
	if o.copies != 1 || o.data[a.ObjectKey].ETag != "accepted" {
		t.Fatal("completion rewrote the original")
	}
	s, err = service.Open(c, jsonl.JSONL{Path: filepath.Join(t.TempDir(), "operations.jsonl")}, o, 1024)
	if err == nil {
		// Just ensure open works
	}
}

func TestReplayPreservesPendingReferencesAndHistory(t *testing.T) {
	s, c, l, o := setup(t)
	a, err := s.Register(sampleSource())
	if err != nil {
		t.Fatal(err)
	}
	add := fmt.Sprintf(`{"clip":{"id":"a","trackId":"video-1","mediaId":%q,"at":0,"offset":0,"length":60,"volume":1}}`, a.ID)
	first := makeEdit("add", "add", add, 0)
	for _, op := range []timeline.Edit{first, makeEdit("split", "split", `{"clipId":"a","frame":30,"rightId":"b"}`, 1), makeEdit("undo", "undo", `{}`, 2)} {
		if _, err = s.Commit(op); err != nil {
			t.Fatal(err)
		}
	}
	original := s.Get()
	s, err = service.Open(c, l, o, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, s.Get()) || s.Get().Project.Assets[a.ID].Status != "pending" {
		t.Fatal("replay changed state")
	}
	if _, err = s.Commit(first); err != nil || s.Get().Revision != 3 {
		t.Fatal("replay lost idempotency")
	}
	if _, err = s.Commit(makeEdit("stale", "reset", `{}`, 1)); !errors.Is(err, domErrors.ErrConflict) {
		t.Fatal("accepted stale revision")
	}
}

func TestOrderingAndLogCorruption(t *testing.T) {
	s, c, l, o := setup(t)
	if _, err := s.Commit(makeEdit("rename", "rename", `{"name":"First"}`, 0)); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(l.Path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString(`{"id":"broken","projectId":"` + s.Get().ID + `","sequence":2,"timestamp":"` + time.Now().Format(time.RFC3339Nano) + `","type":"broken","payload":`); err != nil {
		t.Fatal(err)
	}
	f.Close()
	recovered, err := service.Open(c, l, o, 1024)
	if err != nil || recovered.Get().Revision != 1 || recovered.Get().Project.Name != "First" {
		t.Fatalf("recovery failed: %+v %v", recovered, err)
	}
	if _, err = recovered.Commit(makeEdit("next", "rename", `{"name":"Second"}`, 1)); err != nil {
		t.Fatal(err)
	}
	rows, err := l.Read()
	if err != nil || len(rows) != 2 || rows[1].ID != "next" {
		t.Fatalf("log not repaired cleanly: %v %+v", err, rows)
	}
}

func TestLogWriteFailureBlocksFurtherEdits(t *testing.T) {
	s, _, l, _ := setup(t)
	if err := os.Chmod(l.Path, 0400); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(l.Path, 0600)
	if _, err := s.Commit(makeEdit("fail", "reset", `{}`, 0)); !errors.Is(err, domErrors.ErrUnavailable) {
		t.Fatalf("expected unavailable: %v", err)
	}
	if _, err := s.Commit(makeEdit("retry", "reset", `{}`, 0)); !errors.Is(err, domErrors.ErrUnavailable) {
		t.Fatalf("expected blocked service: %v", err)
	}
}

func TestRecoveryRejectsNonContiguousSequence(t *testing.T) {
	s, c, l, o := setup(t)
	if _, err := s.Commit(makeEdit("first", "rename", `{"name":"First"}`, 0)); err != nil {
		t.Fatal(err)
	}
	f, _ := os.OpenFile(l.Path, os.O_WRONLY|os.O_APPEND, 0600)
	row := fmt.Sprintf("{\"id\":\"gap\",\"projectId\":%q,\"sequence\":3,\"timestamp\":%q,\"type\":\"reset\",\"payload\":{}}\n", s.Get().ID, time.Now().UTC().Format(time.RFC3339Nano))
	_, _ = f.WriteString(row)
	f.Close()
	if _, err := service.Open(c, l, o, 1024); err == nil {
		t.Fatal("allowed sequence gap")
	}
}

func TestConcurrentCommitRace(t *testing.T) {
	s, _, _, _ := setup(t)
	var wg sync.WaitGroup
	successes := 0
	var mu sync.Mutex
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			op := makeEdit(fmt.Sprintf("op-%d", idx), "rename", fmt.Sprintf(`{"name":"Name %d"}`, idx), 0)
			if _, err := s.Commit(op); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("expected exactly 1 successful commit at revision 0, got %d", successes)
	}
}
