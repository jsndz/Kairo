package catalog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"server/internal/adapters/outbound/storage/fs"
	"server/internal/domain/record"
)

// ProjectIndexStore manages the global index of projects stored in JSON.
type ProjectIndexStore struct {
	mu   sync.RWMutex
	path string
}

// NewProjectIndexStore initializes a ProjectIndexStore at the given path.
func NewProjectIndexStore(path string) *ProjectIndexStore {
	return &ProjectIndexStore{path: path}
}

// Read returns all project summaries from disk. If the file does not exist, it returns an empty slice.
func (s *ProjectIndexStore) Read() ([]record.ProjectMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []record.ProjectMetadata{}, nil
		}
		return nil, err
	}

	var summaries []record.ProjectMetadata
	if err := json.Unmarshal(b, &summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}

// Save writes project summaries atomically to disk.
func (s *ProjectIndexStore) Save(summaries []record.ProjectMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	f, err := os.CreateTemp(dir, ".index-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(summaries); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), s.path); err != nil {
		return err
	}
	return fs.SyncDir(dir)
}

// Upsert adds or updates a project summary in the index.
func (s *ProjectIndexStore) Upsert(summary record.ProjectMetadata) error {
	summaries, err := s.Read()
	if err != nil {
		return err
	}

	found := false
	for i, p := range summaries {
		if p.ID == summary.ID {
			summaries[i] = summary
			found = true
			break
		}
	}
	if !found {
		// Prepend new project so newest appears first
		summaries = append([]record.ProjectMetadata{summary}, summaries...)
	}

	return s.Save(summaries)
}

// Delete removes a project by ID from the index.
func (s *ProjectIndexStore) Delete(id string) error {
	summaries, err := s.Read()
	if err != nil {
		return err
	}

	filtered := make([]record.ProjectMetadata, 0, len(summaries))
	for _, p := range summaries {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}

	return s.Save(filtered)
}
