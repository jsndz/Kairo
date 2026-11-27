package project

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// Service manages authoritative project state and mutations.
type Service struct {
	mu              sync.Mutex
	meta            record.Metadata
	catalog         ports.CatalogRepository
	log             ports.OperationLogRepository
	objects         ports.ObjectStore
	byID            map[string]record.Record
	hist            history.History
	revision        uint64
	lastSnapshotRev uint64
	blocked         bool
	maxUpload       int64
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

	// 1. Check for materialized snapshot (O(1) fast hydration)
	var snapSeq uint64
	if snapRepo, ok := log.(ports.SnapshotRepository); ok {
		snap, err := snapRepo.GetLatestSnapshot()
		if err == nil && snap != nil && snap.Sequence > 0 {
			var snapState struct {
				Revision uint64            `json:"revision"`
				Timeline timeline.Timeline `json:"timeline"`
			}
			if err := json.Unmarshal(snap.State, &snapState); err == nil && snapState.Revision > 0 {
				s.revision = snapState.Revision
				s.hist = history.New(snapState.Timeline)
				s.lastSnapshotRev = snapState.Revision
				snapSeq = snapState.Revision
			}
		}
	}

	// 2. Read log records (delta if snapshot exists, full log otherwise)
	var rows []record.Record
	var err error
	if snapSeq > 0 {
		if deltaLog, ok := log.(record.DeltaLogReader); ok {
			rows, err = deltaLog.ReadSince(snapSeq)
		} else {
			all, readErr := log.Read()
			if readErr != nil {
				return nil, fmt.Errorf("read operations: %w", readErr)
			}
			for _, r := range all {
				if r.Sequence > snapSeq {
					rows = append(rows, r)
				}
			}
		}
	} else {
		rows, err = log.Read()
	}
	if err != nil {
		return nil, fmt.Errorf("read operations: %w", err)
	}

	// 3. Load metadata
	meta, err := s.loadOrCreateMetadata(snapSeq > 0 || len(rows) > 0)
	if err != nil {
		return nil, err
	}
	s.meta = meta

	// 4. Replay operations
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
func (s *Service) Get() ports.State {
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

// Metadata returns lightweight project metadata for catalogs and dashboards.
func (s *Service) Metadata() record.ProjectMetadata {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state()
	name := st.Project.Name
	if name == "" {
		name = "Untitled Project"
	}
	return record.ProjectMetadata{
		ID:        st.ID,
		Name:      name,
		Revision:  st.Revision,
		Clips:     len(st.Project.Clips),
		Assets:    len(st.Project.Assets),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}



// Close flushes and safely releases resources.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Save final snapshot if new operations were committed
	if s.revision > s.lastSnapshotRev {
		if snapRepo, ok := s.log.(ports.SnapshotRepository); ok {
			snapPayload, err := json.Marshal(map[string]any{
				"revision": s.revision,
				"timeline": s.hist.Current,
			})
			if err == nil {
				_ = snapRepo.SaveSnapshot(s.revision, snapPayload)
				s.lastSnapshotRev = s.revision
			}
		}
	}

	// 2. Release storage resources (e.g. SQLite connection pool)
	if closer, ok := s.log.(io.Closer); ok {
		_ = closer.Close()
	}
	if closer, ok := s.catalog.(io.Closer); ok {
		_ = closer.Close()
	}
	return nil
}

// GetOpsSince returns operations after sinceSeq.
func (s *Service) GetOpsSince(sinceSeq uint64) ([]record.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if deltaLog, ok := s.log.(record.DeltaLogReader); ok {
		return deltaLog.ReadSince(sinceSeq)
	}

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
func (s *Service) Commit(e timeline.Edit) (ports.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.blocked {
		return ports.State{}, domErrors.ErrUnavailable
	}

	// 1. Validate headers
	if strings.TrimSpace(e.ID) == "" || len(e.ID) > 128 || e.ExpectedRevision == nil {
		return ports.State{}, fmt.Errorf("%w: operation ID and revision are required", domErrors.ErrInvalid)
	}

	// 2. Canonicalize payload
	canonicalPayload, err := canonicalizePayload(e.Payload)
	if err != nil {
		return ports.State{}, err
	}

	// 3. Idempotency check
	if existing, exists := s.byID[e.ID]; exists {
		if existing.Type != e.Type || !bytes.Equal(existing.Payload, canonicalPayload) {
			return ports.State{}, fmt.Errorf("%w: operation ID already used with different payload", domErrors.ErrConflict)
		}
		return s.state(), nil
	}

	// 4. Revision check
	if *e.ExpectedRevision != s.revision {
		return ports.State{}, fmt.Errorf("%w: project revision changed", domErrors.ErrConflict)
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
		return ports.State{}, fmt.Errorf("%w: %v", domErrors.ErrInvalid, err)
	}

	// 7. Append to log
	if err := s.log.Append(row); err != nil {
		s.blocked = true
		return ports.State{}, fmt.Errorf("%w: append operation: %v", domErrors.ErrUnavailable, err)
	}

	// 8. Update state
	s.accept(row, nextHistory)

	// Check if periodic snapshot checkpoint is due (every 500 operations)
	if s.revision-s.lastSnapshotRev >= 500 {
		if snapRepo, ok := s.log.(ports.SnapshotRepository); ok {
			snapPayload, err := json.Marshal(map[string]any{
				"revision": s.revision,
				"timeline": s.hist.Current,
			})
			if err == nil {
				_ = snapRepo.SaveSnapshot(s.revision, snapPayload)
				s.lastSnapshotRev = s.revision
			}
		}
	}

	return s.state(), nil
}

// CommitAuthor validates ownership before committing the edit.
func (s *Service) CommitAuthor(e timeline.Edit, authorID string, checker ports.OwnershipChecker) (ports.State, error) {
	if checker != nil && authorID != "" {
		if clipID, ok := timeline.ExtractClipID(e); ok {
			if !checker.IsAllowed(authorID, clipID) {
				return ports.State{}, fmt.Errorf("%w: clip %s is owned by peer", domErrors.ErrConflict, clipID)
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
func (s *Service) state() ports.State {
	out := ports.State{
		ID:       s.meta.ID,
		Revision: s.revision,
		CanUndo:  s.hist.CanUndo(),
		CanRedo:  s.hist.CanRedo(),
	}
	out.Project.Timeline = s.hist.Current
	out.Project.Assets = maps.Clone(s.meta.Assets)
	return out
}
