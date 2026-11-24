package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	domErrors "server/internal/domain/errors"
	"server/internal/domain/export"
	"server/internal/ports"
)

var _ ports.JobRepository = (*FSRepository)(nil)

// FSRepository implements ports.JobRepository using the local filesystem.
type FSRepository struct {
	mu       sync.RWMutex
	baseDir  string
	jobsDir  string
	mansDir  string
}

// NewFSRepository creates a new filesystem repository and ensures directories exist.
func NewFSRepository(baseDir string) (*FSRepository, error) {
	jobsDir := filepath.Join(baseDir, "jobs")
	mansDir := filepath.Join(baseDir, "manifests")

	if err := os.MkdirAll(jobsDir, 0700); err != nil {
		return nil, fmt.Errorf("create jobs dir: %w", err)
	}
	if err := os.MkdirAll(mansDir, 0700); err != nil {
		return nil, fmt.Errorf("create manifests dir: %w", err)
	}

	return &FSRepository{
		baseDir: baseDir,
		jobsDir: jobsDir,
		mansDir: mansDir,
	}, nil
}

// SaveJob writes a job atomically to <jobsDir>/<id>.json.
func (r *FSRepository) SaveJob(job export.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.atomicWrite(filepath.Join(r.jobsDir, job.ID+".json"), job)
}

// GetJob reads a job record by ID.
func (r *FSRepository) GetJob(id string) (export.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var job export.Job
	err := r.readJSON(filepath.Join(r.jobsDir, id+".json"), &job)
	if err != nil {
		return export.Job{}, err
	}
	return job, nil
}

// GetJobByRequestID finds an existing job matching projectID and requestID.
func (r *FSRepository) GetJobByRequestID(projectID, requestID string) (export.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, err := os.ReadDir(r.jobsDir)
	if err != nil {
		return export.Job{}, fmt.Errorf("%w: read jobs dir: %v", domErrors.ErrUnavailable, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var job export.Job
		if err := r.readJSON(filepath.Join(r.jobsDir, entry.Name()), &job); err == nil {
			if job.ProjectID == projectID && job.RequestID == requestID {
				return job, nil
			}
		}
	}

	return export.Job{}, domErrors.ErrNotFound
}

// ListJobs returns all stored jobs.
func (r *FSRepository) ListJobs() ([]export.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, err := os.ReadDir(r.jobsDir)
	if err != nil {
		return nil, fmt.Errorf("%w: read jobs dir: %v", domErrors.ErrUnavailable, err)
	}

	var jobs []export.Job
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var job export.Job
		if err := r.readJSON(filepath.Join(r.jobsDir, entry.Name()), &job); err == nil {
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

// SaveManifest writes an immutable manifest to <mansDir>/<id>.json.
func (r *FSRepository) SaveManifest(manifest export.Manifest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Use projectID + revision or a manifest ID
	key := fmt.Sprintf("%s_%d.json", manifest.ProjectID, manifest.Revision)
	return r.atomicWrite(filepath.Join(r.mansDir, key), manifest)
}

// GetManifest reads an immutable manifest for a project revision.
func (r *FSRepository) GetManifest(id string) (export.Manifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// id can be either direct filename key or projectID_revision
	filename := id
	if filepath.Ext(filename) != ".json" {
		filename += ".json"
	}

	var m export.Manifest
	err := r.readJSON(filepath.Join(r.mansDir, filename), &m)
	if err != nil {
		return export.Manifest{}, err
	}
	return m, nil
}

func (r *FSRepository) readJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domErrors.ErrNotFound
		}
		return fmt.Errorf("%w: read file: %v", domErrors.ErrUnavailable, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("%w: parse json: %v", domErrors.ErrInvalid, err)
	}
	return nil
}

func (r *FSRepository) atomicWrite(targetPath string, src any) error {
	data, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: marshal json: %v", domErrors.ErrInvalid, err)
	}

	dir := filepath.Dir(targetPath)
	tmpFile, err := os.CreateTemp(dir, "tmp-*.json")
	if err != nil {
		return fmt.Errorf("%w: create temp file: %v", domErrors.ErrUnavailable, err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("%w: write temp file: %v", domErrors.ErrUnavailable, err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("%w: sync temp file: %v", domErrors.ErrUnavailable, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("%w: close temp file: %v", domErrors.ErrUnavailable, err)
	}

	if err := os.Rename(tmpName, targetPath); err != nil {
		return fmt.Errorf("%w: rename to target: %v", domErrors.ErrUnavailable, err)
	}
	return nil
}
