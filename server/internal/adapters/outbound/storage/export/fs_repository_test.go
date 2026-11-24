package export_test

import (
	"errors"
	"testing"
	"time"

	exportstorage "server/internal/adapters/outbound/storage/export"
	domErrors "server/internal/domain/errors"
	"server/internal/domain/export"
	"server/internal/domain/timeline"
)

func TestFSRepository(t *testing.T) {
	dir := t.TempDir()
	repo, err := exportstorage.NewFSRepository(dir)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	job := export.Job{
		ID:        "job-1",
		RequestID: "req-1",
		ProjectID: "proj-1",
		Revision:  5,
		Status:    export.JobStatusQueued,
		Progress:  0,
		Attempts:  1,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}

	if err := repo.SaveJob(job); err != nil {
		t.Fatalf("failed to save job: %v", err)
	}

	// Read job back
	got, err := repo.GetJob("job-1")
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}
	if got.ID != job.ID || got.RequestID != job.RequestID || got.Status != job.Status {
		t.Fatalf("job mismatch: got %+v, want %+v", got, job)
	}

	// Lookup by request ID
	byReq, err := repo.GetJobByRequestID("proj-1", "req-1")
	if err != nil {
		t.Fatalf("failed to get job by request ID: %v", err)
	}
	if byReq.ID != job.ID {
		t.Fatalf("expected job-1, got %s", byReq.ID)
	}

	// Not found request ID
	_, err = repo.GetJobByRequestID("proj-1", "req-unknown")
	if !errors.Is(err, domErrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// List jobs
	jobs, err := repo.ListJobs()
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "job-1" {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	// Save and retrieve manifest
	manifest := export.Manifest{
		ProjectID: "proj-1",
		Revision:  5,
		Timeline:  timeline.Empty(),
		Settings:  export.DefaultSettings(),
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := repo.SaveManifest(manifest); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	gotM, err := repo.GetManifest("proj-1_5")
	if err != nil {
		t.Fatalf("failed to get manifest: %v", err)
	}
	if gotM.ProjectID != "proj-1" || gotM.Revision != 5 {
		t.Fatalf("manifest mismatch: %+v", gotM)
	}
}
