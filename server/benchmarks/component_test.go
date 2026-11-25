package benchmarks

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"server/internal/adapters/outbound/storage/catalog"
	exportstorage "server/internal/adapters/outbound/storage/export"
	"server/internal/adapters/outbound/storage/jsonl"
	"server/internal/adapters/outbound/ffmpeg"
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

type mockObjects struct {
	data map[string]asset.ObjectInfo
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
func (o *mockObjects) Promote(_ context.Context, _, key, _ string) error {
	return nil
}
func (o *mockObjects) ReadURL(context.Context, string) (string, error) {
	return "https://minio.invalid/file", nil
}
func (o *mockObjects) Download(context.Context, string, string) error { return nil }
func (o *mockObjects) Upload(context.Context, string, string, string) error   { return nil }

// BenchmarkStorage_AppendSequential measures raw append throughput and I/O latency
// when appending operations sequentially to operations.jsonl with fsync.
func BenchmarkStorage_AppendSequential(b *testing.B) {
	dir := b.TempDir()
	logPath := filepath.Join(dir, "operations.jsonl")
	if err := os.WriteFile(logPath, nil, 0600); err != nil {
		b.Fatalf("failed to create log file: %v", err)
	}
	repo := jsonl.JSONL{Path: logPath}
	payload := json.RawMessage(`{"name":"benchmark-title"}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rec := record.Record{
			ID:        fmt.Sprintf("op-append-%d", i),
			ProjectID: "bench-proj-1",
			Sequence:  uint64(i + 1),
			Timestamp: time.Now().UTC(),
			Type:      "rename",
			Payload:   payload,
		}
		if err := repo.Append(rec); err != nil {
			b.Fatalf("append failed at %d: %v", i, err)
		}
	}
}

// BenchmarkStorage_CommitAuthor measures the full ProjectService.CommitAuthor lifecycle
// including validation, payload canonicalization, revision checking, state reduction,
// append-only disk write with fsync, and new state generation.
func BenchmarkStorage_CommitAuthor(b *testing.B) {
	dir := b.TempDir()
	cat := catalog.JSONCatalog{Path: filepath.Join(dir, "metadata.json")}
	logRepo := jsonl.JSONL{Path: filepath.Join(dir, "operations.jsonl")}
	mockObj := &mockObjects{data: make(map[string]asset.ObjectInfo)}

	svc, err := project.Open(cat, logRepo, mockObj, 1024*1024)
	if err != nil {
		b.Fatalf("failed to open project service: %v", err)
	}

	engine := collaboration.NewEngine()
	authorID := "bench-author-1"
	payload := json.RawMessage(`{"name":"Updated Project"}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rev := svc.Get().Revision
		edit := timeline.Edit{
			ID:               fmt.Sprintf("op-commit-%d", i),
			Type:             "rename",
			Payload:          payload,
			ExpectedRevision: &rev,
		}
		if _, err := svc.CommitAuthor(edit, authorID, engine); err != nil {
			b.Fatalf("commit author failed at %d: %v", i, err)
		}
	}
}

// BenchmarkStorage_ReplayHydration measures cold-start hydration time when reading
// an existing project log from disk and rebuilding state across different log depths.
func BenchmarkStorage_ReplayHydration(b *testing.B) {
	depths := []int{100, 500, 1000, 5000}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("%d_ops", depth), func(b *testing.B) {
			dir := b.TempDir()
			metaPath := filepath.Join(dir, "metadata.json")
			logPath := filepath.Join(dir, "operations.jsonl")

			cat := catalog.JSONCatalog{Path: metaPath}
			logRepo := jsonl.JSONL{Path: logPath}
			mockObj := &mockObjects{data: make(map[string]asset.ObjectInfo)}

			// Save metadata
			meta := record.Metadata{
				ID:     "hydration-proj",
				Assets: make(map[string]asset.Asset),
			}
			if err := cat.Save(meta); err != nil {
				b.Fatalf("failed to save metadata: %v", err)
			}

			// Pre-populate operations.jsonl with `depth` operations
			f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			if err != nil {
				b.Fatalf("failed to create operations log: %v", err)
			}
			buf := bufio.NewWriter(f)
			for j := 1; j <= depth; j++ {
				rec := record.Record{
					ID:        fmt.Sprintf("hydrate-op-%d", j),
					ProjectID: "hydration-proj",
					Sequence:  uint64(j),
					Timestamp: time.Now().UTC(),
					Type:      "rename",
					Payload:   []byte(fmt.Sprintf(`{"name":"Title %d"}`, j)),
				}
				data, err := json.Marshal(rec)
				if err != nil {
					b.Fatalf("marshal failed: %v", err)
				}
				if _, err := buf.Write(append(data, '\n')); err != nil {
					b.Fatalf("write failed: %v", err)
				}
			}
			if err := buf.Flush(); err != nil {
				b.Fatalf("flush failed: %v", err)
			}
			if err := f.Sync(); err != nil {
				b.Fatalf("sync failed: %v", err)
			}
			f.Close()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				svc, err := project.Open(cat, logRepo, mockObj, 1024*1024)
				if err != nil {
					b.Fatalf("open project failed: %v", err)
				}
				_ = svc
			}
		})
	}
}

// BenchmarkOwnership_Contention simulates 2 competing clients hammering OwnershipEngine.Claim
// and Release concurrently across 10 shared clips to measure mutex contention and lock throughput.
func BenchmarkOwnership_Contention(b *testing.B) {
	engine := collaboration.NewEngine()
	clips := make([]string, 10)
	for i := 0; i < 10; i++ {
		clips[i] = fmt.Sprintf("clip-%d", i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.SetParallelism(2)

	var clientToggle uint32
	b.RunParallel(func(pb *testing.PB) {
		idNum := atomic.AddUint32(&clientToggle, 1) % 2
		clientID := fmt.Sprintf("client-%d", idNum+1)

		for pb.Next() {
			engine.Claim(clientID, clips)
			engine.Release(clientID, clips)
		}
	})
}

// BenchmarkProjectManager_LazyLoad measures ProjectManager retrieval speed,
// covering in-memory cache hits and lazy disk load.
func BenchmarkProjectManager_LazyLoad(b *testing.B) {
	dir := b.TempDir()
	mockObj := &mockObjects{data: make(map[string]asset.ObjectInfo)}

	mgr, err := project.NewManager(dir, mockObj, 1024*1024)
	if err != nil {
		b.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()
	summary, _, err := mgr.Create(ctx, "Bench Project")
	if err != nil {
		b.Fatalf("failed to create project: %v", err)
	}

	b.Run("CacheHit", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := mgr.Get(ctx, summary.ID); err != nil {
				b.Fatalf("get failed: %v", err)
			}
		}
	})

	b.Run("ColdLoadWithEviction", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			mgr.EvictIdle(0, func(string) bool { return false })
			b.StartTimer()

			if _, err := mgr.Get(ctx, summary.ID); err != nil {
				b.Fatalf("get failed: %v", err)
			}
		}
	})
}

type mockExportReader struct {
	state ports.State
}

func (m *mockExportReader) Get() ports.State                             { return m.state }
func (m *mockExportReader) GetOpsSince(_ uint64) ([]record.Record, error) { return nil, nil }
func (m *mockExportReader) ProjectID() string                             { return m.state.ID }
func (m *mockExportReader) Summary() record.ProjectSummary {
	return record.ProjectSummary{ID: m.state.ID, Revision: m.state.Revision}
}

type mockExportResolver struct {
	reader ports.ProjectReader
}

func (m *mockExportResolver) ResolveReader(_ context.Context, _ string) (ports.ProjectReader, error) {
	return m.reader, nil
}

type mockExportRenderer struct{}

func (m *mockExportRenderer) Render(_ context.Context, _ domainExport.Manifest, _ string, _ func(float64)) (string, error) {
	return "/tmp/out.mp4", nil
}

// BenchmarkExport_FilterGraphBuilder measures the performance of building complex FFmpeg
// filtergraphs across timelines containing 10, 50, 100, and 250 clips.
func BenchmarkExport_FilterGraphBuilder(b *testing.B) {
	clipCounts := []int{10, 50, 100, 250}
	for _, count := range clipCounts {
		b.Run(fmt.Sprintf("%d_clips", count), func(b *testing.B) {
			tracks := []timeline.Track{
				{ID: "video-1", Kind: "video"},
				{ID: "video-2", Kind: "video"},
				{ID: "audio-1", Kind: "audio"},
			}
			clips := make([]timeline.Clip, count)
			assets := make(map[string]asset.Asset)
			sourcePaths := make(map[string]string)
			sourceHasAudio := make(map[string]bool)
			sourceHasVideo := make(map[string]bool)

			for i := 0; i < count; i++ {
				clipID := fmt.Sprintf("clip-%d", i)
				mediaID := fmt.Sprintf("asset-%d", i%5)
				trackID := "video-1"
				if i%3 == 1 {
					trackID = "video-2"
				} else if i%3 == 2 {
					trackID = "audio-1"
				}

				clips[i] = timeline.Clip{
					ID:      clipID,
					TrackID: trackID,
					MediaID: mediaID,
					At:      i * 30,
					Offset:  0,
					Length:  60,
					Volume:  0.8,
				}
				assets[mediaID] = asset.Asset{
					ID: mediaID,
					Source: asset.Source{
						Kind:   "video",
						Frames: 300,
					},
					Status: "completed",
				}
				sourcePaths[mediaID] = fmt.Sprintf("/tmp/fake/%s.mp4", mediaID)
				sourceHasVideo[mediaID] = true
				sourceHasAudio[mediaID] = true
			}

			manifest := domainExport.Manifest{
				ProjectID: "bench-export-proj",
				Revision:  1,
				Timeline: timeline.Timeline{
					Name:   "Export Bench",
					Tracks: tracks,
					Clips:  clips,
				},
				Assets:    assets,
				Settings:  domainExport.DefaultSettings(),
				CreatedAt: time.Now(),
			}

			cfg := ffmpeg.PipelineConfig{
				Manifest:       manifest,
				ScratchDir:     "/tmp/bench-scratch",
				SourcePaths:    sourcePaths,
				SourceHasAudio: sourceHasAudio,
				SourceHasVideo: sourceHasVideo,
				OutputPath:     "/tmp/bench-scratch/out.mp4",
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				args, err := ffmpeg.BuildFFmpegArgs(cfg)
				if err != nil {
					b.Fatalf("build args failed: %v", err)
				}
				_ = args
			}
		})
	}
}

// BenchmarkExport_JobSubmit measures export job submission, manifest generation,
// queue handoff, and metadata persistence under high throughput.
func BenchmarkExport_JobSubmit(b *testing.B) {
	dir := b.TempDir()
	repo, err := exportstorage.NewFSRepository(dir)
	if err != nil {
		b.Fatal(err)
	}

	var state ports.State
	state.ID = "bench-proj"
	state.Revision = 1
	state.Project.Timeline = timeline.Timeline{
		Name: "Bench Project",
		Tracks: []timeline.Track{
			{ID: "video-1", Kind: "video"},
		},
		Clips: []timeline.Clip{
			{ID: "c1", TrackID: "video-1", MediaID: "a1", At: 0, Offset: 0, Length: 60, Volume: 1.0},
		},
	}
	state.Project.Assets = map[string]asset.Asset{
		"a1": {ID: "a1", Source: asset.Source{Kind: "video", Frames: 300}, Status: "completed"},
	}

	reader := &mockExportReader{state: state}
	resolver := &mockExportResolver{reader: reader}
	renderer := &mockExportRenderer{}
	objects := &mockObjects{data: make(map[string]asset.ObjectInfo)}

	svc, err := appexport.New(appexport.Config{
		ProjectReader:   reader,
		ProjectResolver: resolver,
		Repo:            repo,
		Renderer:        renderer,
		ObjectStore:     objects,
		ScratchBaseDir:  filepath.Join(dir, "scratch"),
		QueueCapacity:   100000,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer svc.Stop()

	settings := domainExport.DefaultSettings()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := ports.SubmitRequest{
			ProjectID: "bench-proj",
			RequestID: fmt.Sprintf("req-%d", i),
			Revision:  1,
			Settings:  settings,
		}
		job, err := svc.Submit(ctx, req)
		if err != nil {
			b.Fatalf("submit failed: %v", err)
		}
		_ = job
	}
}

