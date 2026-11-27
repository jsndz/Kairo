package project

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	sqliteStorage "server/internal/adapters/outbound/storage/sqlite"
	"server/internal/domain/asset"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

var (
	_ ports.ProjectManager  = (*Manager)(nil)
	_ ports.ProjectResolver = (*Manager)(nil)
)

// Manager coordinates project creation, lazy loading, and lifecycle.
type Manager struct {
	mu          sync.RWMutex
	projectsDir string
	indexStore  ports.ProjectIndexRepository
	objects     ports.ObjectStore
	maxUpload   int64

	active     map[string]*Service
	lastActive map[string]time.Time
}

// NewManager initializes a ProjectManager at projectsDir.
func NewManager(projectsDir string, objects ports.ObjectStore, maxUpload int64) (*Manager, error) {
	if err := os.MkdirAll(projectsDir, 0700); err != nil {
		return nil, fmt.Errorf("create projects directory: %w", err)
	}

	indexDBPath := filepath.Join(projectsDir, "index.db")

	indexStore, err := sqliteStorage.OpenIndex(indexDBPath)
	if err != nil {
		return nil, fmt.Errorf("open project index db: %w", err)
	}

	return &Manager{
		projectsDir: projectsDir,
		indexStore:  indexStore,
		objects:     objects,
		maxUpload:   maxUpload,
		active:      make(map[string]*Service),
		lastActive:  make(map[string]time.Time),
	}, nil
}

// List returns metadata for all available projects, refreshing any active in-memory states.
func (m *Manager) List(ctx context.Context) ([]record.ProjectMetadata, error) {
	summaries, err := m.indexStore.Read()
	if err != nil {
		return nil, fmt.Errorf("read project index: %w", err)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Update summaries with live state if active in memory
	for i, s := range summaries {
		if activeSvc, ok := m.active[s.ID]; ok {
			live := activeSvc.Metadata()
			summaries[i].Name = live.Name
			summaries[i].Revision = live.Revision
			summaries[i].Clips = live.Clips
			summaries[i].Assets = live.Assets
			summaries[i].UpdatedAt = live.UpdatedAt
		}
	}

	return summaries, nil
}

// Create generates a new project directory, initializes metadata & log, and returns an active service.
func (m *Manager) Create(ctx context.Context, name string) (record.ProjectMetadata, ports.ProjectInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "" {
		name = "Untitled Project"
	}

	id := rand.Text()
	projDir := filepath.Join(m.projectsDir, id)
	if err := os.MkdirAll(projDir, 0700); err != nil {
		return record.ProjectMetadata{}, nil, fmt.Errorf("create project dir: %w", err)
	}

	dbPath := filepath.Join(projDir, "project.db")
	pdb, err := sqliteStorage.OpenProject(dbPath)
	if err != nil {
		return record.ProjectMetadata{}, nil, fmt.Errorf("open new project db: %w", err)
	}

	if err := pdb.Catalog().Save(record.Metadata{
		ID:     id,
		Assets: make(map[string]asset.Asset),
	}); err != nil {
		_ = pdb.Close()
		return record.ProjectMetadata{}, nil, fmt.Errorf("initialize project metadata: %w", err)
	}

	svc, err := Open(pdb.Catalog(), pdb.Log(), m.objects, m.maxUpload)
	if err != nil {
		_ = pdb.Close()
		return record.ProjectMetadata{}, nil, fmt.Errorf("open new project: %w", err)
	}

	// Set initial project name if specified
	if name != "Untitled Project" && name != "" {
		payload, _ := json.Marshal(map[string]string{"name": name})
		initRev := uint64(0)
		_, _ = svc.Commit(timeline.Edit{
			ID:               "init-rename",
			Type:             "rename",
			Payload:          payload,
			ExpectedRevision: &initRev,
		})
	}

	now := time.Now().UTC()
	summary := record.ProjectMetadata{
		ID:        id,
		Name:      name,
		Revision:  svc.Get().Revision,
		Clips:     0,
		Assets:    0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := m.indexStore.Upsert(summary); err != nil {
		return record.ProjectMetadata{}, nil, fmt.Errorf("save project index: %w", err)
	}

	m.active[id] = svc
	m.lastActive[id] = now

	return summary, svc, nil
}

// Get returns an active project service, loading it from disk if not already in memory.
func (m *Manager) Get(ctx context.Context, id string) (ports.ProjectInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Check in-memory cache
	if svc, ok := m.active[id]; ok {
		m.lastActive[id] = time.Now()
		return svc, nil
	}

	// 2. Verify project database exists on disk
	projDir := filepath.Join(m.projectsDir, id)
	dbPath := filepath.Join(projDir, "project.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("project %q not found", id)
	}

	// 3. Open project SQLite database
	pdb, err := sqliteStorage.OpenProject(dbPath)
	if err != nil {
		return nil, fmt.Errorf("load project db %q: %w", id, err)
	}

	svc, err := Open(pdb.Catalog(), pdb.Log(), m.objects, m.maxUpload)
	if err != nil {
		_ = pdb.Close()
		return nil, fmt.Errorf("load project %q from disk: %w", id, err)
	}

	now := time.Now()
	m.active[id] = svc
	m.lastActive[id] = now

	// Update index with current name/revision
	summary := svc.Metadata()
	_ = m.indexStore.Upsert(summary)

	return svc, nil
}

// Delete removes a project from memory and deletes its storage on disk.
func (m *Manager) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if svc, ok := m.active[id]; ok {
		_ = svc.Close()
		delete(m.active, id)
		delete(m.lastActive, id)
	}

	_ = m.indexStore.Delete(id)

	projDir := filepath.Join(m.projectsDir, id)
	return os.RemoveAll(projDir)
}

// Touch updates the last activity timestamp for a project.
func (m *Manager) Touch(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastActive[id] = time.Now()
}

// ResolveReader implements ports.ProjectResolver.
func (m *Manager) ResolveReader(ctx context.Context, projectID string) (ports.ProjectReader, error) {
	return m.Get(ctx, projectID)
}

// EvictIdle unloads projects from memory that have been inactive longer than timeout.
func (m *Manager) EvictIdle(timeout time.Duration, isBusy func(projectID string) bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, last := range m.lastActive {
		if now.Sub(last) > timeout {
			if isBusy != nil && isBusy(id) {
				continue
			}
			if svc, ok := m.active[id]; ok {
				_ = svc.Close()
				delete(m.active, id)
				delete(m.lastActive, id)
			}
		}
	}
}

// Close flushes and shuts down the global index store.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, svc := range m.active {
		_ = svc.Close()
		delete(m.active, id)
	}
	if m.indexStore != nil {
		return m.indexStore.Close()
	}
	return nil
}
