package export

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	domErrors "server/internal/domain/errors"
	domainExport "server/internal/domain/export"
	"server/internal/ports"
)

var _ ports.ExportService = (*Service)(nil)

// Service orchestrates export requests, queueing, and sequential rendering.
type Service struct {
	mu             sync.Mutex
	projectReader  ports.ProjectReader
	repo           ports.JobRepository
	renderer       ports.MediaRenderer
	prober         ports.MediaProber
	objectStore    ports.ObjectStore
	scratchBaseDir string
	log            *slog.Logger

	queue         chan string
	activeCancels map[string]context.CancelFunc
	stopCh        chan struct{}
	workerWg      sync.WaitGroup
}

// Config contains dependencies and directories for export Service.
type Config struct {
	ProjectReader  ports.ProjectReader
	Repo           ports.JobRepository
	Renderer       ports.MediaRenderer
	Prober         ports.MediaProber
	ObjectStore    ports.ObjectStore
	ScratchBaseDir string
	QueueCapacity  int
	Logger         *slog.Logger
}

// New creates and starts a sequential export service.
func New(cfg Config) (*Service, error) {
	if cfg.ScratchBaseDir == "" {
		cfg.ScratchBaseDir = filepath.Join(os.TempDir(), "kairo-exports")
	}
	if err := os.MkdirAll(cfg.ScratchBaseDir, 0700); err != nil {
		return nil, fmt.Errorf("create scratch dir: %w", err)
	}
	if cfg.QueueCapacity <= 0 {
		cfg.QueueCapacity = 100
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	s := &Service{
		projectReader:  cfg.ProjectReader,
		repo:           cfg.Repo,
		renderer:       cfg.Renderer,
		prober:         cfg.Prober,
		objectStore:    cfg.ObjectStore,
		scratchBaseDir: cfg.ScratchBaseDir,
		log:            cfg.Logger,
		queue:          make(chan string, cfg.QueueCapacity),
		activeCancels:  make(map[string]context.CancelFunc),
		stopCh:         make(chan struct{}),
	}

	s.workerWg.Add(1)
	go s.workerLoop()

	return s, nil
}

// Stop shuts down the background export worker.
func (s *Service) Stop() {
	s.mu.Lock()
	for _, cancel := range s.activeCancels {
		cancel()
	}
	s.mu.Unlock()

	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	s.workerWg.Wait()
}

// Recover requeues any interrupted jobs from previous server runs.
func (s *Service) Recover() error {
	jobs, err := s.repo.ListJobs()
	if err != nil {
		return fmt.Errorf("list jobs for recovery: %w", err)
	}

	recoveredCount := 0
	for _, job := range jobs {
		if job.Status == domainExport.JobStatusRendering {
			job.Status = domainExport.JobStatusQueued
			job.Progress = 0
			_ = s.repo.SaveJob(job)
			s.enqueue(job.ID)
			recoveredCount++
		} else if job.Status == domainExport.JobStatusQueued {
			s.enqueue(job.ID)
			recoveredCount++
		}
	}

	if recoveredCount > 0 {
		s.log.Info("recovered export jobs", "count", recoveredCount)
	}
	return nil
}

// Submit validates export parameters, stores manifest & job, and queues rendering.
func (s *Service) Submit(ctx context.Context, req ports.SubmitRequest) (domainExport.Job, error) {
	if err := domainExport.ValidateSettings(&req.Settings); err != nil {
		return domainExport.Job{}, err
	}

	state := s.projectReader.Get()

	// Stale revision check (SD1 requirement)
	if req.Revision != state.Revision {
		return domainExport.Job{}, fmt.Errorf("%w: stale project revision (requested %d, current %d)",
			domErrors.ErrConflict, req.Revision, state.Revision)
	}

	// Idempotency check: repeated request ID returns existing job
	if req.RequestID != "" {
		if existing, err := s.repo.GetJobByRequestID(state.ID, req.RequestID); err == nil {
			return existing, nil
		}
	}

	// Validate timeline and assets
	usedAssets, err := domainExport.ValidateTimelineForExport(state.Project.Timeline, state.Project.Assets)
	if err != nil {
		return domainExport.Job{}, err
	}

	manifest := domainExport.Manifest{
		ProjectID: state.ID,
		Revision:  state.Revision,
		Timeline:  state.Project.Timeline,
		Assets:    usedAssets,
		Settings:  req.Settings,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.SaveManifest(manifest); err != nil {
		return domainExport.Job{}, fmt.Errorf("save manifest: %w", err)
	}

	jobID := generateID()
	job := domainExport.Job{
		ID:        jobID,
		RequestID: req.RequestID,
		ProjectID: state.ID,
		Revision:  state.Revision,
		Status:    domainExport.JobStatusQueued,
		Progress:  0,
		Attempts:  1,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.SaveJob(job); err != nil {
		return domainExport.Job{}, fmt.Errorf("save job: %w", err)
	}

	s.enqueue(job.ID)
	return job, nil
}

// Get returns the job record by ID.
func (s *Service) Get(ctx context.Context, id string) (domainExport.Job, error) {
	return s.repo.GetJob(id)
}

// Cancel terminates a running render or cancels a queued job.
func (s *Service) Cancel(ctx context.Context, id string) (domainExport.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, err := s.repo.GetJob(id)
	if err != nil {
		return domainExport.Job{}, err
	}

	if job.Status == domainExport.JobStatusCompleted || job.Status == domainExport.JobStatusFailed {
		return job, fmt.Errorf("%w: job is already %s", domErrors.ErrInvalid, job.Status)
	}

	if cancelFn, active := s.activeCancels[id]; active {
		cancelFn()
	}

	now := time.Now().UTC()
	job.Status = domainExport.JobStatusCancelled
	job.Error = "export cancelled by user"
	job.EndedAt = &now

	if err := s.repo.SaveJob(job); err != nil {
		return domainExport.Job{}, err
	}

	return job, nil
}

// Retry requeues a failed or cancelled job using the original manifest.
func (s *Service) Retry(ctx context.Context, id string) (domainExport.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, err := s.repo.GetJob(id)
	if err != nil {
		return domainExport.Job{}, err
	}

	if job.Status != domainExport.JobStatusFailed && job.Status != domainExport.JobStatusCancelled {
		return domainExport.Job{}, fmt.Errorf("%w: can only retry failed or cancelled jobs", domErrors.ErrInvalid)
	}

	job.Status = domainExport.JobStatusQueued
	job.Progress = 0
	job.Error = ""
	job.Attempts++
	job.StartedAt = nil
	job.EndedAt = nil

	if err := s.repo.SaveJob(job); err != nil {
		return domainExport.Job{}, err
	}

	s.enqueue(job.ID)
	return job, nil
}

// DownloadURL returns a presigned download URL for a completed and verified export.
func (s *Service) DownloadURL(ctx context.Context, id string) (string, error) {
	job, err := s.repo.GetJob(id)
	if err != nil {
		return "", err
	}

	if job.Status != domainExport.JobStatusCompleted || job.OutputKey == "" {
		return "", fmt.Errorf("%w: export is not completed", domErrors.ErrInvalid)
	}

	return s.objectStore.ReadURL(ctx, job.OutputKey)
}

func (s *Service) enqueue(jobID string) {
	select {
	case s.queue <- jobID:
	default:
		s.log.Error("export queue full, dropping job", "jobID", jobID)
	}
}

func (s *Service) workerLoop() {
	defer s.workerWg.Done()
	for {
		select {
		case <-s.stopCh:
			return
		case jobID := <-s.queue:
			s.processJob(jobID)
		}
	}
}

func (s *Service) processJob(jobID string) {
	job, err := s.repo.GetJob(jobID)
	if err != nil {
		s.log.Error("load job failed", "jobID", jobID, "error", err)
		return
	}

	// If cancelled while waiting in queue, skip
	if job.Status != domainExport.JobStatusQueued {
		return
	}

	manifestKey := fmt.Sprintf("%s_%d", job.ProjectID, job.Revision)
	manifest, err := s.repo.GetManifest(manifestKey)
	if err != nil {
		s.markFailed(job, fmt.Sprintf("load manifest: %v", err))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.activeCancels[jobID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.activeCancels, jobID)
		s.mu.Unlock()
		cancel()
	}()

	now := time.Now().UTC()
	job.Status = domainExport.JobStatusRendering
	job.StartedAt = &now
	job.Progress = 0
	_ = s.repo.SaveJob(job)

	scratchDir := filepath.Join(s.scratchBaseDir, jobID)
	_ = os.MkdirAll(scratchDir, 0700)
	defer os.RemoveAll(scratchDir)

	s.log.Info("starting export render", "jobID", jobID, "revision", job.Revision)

	lastProgressUpdate := time.Now()
	onProgress := func(percent float64) {
		if percent > 100 {
			percent = 100
		}
		if time.Since(lastProgressUpdate) > 500*time.Millisecond || percent >= 100 {
			job.Progress = percent
			_ = s.repo.SaveJob(job)
			lastProgressUpdate = time.Now()
		}
	}

	outputPath, err := s.renderer.Render(ctx, manifest, scratchDir, onProgress)
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			s.log.Info("render cancelled", "jobID", jobID)
			s.markCancelled(job)
			return
		}
		s.log.Error("render failed", "jobID", jobID, "error", err)
		s.markFailed(job, err.Error())
		return
	}

	// Stage 10: Output Verification with ffprobe
	if s.prober != nil {
		probeRes, err := s.prober.Probe(ctx, outputPath)
		if err != nil {
			s.log.Error("probe verification failed", "jobID", jobID, "error", err)
			s.markFailed(job, fmt.Sprintf("probe verification failed: %v", err))
			return
		}

		if err := verifyProbeResult(probeRes, manifest); err != nil {
			s.log.Error("verification criteria failed", "jobID", jobID, "error", err)
			s.markFailed(job, fmt.Sprintf("verification failed: %v", err))
			return
		}
	}

	// Publish to MinIO
	outputKey := fmt.Sprintf("exports/%s/%s/%s", job.ProjectID, job.ID, manifest.Settings.Filename)
	if err := s.objectStore.Upload(ctx, outputKey, outputPath, "video/mp4"); err != nil {
		s.log.Error("publish output to storage failed", "jobID", jobID, "error", err)
		s.markFailed(job, fmt.Sprintf("publish to storage: %v", err))
		return
	}

	ended := time.Now().UTC()
	job.Status = domainExport.JobStatusCompleted
	job.Progress = 100
	job.OutputKey = outputKey
	job.EndedAt = &ended
	_ = s.repo.SaveJob(job)
	s.log.Info("export completed successfully", "jobID", jobID, "outputKey", outputKey)
}

func verifyProbeResult(res ports.ProbeResult, manifest domainExport.Manifest) error {
	hasVideo := false
	hasAudio := false

	expectedW, expectedH := manifest.Settings.Resolution.Dimensions()

	for _, s := range res.Streams {
		if s.CodecType == "video" {
			if s.CodecName != "h264" {
				return fmt.Errorf("expected h264 video, got %s", s.CodecName)
			}
			if s.Width != expectedW || s.Height != expectedH {
				return fmt.Errorf("expected resolution %dx%d, got %dx%d", expectedW, expectedH, s.Width, s.Height)
			}
			hasVideo = true
		} else if s.CodecType == "audio" {
			if s.CodecName != "aac" {
				return fmt.Errorf("expected aac audio, got %s", s.CodecName)
			}
			hasAudio = true
		}
	}

	if !hasVideo {
		return fmt.Errorf("output MP4 missing video stream")
	}
	if !hasAudio {
		return fmt.Errorf("output MP4 missing audio stream")
	}

	// Check duration tolerance (within 1 second)
	maxFrames := 1
	for _, c := range manifest.Timeline.Clips {
		if c.At+c.Length > maxFrames {
			maxFrames = c.At + c.Length
		}
	}
	expectedDur := float64(maxFrames) / 30.0
	if math.Abs(res.DurationSeconds-expectedDur) > 1.0 {
		return fmt.Errorf("duration mismatch: expected ~%.2fs, got %.2fs", expectedDur, res.DurationSeconds)
	}

	return nil
}

func (s *Service) markFailed(job domainExport.Job, reason string) {
	now := time.Now().UTC()
	job.Status = domainExport.JobStatusFailed
	job.Error = reason
	job.EndedAt = &now
	_ = s.repo.SaveJob(job)
}

func (s *Service) markCancelled(job domainExport.Job) {
	now := time.Now().UTC()
	job.Status = domainExport.JobStatusCancelled
	job.Error = "cancelled"
	job.EndedAt = &now
	_ = s.repo.SaveJob(job)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
