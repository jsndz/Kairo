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

// Open initializes the service, loads metadata, and replays all log operations sequentially.
func Open(catalog record.Catalog, log record.Log, objects asset.Objects, maxUpload int64) (*Service, error) {
	s := &Service{
		catalog:   catalog,
		log:       log,
		objects:   objects,
		byID:      make(map[string]record.Record),
		hist:      history.New(timeline.Empty()),
		maxUpload: maxUpload,
	}

	meta, err := catalog.Read()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read metadata: %w", err)
	}

	rows, readErr := log.Read()
	if readErr != nil {
		return nil, fmt.Errorf("read operations: %w", readErr)
	}

	if errors.Is(err, os.ErrNotExist) {
		if len(rows) != 0 {
			return nil, fmt.Errorf("project metadata is missing for an existing log")
		}
		meta = record.Metadata{ID: rand.Text(), Assets: make(map[string]asset.Asset)}
		if err = catalog.Save(meta); err != nil {
			return nil, fmt.Errorf("initialize metadata: %w", err)
		}
	}

	if meta.ID == "" || meta.Assets == nil {
		return nil, fmt.Errorf("invalid project metadata")
	}

	for id, a := range meta.Assets {
		if id == "" || id != a.ID || a.Frames < 1 || a.ObjectKey == "" || a.Bucket == "" || a.CreatedAt.IsZero() ||
			(a.Kind != "video" && a.Kind != "audio" && a.Kind != "image") ||
			(a.Status != "pending" && a.Status != "uploading" && a.Status != "failed" && a.Status != "completed") {
			return nil, fmt.Errorf("invalid metadata for asset %q", id)
		}
	}
	s.meta = meta

	// Replay operations sequentially to reconstruct project state
	for _, row := range rows {
		if row.ProjectID != meta.ID || row.Sequence != s.revision+1 || row.ID == "" || row.Timestamp.IsZero() || s.byID[row.ID].ID != "" {
			return nil, fmt.Errorf("invalid operation at sequence %d", s.revision+1)
		}
		next, err := s.reduce(row)
		if err != nil {
			return nil, fmt.Errorf("replay sequence %d: %w", row.Sequence, err)
		}
		s.accept(row, next)
	}

	return s, nil
}

// Get returns the current state snapshot.
func (s *Service) Get() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state()
}

// Commit verifies revision, validates the edit, appends it to the log, and updates state.
func (s *Service) Commit(e timeline.Edit) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked {
		return State{}, domErrors.ErrUnavailable
	}

	if strings.TrimSpace(e.ID) == "" || len(e.ID) > 128 || e.ExpectedRevision == nil {
		return State{}, fmt.Errorf("%w: operation ID and revision are required", domErrors.ErrInvalid)
	}

	var fields map[string]json.RawMessage
	if json.Unmarshal(e.Payload, &fields) != nil || fields == nil {
		return State{}, fmt.Errorf("%w: payload must be an object", domErrors.ErrInvalid)
	}

	var data any
	decoder := json.NewDecoder(bytes.NewReader(e.Payload))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return State{}, domErrors.ErrInvalid
	}
	canonical, _ := json.Marshal(data)

	// Idempotent duplicate check
	if existing, ok := s.byID[e.ID]; ok {
		if existing.Type != e.Type || !bytes.Equal(existing.Payload, canonical) {
			return State{}, fmt.Errorf("%w: operation ID already used", domErrors.ErrConflict)
		}
		return s.state(), nil
	}

	if *e.ExpectedRevision != s.revision {
		return State{}, fmt.Errorf("%w: project revision changed", domErrors.ErrConflict)
	}

	row := record.Record{
		ID:        e.ID,
		ProjectID: s.meta.ID,
		Sequence:  s.revision + 1,
		Timestamp: time.Now().UTC(),
		Type:      e.Type,
		Payload:   canonical,
	}

	next, err := s.reduce(row)
	if err != nil {
		return State{}, fmt.Errorf("%w: %v", domErrors.ErrInvalid, err)
	}

	if err = s.log.Append(row); err != nil {
		s.blocked = true
		return State{}, fmt.Errorf("%w: append operation: %v", domErrors.ErrUnavailable, err)
	}

	s.accept(row, next)
	return s.state(), nil
}

// reduce calculates the next timeline and undo history state for a given operation.
func (s *Service) reduce(row record.Record) (history.History, error) {
	h := s.hist
	switch row.Type {
	case "undo", "redo":
		if err := timeline.DecodePayload(row.Payload, &struct{}{}); err != nil {
			return h, err
		}
		if row.Type == "undo" && h.CanUndo() {
			h.PushFuture(h.Current)
			h.Current, h.Past = h.Past[len(h.Past)-1], h.Past[:len(h.Past)-1]
		} else if row.Type == "redo" && h.CanRedo() {
			h.Append(h.Current)
			h.Current, h.Future = h.Future[len(h.Future)-1], h.Future[:len(h.Future)-1]
		} else {
			return h, fmt.Errorf("nothing to %s", row.Type)
		}
	default:
		next, err := timeline.Apply(h.Current, s.meta.Assets, row)
		if err != nil {
			return h, err
		}
		if !reflect.DeepEqual(h.Current, next) {
			h.Append(h.Current)
			h.Future = nil
			h.Current = next
		}
	}
	return h, nil
}

// accept commits an operation record and the new history in memory.
func (s *Service) accept(r record.Record, next history.History) {
	s.byID[r.ID] = r
	s.revision = r.Sequence
	s.hist = next
}

// state builds the State DTO. The caller must hold s.mu.
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
