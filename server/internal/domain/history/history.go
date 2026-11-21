package history

import (
	"slices"

	"server/internal/domain/timeline"
)

// History maintains the active timeline alongside undo and redo stacks.
type History struct {
	Current      timeline.Timeline
	Past, Future []timeline.Timeline
}

// New creates an initial history instance with a starting timeline.
func New(initial timeline.Timeline) History {
	return History{Current: initial}
}

// CanUndo returns true if there are prior snapshots in the past stack.
func (h History) CanUndo() bool {
	return len(h.Past) > 0
}

// CanRedo returns true if there are undone snapshots in the future stack.
func (h History) CanRedo() bool {
	return len(h.Future) > 0
}

// Append records a snapshot into the past history stack, keeping at most 50 entries.
func (h *History) Append(t timeline.Timeline) {
	if len(h.Past) == 50 {
		h.Past = h.Past[1:]
	}
	h.Past = append(slices.Clone(h.Past), t)
}

// PushFuture records a snapshot into the future (redo) stack, keeping at most 50 entries.
func (h *History) PushFuture(t timeline.Timeline) {
	if len(h.Future) == 50 {
		h.Future = h.Future[1:]
	}
	h.Future = append(slices.Clone(h.Future), t)
}
