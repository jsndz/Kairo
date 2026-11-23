package ownership

import (
	"errors"
	"slices"
	"sync"
)

// Engine manages in-memory clip ownership locks.
type Engine struct {
	mu         sync.RWMutex
	clipOwners map[string]string // clipID -> clientID
}

// New creates an ownership engine.
func New() *Engine {
	return &Engine{
		clipOwners: make(map[string]string),
	}
}

// NewEngine aliases New.
func NewEngine() *Engine {
	return New()
}

// Claim grants unassigned or self-owned clips; rejects peer-owned clips.
func (e *Engine) Claim(clientID string, clipIDs []string) (granted []string, rejected []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	granted = make([]string, 0, len(clipIDs))
	rejected = make([]string, 0)

	if clientID == "" {
		return granted, append(rejected, clipIDs...)
	}

	seen := make(map[string]bool, len(clipIDs))
	for _, clipID := range clipIDs {
		if clipID == "" || seen[clipID] {
			continue
		}
		seen[clipID] = true

		owner, exists := e.clipOwners[clipID]
		if !exists {
			e.clipOwners[clipID] = clientID
			granted = append(granted, clipID)
		} else if owner == clientID {
			granted = append(granted, clipID)
		} else {
			rejected = append(rejected, clipID)
		}
	}

	return granted, rejected
}

// Release unlocks clips owned by clientID.
func (e *Engine) Release(clientID string, clipIDs []string) (released []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	released = make([]string, 0, len(clipIDs))
	if clientID == "" {
		return released
	}

	for _, clipID := range clipIDs {
		if e.clipOwners[clipID] == clientID {
			delete(e.clipOwners, clipID)
			released = append(released, clipID)
		}
	}
	return released
}

// ReleaseClient clears all locks held by clientID.
func (e *Engine) ReleaseClient(clientID string) (released []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	released = make([]string, 0)
	if clientID == "" {
		return released
	}

	for clipID, owner := range e.clipOwners {
		if owner == clientID {
			delete(e.clipOwners, clipID)
			released = append(released, clipID)
		}
	}
	slices.Sort(released)
	return released
}

// Transfer reassigns clipID to toClient.
func (e *Engine) Transfer(clipID, toClient string) error {
	if clipID == "" || toClient == "" {
		return errors.New("clipID and toClient are required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.clipOwners[clipID] = toClient
	return nil
}

// IsAllowed checks if clientID has write access to clipID.
func (e *Engine) IsAllowed(clientID, clipID string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	owner, exists := e.clipOwners[clipID]
	if !exists || owner == clientID {
		return true
	}
	return false
}

// GetOwner returns current owner of clipID.
func (e *Engine) GetOwner(clipID string) (clientID string, ok bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	owner, exists := e.clipOwners[clipID]
	return owner, exists
}

// AllOwners returns an immutable snapshot of all claimed clips.
func (e *Engine) AllOwners() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	snapshot := make(map[string]string, len(e.clipOwners))
	for k, v := range e.clipOwners {
		snapshot[k] = v
	}
	return snapshot
}
