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

	catalogStorage "server/internal/adapters/outbound/storage/catalog"
	"server/internal/adapters/outbound/storage/fs"
	"server/internal/adapters/outbound/storage/jsonl"
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
	indexStore  *catalogStorage.ProjectIndexStore
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

	indexPath := filepath.Join(projectsDir, "index.json")
	indexStore := catalogStorage.NewProjectIndexStore(indexPath)

	return &Manager{
		projectsDir: projectsDir,
		indexStore:  indexStore,
		objects:     objects,
		maxUpload:   maxUpload,
		active:      make(map[string]*Service),
		lastActive:  make(map[string]time.Time),
	}, nil
}

// List returns summaries for all available projects, refreshing any active in-memory states.
func (m *Manager) List(ctx context.Context) ([]record.ProjectSummary, error) {
	summaries, err := m.indexStore.Read()
	if err != nil {
		return nil, fmt.Errorf("read project index: %w", err)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Update summaries with live state if active in memory
	for i, s := range summaries {
		if activeSvc, ok := m.active[s.ID]; ok {
			live := activeSvc.Summary()
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
func (m *Manager) Create(ctx context.Context, name string) (record.ProjectSummary, ports.ProjectInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "" {
		name = "Untitled Project"
	}

	id := rand.Text()
	projDir := filepath.Join(m.projectsDir, id)
	if err := os.MkdirAll(projDir, 0700); err != nil {
		return record.ProjectSummary{}, nil, fmt.Errorf("create project dir: %w", err)
	}

	cat := catalogStorage.JSONCatalog{Path: filepath.Join(projDir, "metadata.json")}
	if err := cat.Save(record.Metadata{
		ID:     id,
		Assets: make(map[string]asset.Asset),
	}); err != nil {
		return record.ProjectSummary{}, nil, fmt.Errorf("initialize project metadata: %w", err)
	}
	lg := jsonl.JSONL{Path: filepath.Join(projDir, "operations.jsonl")}

	svc, err := Open(cat, lg, m.objects, m.maxUpload)
	if err != nil {
		return record.ProjectSummary{}, nil, fmt.Errorf("open new project: %w", err)
	}

	// Set initial project name if specified
	if name != "Untitled Project" {
		payload, _ := json.Marshal(map[string]string{"name": name})
		_, _ = svc.Commit(timeline.Edit{
			Type:    "rename",
			Payload: payload,
		})
	}

	now := time.Now().UTC()
	summary := record.ProjectSummary{
		ID:        id,
		Name:      name,
		Revision:  svc.Get().Revision,
		Clips:     0,
		Assets:    0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := m.indexStore.Upsert(summary); err != nil {
		return record.ProjectSummary{}, nil, fmt.Errorf("save project index: %w", err)
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

	// 2. Verify project directory exists on disk
	projDir := filepath.Join(m.projectsDir, id)
	metaPath := filepath.Join(projDir, "metadata.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("project %q not found", id)
	}

	// 3. Open project
	cat := catalogStorage.JSONCatalog{Path: metaPath}
	lg := jsonl.JSONL{Path: filepath.Join(projDir, "operations.jsonl")}

	svc, err := Open(cat, lg, m.objects, m.maxUpload)
	if err != nil {
		return nil, fmt.Errorf("load project %q from disk: %w", id, err)
	}

	now := time.Now()
	m.active[id] = svc
	m.lastActive[id] = now

	// Update index with current name/revision
	summary := svc.Summary()
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

// MigrateLegacy checks for a root metadata.json and operations.jsonl and migrates them into projects/{id}/.
func (m *Manager) MigrateLegacy(legacyDataDir string) error {
	metaPath := filepath.Join(legacyDataDir, "metadata.json")
	opsPath := filepath.Join(legacyDataDir, "operations.jsonl")

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil // No legacy files to migrate
	}

	cat := catalogStorage.JSONCatalog{Path: metaPath}
	meta, err := cat.Read()
	if err != nil || meta.ID == "" {
		return nil // Invalid or unreadable metadata
	}

	projDir := filepath.Join(m.projectsDir, meta.ID)
	if err := os.MkdirAll(projDir, 0700); err != nil {
		return fmt.Errorf("create legacy migrated project dir: %w", err)
	}

	destMeta := filepath.Join(projDir, "metadata.json")
	destOps := filepath.Join(projDir, "operations.jsonl")

	if err := os.Rename(metaPath, destMeta); err != nil {
		return fmt.Errorf("migrate legacy metadata.json: %w", err)
	}
	if _, err := os.Stat(opsPath); err == nil {
		if err := os.Rename(opsPath, destOps); err != nil {
			return fmt.Errorf("migrate legacy operations.jsonl: %w", err)
		}
	}

	_ = fs.SyncDir(projDir)
	_ = fs.SyncDir(legacyDataDir)

	// Load and record in index
	svc, err := Open(
		catalogStorage.JSONCatalog{Path: destMeta},
		jsonl.JSONL{Path: destOps},
		m.objects,
		m.maxUpload,
	)
	if err == nil {
		summary := svc.Summary()
		_ = m.indexStore.Upsert(summary)
	}

	return nil
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
