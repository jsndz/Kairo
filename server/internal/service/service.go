package service

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	"server/internal/domain/history"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
)

// State represents the complete project snapshot exposed to the client.
type State struct {
	ID       string `json:"id"`
	Revision uint64 `json:"revision"`
	Project  struct {
		timeline.Timeline
		Assets map[string]asset.Asset `json:"assets"`
	} `json:"project"`
	CanUndo bool `json:"canUndo"`
	CanRedo bool `json:"canRedo"`
}

// Service is the authoritative single-writer coordinator managing the project state.
type Service struct {
	mu        sync.Mutex
	meta      record.Metadata
	catalog   record.Catalog
	log       record.Log
	objects   asset.Objects
	byID      map[string]record.Record
	hist      history.History
	revision  uint64
	blocked   bool
	maxUpload int64
}

// Open initializes the service, loads/initializes metadata, and replays all log operations.
func Open(catalog record.Catalog, log record.Log, objects asset.Objects, maxUpload int64) (*Service, error) {
	s := &Service{
		catalog:   catalog,
		log:       log,
		objects:   objects,
		byID:      make(map[string]record.Record),
		hist:      history.New(timeline.Empty()),
		maxUpload: maxUpload,
	}

	// 1. Read existing operation log
	rows, err := log.Read()
	if err != nil {
		return nil, fmt.Errorf("read operations: %w", err)
	}

	// 2. Load or create project metadata
	meta, err := s.loadOrCreateMetadata(len(rows) > 0)
	if err != nil {
		return nil, err
	}
	s.meta = meta

	// 3. Replay operations to reconstruct timeline state
	if err := s.replayLog(rows); err != nil {
		return nil, err
	}

	return s, nil
}

// loadOrCreateMetadata loads metadata from the catalog, or initializes a new project if none exists.
func (s *Service) loadOrCreateMetadata(hasExistingLog bool) (record.Metadata, error) {
	meta, err := s.catalog.Read()
	if err == nil {
		return meta, validateMetadata(meta)
	}

	// If metadata file doesn't exist, create initial metadata
	if errors.Is(err, os.ErrNotExist) {
		if hasExistingLog {
			return record.Metadata{}, fmt.Errorf("project metadata is missing for an existing log")
		}
		newMeta := record.Metadata{
			ID:     rand.Text(),
			Assets: make(map[string]asset.Asset),
		}
		if err := s.catalog.Save(newMeta); err != nil {
			return record.Metadata{}, fmt.Errorf("initialize metadata: %w", err)
		}
		return newMeta, nil
	}

	return record.Metadata{}, fmt.Errorf("read metadata: %w", err)
}

// validateMetadata checks that project ID and asset metadata are well-formed.
func validateMetadata(meta record.Metadata) error {
	if meta.ID == "" || meta.Assets == nil {
		return fmt.Errorf("invalid project metadata")
	}

	for id, a := range meta.Assets {
		if id == "" || id != a.ID || a.Frames < 1 || a.ObjectKey == "" || a.Bucket == "" || a.CreatedAt.IsZero() {
			return fmt.Errorf("invalid metadata for asset %q", id)
		}
		if a.Kind != "video" && a.Kind != "audio" && a.Kind != "image" {
			return fmt.Errorf("invalid media kind for asset %q", id)
		}
		if a.Status != "pending" && a.Status != "uploading" && a.Status != "failed" && a.Status != "completed" {
			return fmt.Errorf("invalid status for asset %q", id)
		}
	}
	return nil
}

// replayLog sequentially executes operations to rebuild in-memory state.
func (s *Service) replayLog(rows []record.Record) error {
	for _, row := range rows {
		// Verify strict sequence continuity and integrity
		if row.ProjectID != s.meta.ID || row.Sequence != s.revision+1 || row.ID == "" || row.Timestamp.IsZero() {
			return fmt.Errorf("invalid operation at sequence %d", s.revision+1)
		}
		if _, exists := s.byID[row.ID]; exists {
			return fmt.Errorf("duplicate operation ID %q at sequence %d", row.ID, row.Sequence)
		}

		next, err := s.reduce(row)
		if err != nil {
			return fmt.Errorf("replay sequence %d: %w", row.Sequence, err)
		}
		s.accept(row, next)
	}
	return nil
}

// Get returns the current project state snapshot under lock.
func (s *Service) Get() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state()
}

// ProjectID returns the current project's ID.
func (s *Service) ProjectID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.meta.ID
}

// GetOpsSince returns all operations with sequence > sinceSeq.
func (s *Service) GetOpsSince(sinceSeq uint64) ([]record.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.log.Read()
	if err != nil {
		return nil, err
	}
	var ops []record.Record
	for _, r := range all {
		if r.Sequence > sinceSeq {
			ops = append(ops, r)
		}
	}
	return ops, nil
}

// Commit validates an incoming edit, appends it to the durable log, and applies it to state.
func (s *Service) Commit(e timeline.Edit) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.blocked {
		return State{}, domErrors.ErrUnavailable
	}

	// 1. Validate request headers
	if strings.TrimSpace(e.ID) == "" || len(e.ID) > 128 || e.ExpectedRevision == nil {
		return State{}, fmt.Errorf("%w: operation ID and revision are required", domErrors.ErrInvalid)
	}

	// 2. Canonicalize JSON payload
	canonicalPayload, err := canonicalizePayload(e.Payload)
	if err != nil {
		return State{}, err
	}

	// 3. Idempotent check: return state if this exact edit was already committed
	if existing, exists := s.byID[e.ID]; exists {
		if existing.Type != e.Type || !bytes.Equal(existing.Payload, canonicalPayload) {
			return State{}, fmt.Errorf("%w: operation ID already used with different payload", domErrors.ErrConflict)
		}
		return s.state(), nil
	}

	// 4. Optimistic concurrency check: client revision must match current server revision
	if *e.ExpectedRevision != s.revision {
		return State{}, fmt.Errorf("%w: project revision changed", domErrors.ErrConflict)
	}

	// 5. Construct new record
	row := record.Record{
		ID:        e.ID,
		ProjectID: s.meta.ID,
		Sequence:  s.revision + 1,
		Timestamp: time.Now().UTC(),
		Type:      e.Type,
		Payload:   canonicalPayload,
	}

	// 6. Calculate next timeline state
	nextHistory, err := s.reduce(row)
	if err != nil {
		return State{}, fmt.Errorf("%w: %v", domErrors.ErrInvalid, err)
	}

	// 7. Durably append to disk before acknowledging
	if err := s.log.Append(row); err != nil {
		s.blocked = true // Block subsequent writes to prevent inconsistent state
		return State{}, fmt.Errorf("%w: append operation: %v", domErrors.ErrUnavailable, err)
	}

	// 8. Commit to memory
	s.accept(row, nextHistory)
	return s.state(), nil
}

// canonicalizePayload validates that raw is a JSON object and strips arbitrary whitespace.
func canonicalizePayload(raw json.RawMessage) ([]byte, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return nil, fmt.Errorf("%w: payload must be a JSON object", domErrors.ErrInvalid)
	}

	var data any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil, domErrors.ErrInvalid
	}
	return json.Marshal(data)
}

// reduce applies an operation to history and returns the next history snapshot.
func (s *Service) reduce(row record.Record) (history.History, error) {
	h := s.hist

	switch row.Type {
	case "undo":
		return s.reduceUndo(h, row.Payload)

	case "redo":
		return s.reduceRedo(h, row.Payload)

	default:
		return s.reduceTimelineEdit(h, row)
	}
}

// reduceUndo reverts to the previous timeline state in history.
func (s *Service) reduceUndo(h history.History, payload json.RawMessage) (history.History, error) {
	if err := timeline.DecodePayload(payload, &struct{}{}); err != nil {
		return h, err
	}
	if !h.CanUndo() {
		return h, fmt.Errorf("nothing to undo")
	}

	h.PushFuture(h.Current)
	h.Current, h.Past = h.Past[len(h.Past)-1], h.Past[:len(h.Past)-1]
	return h, nil
}

// reduceRedo re-applies an undone timeline state.
func (s *Service) reduceRedo(h history.History, payload json.RawMessage) (history.History, error) {
	if err := timeline.DecodePayload(payload, &struct{}{}); err != nil {
		return h, err
	}
	if !h.CanRedo() {
		return h, fmt.Errorf("nothing to redo")
	}

	h.Append(h.Current)
	h.Current, h.Future = h.Future[len(h.Future)-1], h.Future[:len(h.Future)-1]
	return h, nil
}

// reduceTimelineEdit applies a domain edit, appends previous state to undo history, and clears redo.
func (s *Service) reduceTimelineEdit(h history.History, row record.Record) (history.History, error) {
	nextTimeline, err := timeline.Apply(h.Current, s.meta.Assets, row)
	if err != nil {
		return h, err
	}

	// Only record history if state actually changed
	if !reflect.DeepEqual(h.Current, nextTimeline) {
		h.Append(h.Current)
		h.Future = nil
		h.Current = nextTimeline
	}
	return h, nil
}

// accept records an operation and adopts the new history state in memory.
func (s *Service) accept(r record.Record, next history.History) {
	s.byID[r.ID] = r
	s.revision = r.Sequence
	s.hist = next
}

// state serializes the active state for API responses. Caller must hold s.mu.
func (s *Service) state() State {
	out := State{
		ID:       s.meta.ID,
		Revision: s.revision,
		CanUndo:  s.hist.CanUndo(),
		CanRedo:  s.hist.CanRedo(),
	}
	out.Project.Timeline = s.hist.Current
	out.Project.Assets = maps.Clone(s.meta.Assets)
	return out
}
