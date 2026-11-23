package project

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
	"server/internal/ports"
)

// State is the project snapshot exposed to clients.
type State = ports.State

// Service manages authoritative project state and mutations.
type Service struct {
	mu        sync.Mutex
	meta      record.Metadata
	catalog   ports.CatalogRepository
	log       ports.OperationLogRepository
	objects   ports.ObjectStore
	byID      map[string]record.Record
	hist      history.History
	revision  uint64
	blocked   bool
	maxUpload int64
}

// Open loads metadata and replays the operation log.
func Open(catalog ports.CatalogRepository, log ports.OperationLogRepository, objects ports.ObjectStore, maxUpload int64) (*Service, error) {
	s := &Service{
		catalog:   catalog,
		log:       log,
		objects:   objects,
		byID:      make(map[string]record.Record),
		hist:      history.New(timeline.Empty()),
		maxUpload: maxUpload,
	}

	// 1. Read log
	rows, err := log.Read()
	if err != nil {
		return nil, fmt.Errorf("read operations: %w", err)
	}

	// 2. Load metadata
	meta, err := s.loadOrCreateMetadata(len(rows) > 0)
	if err != nil {
		return nil, err
	}
	s.meta = meta

	// 3. Replay operations
	if err := s.replayLog(rows); err != nil {
		return nil, err
	}

	return s, nil
}

// loadOrCreateMetadata reads or initializes project metadata.
func (s *Service) loadOrCreateMetadata(hasExistingLog bool) (record.Metadata, error) {
	meta, err := s.catalog.Read()
	if err == nil {
		return meta, validateMetadata(meta)
	}

	// Initialize metadata if missing
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

// validateMetadata checks metadata format.
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

// replayLog rebuilds state from log records.
func (s *Service) replayLog(rows []record.Record) error {
	for _, row := range rows {
		// Verify sequence continuity
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

// Get returns current project state snapshot.
func (s *Service) Get() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state()
}

// ProjectID returns project ID.
func (s *Service) ProjectID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.meta.ID
}

// GetOpsSince returns operations after sinceSeq.
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

// Commit appends an edit to the log and applies it to state.
func (s *Service) Commit(e timeline.Edit) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.blocked {
		return State{}, domErrors.ErrUnavailable
	}

	// 1. Validate headers
	if strings.TrimSpace(e.ID) == "" || len(e.ID) > 128 || e.ExpectedRevision == nil {
		return State{}, fmt.Errorf("%w: operation ID and revision are required", domErrors.ErrInvalid)
	}

	// 2. Canonicalize payload
	canonicalPayload, err := canonicalizePayload(e.Payload)
	if err != nil {
		return State{}, err
	}

	// 3. Idempotency check
	if existing, exists := s.byID[e.ID]; exists {
		if existing.Type != e.Type || !bytes.Equal(existing.Payload, canonicalPayload) {
			return State{}, fmt.Errorf("%w: operation ID already used with different payload", domErrors.ErrConflict)
		}
		return s.state(), nil
	}

	// 4. Revision check
	if *e.ExpectedRevision != s.revision {
		return State{}, fmt.Errorf("%w: project revision changed", domErrors.ErrConflict)
	}

	// 5. Build record
	row := record.Record{
		ID:        e.ID,
		ProjectID: s.meta.ID,
		Sequence:  s.revision + 1,
		Timestamp: time.Now().UTC(),
		Type:      e.Type,
		Payload:   canonicalPayload,
	}

	// 6. Reduce next state
	nextHistory, err := s.reduce(row)
	if err != nil {
		return State{}, fmt.Errorf("%w: %v", domErrors.ErrInvalid, err)
	}

	// 7. Append to log
	if err := s.log.Append(row); err != nil {
		s.blocked = true
		return State{}, fmt.Errorf("%w: append operation: %v", domErrors.ErrUnavailable, err)
	}

	// 8. Update state
	s.accept(row, nextHistory)
	return s.state(), nil
}

// OwnershipChecker verifies clip edit permissions.
type OwnershipChecker = ports.OwnershipChecker

// CommitAuthor validates ownership before committing the edit.
func (s *Service) CommitAuthor(e timeline.Edit, authorID string, checker OwnershipChecker) (State, error) {
	if checker != nil && authorID != "" {
		if clipID, ok := timeline.ExtractClipID(e); ok {
			if !checker.IsAllowed(authorID, clipID) {
				return State{}, fmt.Errorf("%w: clip %s is owned by peer", domErrors.ErrConflict, clipID)
			}
		}
	}
	return s.Commit(e)
}

// canonicalizePayload validates and formats JSON payload.
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

// reduce applies an operation to history.
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

// reduceUndo reverts to previous timeline state.
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

// reduceTimelineEdit applies edit and updates undo/redo history.
func (s *Service) reduceTimelineEdit(h history.History, row record.Record) (history.History, error) {
	nextTimeline, err := timeline.Apply(h.Current, s.meta.Assets, row)
	if err != nil {
		return h, err
	}

	// Record history only on state change
	if !reflect.DeepEqual(h.Current, nextTimeline) {
		h.Append(h.Current)
		h.Future = nil
		h.Current = nextTimeline
	}
	return h, nil
}

// accept records operation and updates memory state.
func (s *Service) accept(r record.Record, next history.History) {
	s.byID[r.ID] = r
	s.revision = r.Sequence
	s.hist = next
}

// state serializes active state. Caller must hold s.mu.
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
