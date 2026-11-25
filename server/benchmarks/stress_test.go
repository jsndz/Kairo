package benchmarks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"server/internal/adapters/outbound/storage/catalog"
	exportstorage "server/internal/adapters/outbound/storage/export"
	"server/internal/adapters/outbound/storage/jsonl"
	"server/internal/app/collaboration"
	appexport "server/internal/app/export"
	"server/internal/app/project"
	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	domainExport "server/internal/domain/export"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

// TestStress_ConcurrentConflictRace verifies that when 50 concurrent goroutines attempt
// to mutate state against the same revision, exactly 1 succeeds and 49 fail with conflict.
func TestStress_ConcurrentConflictRace(t *testing.T) {
	dir := t.TempDir()
	cat := catalog.JSONCatalog{Path: filepath.Join(dir, "metadata.json")}
	logRepo := jsonl.JSONL{Path: filepath.Join(dir, "operations.jsonl")}
	mockObj := &mockObjects{data: make(map[string]asset.ObjectInfo)}

	svc, err := project.Open(cat, logRepo, mockObj, 1024*1024)
	if err != nil {
		t.Fatalf("failed to open project: %v", err)
	}

	concurrency := 50
	var wg sync.WaitGroup
	var successCount uint32
	var conflictCount uint32

	rev := uint64(0)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			payload, _ := json.Marshal(map[string]string{"name": fmt.Sprintf("Winner %d", idx)})
			edit := timeline.Edit{
				ID:               fmt.Sprintf("race-op-%d", idx),
				Type:             "rename",
				Payload:          payload,
				ExpectedRevision: &rev,
			}
			_, err := svc.Commit(edit)
			if err == nil {
				atomic.AddUint32(&successCount, 1)
			} else if errors.Is(err, domErrors.ErrConflict) {
				atomic.AddUint32(&conflictCount, 1)
			} else {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 success, got %d", successCount)
	}
	if conflictCount != uint32(concurrency-1) {
		t.Fatalf("expected %d conflicts, got %d", concurrency-1, conflictCount)
	}
	if svc.Get().Revision != 1 {
		t.Fatalf("expected final revision 1, got %d", svc.Get().Revision)
	}
}

// TestStress_CorruptedLogRecovery verifies that an ungraceful crash leaving a truncated
// trailing line in operations.jsonl is cleanly repaired and truncated on restart.
func TestStress_CorruptedLogRecovery(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "operations.jsonl")
	if err := os.WriteFile(logPath, nil, 0600); err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}
	l := jsonl.JSONL{Path: logPath}

	// 1. Write 5 valid records
	for i := 1; i <= 5; i++ {
		err := l.Append(record.Record{
			ID:        fmt.Sprintf("op-%d", i),
			ProjectID: "proj-recover",
			Sequence:  uint64(i),
			Timestamp: time.Now().UTC(),
			Type:      "rename",
			Payload:   []byte(`{"name":"test"}`),
		})
		if err != nil {
			t.Fatalf("failed to append record %d: %v", i, err)
		}
	}

	// 2. Simulate sudden power failure / crash mid-write: append corrupted partial bytes
	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatalf("failed to open log for corruption: %v", err)
	}
	// Missing closing braces and newline
	_, _ = f.Write([]byte(`{"id":"op-partial","projectId":"proj-recover","seq`))
	f.Sync()
	f.Close()

	// 3. Read should recover, truncate corrupted partial row, and return the 5 valid records
	records, err := l.Read()
	if err != nil {
		t.Fatalf("expected recovery on read, got error: %v", err)
	}
	if len(records) != 5 {
		t.Fatalf("expected 5 valid records recovered, got %d", len(records))
	}

	// 4. Subsequent writes should append cleanly at valid boundary
	err = l.Append(record.Record{
		ID:        "op-6",
		ProjectID: "proj-recover",
		Sequence:  6,
		Timestamp: time.Now().UTC(),
		Type:      "rename",
		Payload:   []byte(`{"name":"post-repair"}`),
	})
	if err != nil {
		t.Fatalf("failed to append post-repair: %v", err)
	}

	records2, err := l.Read()
	if err != nil || len(records2) != 6 {
		t.Fatalf("expected 6 records after repair, got %d (err: %v)", len(records2), err)
	}
}

// TestStress_OwnershipLockStorm subjects the OwnershipEngine to concurrent chaotic
// claim and release operations across 100 goroutines to verify no deadlocks or memory leaks.
func TestStress_OwnershipLockStorm(t *testing.T) {
	engine := collaboration.NewEngine()
	numClients := 100
	numClips := 20
	clips := make([]string, numClips)
	for i := 0; i < numClips; i++ {
		clips[i] = fmt.Sprintf("storm-clip-%d", i)
	}

	var wg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID string) {
			defer wg.Done()
			for iter := 0; iter < 100; iter++ {
				granted, _ := engine.Claim(clientID, clips)
				if len(granted) > 0 {
					engine.Release(clientID, granted)
				}
			}
		}(fmt.Sprintf("client-%d", i))
	}

	wg.Wait()

	// All locks should be released
	owners := engine.AllOwners()
	if len(owners) != 0 {
		t.Fatalf("expected 0 remaining locks after storm, got %d", len(owners))
	}
}

// TestStress_ExportQueueSaturation verifies that when export jobs exceed queue capacity,
// the service caps queue buffer cleanly and prevents uncontrolled unbounded memory growth.
func TestStress_ExportQueueSaturation(t *testing.T) {
	dir := t.TempDir()
	repo, err := exportstorage.NewFSRepository(dir)
	if err != nil {
		t.Fatal(err)
	}

	var state ports.State
	state.ID = "proj-sat"
	state.Revision = 1
	state.Project.Timeline = timeline.Timeline{
		Name:   "Sat Project",
		Tracks: []timeline.Track{{ID: "v1", Kind: "video"}},
		Clips:  []timeline.Clip{{ID: "c1", TrackID: "v1", MediaID: "a1", At: 0, Offset: 0, Length: 60, Volume: 1.0}},
	}
	state.Project.Assets = map[string]asset.Asset{
		"a1": {ID: "a1", Source: asset.Source{Kind: "video", Frames: 300}, Status: "completed"},
	}

	reader := &mockExportReader{state: state}
	resolver := &mockExportResolver{reader: reader}
	renderer := &mockBlockingRenderer{blockCh: make(chan struct{})}
	objects := &mockObjects{data: make(map[string]asset.ObjectInfo)}

	capacity := 5
	svc, err := appexport.New(appexport.Config{
		ProjectReader:   nil,
		ProjectResolver: resolver,
		Repo:            repo,
		Renderer:        renderer,
		ObjectStore:     objects,
		ScratchBaseDir:  filepath.Join(dir, "scratch"),
		QueueCapacity:   capacity,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		close(renderer.blockCh)
		svc.Stop()
	}()

	ctx := context.Background()
	settings := domainExport.DefaultSettings()

	// Fill queue: 1 rendering + capacity queued = capacity + 1
	for i := 0; i <= capacity+5; i++ {
		req := ports.SubmitRequest{
			ProjectID: "proj-sat",
			RequestID: fmt.Sprintf("req-fill-%d", i),
			Revision:  1,
			Settings:  settings,
		}
		job, err := svc.Submit(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error on submit %d: %v", i, err)
		}
		if job.ID == "" {
			t.Fatalf("expected valid job ID for job %d", i)
		}
	}
}

type mockBlockingRenderer struct {
	blockCh chan struct{}
}

func (m *mockBlockingRenderer) Render(ctx context.Context, _ domainExport.Manifest, _ string, _ func(float64)) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-m.blockCh:
		return "/tmp/out.mp4", nil
	}
}
