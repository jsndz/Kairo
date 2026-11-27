package export_test

import (
	"context"
	"errors"
	"testing"
	"time"

	exportstorage "server/internal/adapters/outbound/storage/export"
	appexport "server/internal/app/export"
	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	domainExport "server/internal/domain/export"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

type mockProjectReader struct {
	state ports.State
}

func (m *mockProjectReader) Get() ports.State {
	return m.state
}
func (m *mockProjectReader) GetOpsSince(_ uint64) ([]record.Record, error) {
	return nil, nil
}
func (m *mockProjectReader) ProjectID() string {
	return m.state.ID
}
func (m *mockProjectReader) Metadata() record.ProjectMetadata {
	return record.ProjectMetadata{
		ID:       m.state.ID,
		Name:     m.state.Project.Name,
		Revision: m.state.Revision,
	}
}

type mockRenderer struct {
	renderDelay time.Duration
	fail        bool
}

func (m *mockRenderer) Render(ctx context.Context, _ domainExport.Manifest, scratchDir string, onProgress func(float64)) (string, error) {
	if m.fail {
		return "", errors.New("renderer boom")
	}
	if onProgress != nil {
		onProgress(50.0)
	}
	if m.renderDelay > 0 {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(m.renderDelay):
		}
	}
	if onProgress != nil {
		onProgress(100.0)
	}
	return scratchDir + "/output.mp4", nil
}

type mockProber struct{}

func (m *mockProber) Probe(_ context.Context, _ string) (ports.ProbeResult, error) {
	return ports.ProbeResult{
		DurationSeconds: 1.0,
		Streams: []ports.ProbeStream{
			{CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080},
			{CodecType: "audio", CodecName: "aac"},
		},
	}, nil
}

type mockStore struct {
	uploaded map[string]string
}

func (m *mockStore) Bucket() string { return "test-bucket" }
func (m *mockStore) UploadPolicy(context.Context, string, asset.Source) (asset.Upload, error) {
	return asset.Upload{}, nil
}
func (m *mockStore) Stat(context.Context, string) (asset.ObjectInfo, error) {
	return asset.ObjectInfo{}, nil
}
func (m *mockStore) Promote(context.Context, string, string, string) error { return nil }
func (m *mockStore) ReadURL(context.Context, string) (string, error) {
	return "https://minio.test/download-signed", nil
}
func (m *mockStore) Download(context.Context, string, string) error { return nil }
func (m *mockStore) Upload(_ context.Context, key, path, _ string) error {
	if m.uploaded == nil {
		m.uploaded = make(map[string]string)
	}
	m.uploaded[key] = path
	return nil
}

func setupTestService(t *testing.T, renderer ports.MediaRenderer) (*appexport.Service, *mockProjectReader, *exportstorage.FSRepository) {
	t.Helper()
	dir := t.TempDir()
	repo, err := exportstorage.NewFSRepository(dir)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	reader := &mockProjectReader{
		state: ports.State{
			ID:       "proj-1",
			Revision: 10,
		},
	}
	reader.state.Project.Timeline = timeline.Timeline{
		Name:   "Test",
		Tracks: []timeline.Track{{ID: "v1", Kind: "video"}},
		Clips: []timeline.Clip{
			{ID: "c1", TrackID: "v1", MediaID: "a1", At: 0, Offset: 0, Length: 30, Volume: 1.0},
		},
	}
	reader.state.Project.Assets = map[string]asset.Asset{
		"a1": {ID: "a1", Source: asset.Source{Kind: "video", Frames: 100}, Status: "completed", ObjectKey: "media/a1.mp4"},
	}

	store := &mockStore{}
	if renderer == nil {
		renderer = &mockRenderer{}
	}

	svc, err := appexport.New(appexport.Config{
		ProjectReader:  reader,
		Repo:           repo,
		Renderer:       renderer,
		Prober:         &mockProber{},
		ObjectStore:    store,
		ScratchBaseDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to init service: %v", err)
	}

	return svc, reader, repo
}

func TestExportService_Submit(t *testing.T) {
	t.Run("stale revision returns conflict", func(t *testing.T) {
		svc, _, _ := setupTestService(t, nil)
		defer svc.Stop()

		_, err := svc.Submit(context.Background(), ports.SubmitRequest{
			Revision: 9, // Current is 10
		})
		if !errors.Is(err, domErrors.ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("empty timeline returns invalid", func(t *testing.T) {
		svc, reader, _ := setupTestService(t, nil)
		defer svc.Stop()
		reader.state.Project.Timeline.Clips = nil

		_, err := svc.Submit(context.Background(), ports.SubmitRequest{
			Revision: 10,
		})
		if !errors.Is(err, domErrors.ErrInvalid) {
			t.Fatalf("expected ErrInvalid, got %v", err)
		}
	})

	t.Run("idempotency with repeated request ID", func(t *testing.T) {
		svc, _, _ := setupTestService(t, nil)
		defer svc.Stop()

		req := ports.SubmitRequest{
			RequestID: "req-xyz",
			Revision:  10,
		}

		job1, err := svc.Submit(context.Background(), req)
		if err != nil {
			t.Fatalf("submit 1 failed: %v", err)
		}

		job2, err := svc.Submit(context.Background(), req)
		if err != nil {
			t.Fatalf("submit 2 failed: %v", err)
		}

		if job1.ID != job2.ID {
			t.Fatalf("expected identical job IDs, got %s and %s", job1.ID, job2.ID)
		}
	})

	t.Run("happy path renders and completes", func(t *testing.T) {
		svc, _, repo := setupTestService(t, nil)
		defer svc.Stop()

		job, err := svc.Submit(context.Background(), ports.SubmitRequest{
			RequestID: "req-happy",
			Revision:  10,
		})
		if err != nil {
			t.Fatalf("submit failed: %v", err)
		}

		// Wait for sequential worker to complete
		var finalJob domainExport.Job
		for i := 0; i < 20; i++ {
			time.Sleep(50 * time.Millisecond)
			finalJob, _ = repo.GetJob(job.ID)
			if finalJob.Status == domainExport.JobStatusCompleted {
				break
			}
		}

		if finalJob.Status != domainExport.JobStatusCompleted {
			t.Fatalf("expected completed status, got %s (err: %s)", finalJob.Status, finalJob.Error)
		}
		if finalJob.OutputKey == "" {
			t.Fatal("expected non-empty OutputKey")
		}

		// Check download URL
		url, err := svc.DownloadURL(context.Background(), job.ID)
		if err != nil {
			t.Fatalf("expected download URL, got error %v", err)
		}
		if url == "" {
			t.Fatal("expected download URL string")
		}
	})
}

func TestExportService_CancelAndRetry(t *testing.T) {
	renderer := &mockRenderer{renderDelay: 500 * time.Millisecond}
	svc, _, repo := setupTestService(t, renderer)
	defer svc.Stop()

	job, err := svc.Submit(context.Background(), ports.SubmitRequest{
		RequestID: "req-cancel",
		Revision:  10,
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	// Give worker time to start rendering
	time.Sleep(50 * time.Millisecond)

	// Cancel
	cancelledJob, err := svc.Cancel(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if cancelledJob.Status != domainExport.JobStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelledJob.Status)
	}

	// Wait for worker to finish clean-up
	time.Sleep(100 * time.Millisecond)
	persisted, _ := repo.GetJob(job.ID)
	if persisted.Status != domainExport.JobStatusCancelled {
		t.Fatalf("expected persisted status cancelled, got %s", persisted.Status)
	}

	// Retry
	retriedJob, err := svc.Retry(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if retriedJob.Status != domainExport.JobStatusQueued || retriedJob.Attempts != 2 {
		t.Fatalf("expected queued status with attempt 2, got status %s attempts %d",
			retriedJob.Status, retriedJob.Attempts)
	}
}
