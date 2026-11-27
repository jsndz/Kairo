package project_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"server/internal/app/project"
	"server/internal/domain/asset"
	"server/internal/domain/timeline"
)

type mockMgrObjects struct{}

func (m *mockMgrObjects) Bucket() string { return "media" }
func (m *mockMgrObjects) UploadPolicy(ctx context.Context, key string, src asset.Source) (asset.Upload, error) {
	return asset.Upload{}, nil
}
func (m *mockMgrObjects) Stat(ctx context.Context, key string) (asset.ObjectInfo, error) {
	return asset.ObjectInfo{Size: 1024, MIME: "video/mp4"}, nil
}
func (m *mockMgrObjects) Promote(ctx context.Context, stage, key, etag string) error { return nil }
func (m *mockMgrObjects) ReadURL(ctx context.Context, key string) (string, error)     { return "", nil }
func (m *mockMgrObjects) Download(ctx context.Context, key string, target string) error {
	return nil
}
func (m *mockMgrObjects) Upload(ctx context.Context, key, source, contentType string) error {
	return nil
}

func TestManager_CreateAndList(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := project.NewManager(tmpDir, &mockMgrObjects{}, 1024*1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// 1. Create two projects
	p1, _, err := mgr.Create(ctx, "Film Edit 1")
	if err != nil {
		t.Fatalf("Create p1 failed: %v", err)
	}
	p2, _, err := mgr.Create(ctx, "Commercial 2")
	if err != nil {
		t.Fatalf("Create p2 failed: %v", err)
	}

	// 2. List projects
	summaries, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	foundP1, foundP2 := false, false
	for _, s := range summaries {
		if s.ID == p1.ID && s.Name == "Film Edit 1" {
			foundP1 = true
		}
		if s.ID == p2.ID && s.Name == "Commercial 2" {
			foundP2 = true
		}
	}
	if !foundP1 || !foundP2 {
		t.Fatalf("missing created projects in list: %+v", summaries)
	}
}

func TestManager_GetAndHydrateFromSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := project.NewManager(tmpDir, &mockMgrObjects{}, 1024*1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// 1. Create project
	summary, svc, err := mgr.Create(ctx, "Snapshot Test")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 2. Commit tracks and rename operations
	trackPayload, _ := json.Marshal(map[string]any{
		"trackId": "video-extra",
		"kind":    "video",
	})
	rev := svc.Get().Revision
	_, err = svc.Commit(timeline.Edit{
		ID:               "op-track-1",
		Type:             "add-track",
		Payload:          trackPayload,
		ExpectedRevision: &rev,
	})
	if err != nil {
		t.Fatalf("Commit add-track failed: %v", err)
	}

	renamePayload, _ := json.Marshal(map[string]string{
		"name": "Updated Snapshot Title",
	})
	rev = svc.Get().Revision
	_, err = svc.Commit(timeline.Edit{
		ID:               "op-rename-2",
		Type:             "rename",
		Payload:          renamePayload,
		ExpectedRevision: &rev,
	})
	if err != nil {
		t.Fatalf("Commit rename failed: %v", err)
	}

	// 3. Evict project from memory (calls svc.Close() which flushes snapshot & WAL checkpoint)
	mgr.EvictIdle(0, nil)

	// 4. Re-open project from SQLite disk
	loadedSvc, err := mgr.Get(ctx, summary.ID)
	if err != nil {
		t.Fatalf("Get after eviction failed: %v", err)
	}

	st := loadedSvc.Get()
	if st.Revision != 3 {
		t.Fatalf("expected revision 3 after reload, got %d", st.Revision)
	}
	if st.Project.Name != "Updated Snapshot Title" {
		t.Fatalf("expected name 'Updated Snapshot Title', got %q", st.Project.Name)
	}
	if len(st.Project.Tracks) != 3 {
		t.Fatalf("expected 3 tracks, got %d", len(st.Project.Tracks))
	}
}

func TestManager_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := project.NewManager(tmpDir, &mockMgrObjects{}, 1024*1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	p, _, err := mgr.Create(ctx, "To Delete")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := mgr.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify directory removed
	if _, err := os.Stat(filepath.Join(tmpDir, p.ID)); !os.IsNotExist(err) {
		t.Fatalf("expected project dir to be deleted")
	}

	// Verify removed from list
	summaries, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(summaries) != 0 {
		t.Fatalf("expected 0 summaries after deletion, got %d", len(summaries))
	}
}
