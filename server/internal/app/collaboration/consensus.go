package collaboration

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"server/internal/domain/timeline"
	"server/internal/ports"
)



// CloneProposal returns a deep copy of the proposal.
func CloneProposal(p *ports.Proposal) *ports.Proposal {
	if p == nil {
		return nil
	}
	votesCopy := make(map[string]bool, len(p.Votes))
	for k, v := range p.Votes {
		votesCopy[k] = v
	}
	cp := *p
	cp.Votes = votesCopy
	return &cp
}

// Coordinator manages two-party consensus proposals and quorum validation.
type Coordinator struct {
	mu        sync.RWMutex
	proposals map[string]*ports.Proposal
	timers    map[string]*time.Timer
	onTimeout func(p *ports.Proposal)
}

// NewCoordinator creates an empty consensus coordinator.
func NewCoordinator() *Coordinator {
	return &Coordinator{
		proposals: make(map[string]*ports.Proposal),
		timers:    make(map[string]*time.Timer),
	}
}

// SetTimeoutCallback sets a hook triggered when a proposal expires.
func (c *Coordinator) SetTimeoutCallback(fn func(p *ports.Proposal)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onTimeout = fn
}

// Propose initiates a new consensus proposal with an expiration timer.
func (c *Coordinator) Propose(proposalID, proposerID string, op timeline.Edit, description string, timeout time.Duration) (*ports.Proposal, error) {
	if proposalID == "" || proposerID == "" {
		return nil, errors.New("proposalID and proposerID are required")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cleanupExpiredLocked()

	if len(c.proposals) > 0 {
		for _, p := range c.proposals {
			return nil, fmt.Errorf("proposal %s is already in progress", p.ID)
		}
	}

	now := time.Now().UTC()
	p := &ports.Proposal{
		ID:          proposalID,
		ProposerID:  proposerID,
		Op:          op,
		Description: description,
		CreatedAt:   now,
		ExpiresAt:   now.Add(timeout),
		Votes: map[string]bool{
			proposerID: true,
		},
	}
	c.proposals[proposalID] = p

	timer := time.AfterFunc(timeout, func() {
		c.handleTimeout(proposalID)
	})
	c.timers[proposalID] = timer

	return CloneProposal(p), nil
}

// Vote records a vote from voterID.
// 2/2 approvals resolve true/approved; any rejection resolves true/rejected.
func (c *Coordinator) Vote(proposalID, voterID string, approve bool) (resolved bool, approved bool, op *timeline.Edit, err error) {
	if proposalID == "" || voterID == "" {
		return false, false, nil, errors.New("proposalID and voterID are required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	p, exists := c.proposals[proposalID]
	if !exists || time.Now().UTC().After(p.ExpiresAt) {
		return false, false, nil, fmt.Errorf("proposal not found or expired")
	}

	p.Votes[voterID] = approve

	if !approve {
		c.stopTimerLocked(proposalID)
		delete(c.proposals, proposalID)
		opCopy := p.Op
		return true, false, &opCopy, nil
	}

	approvals := 0
	for _, v := range p.Votes {
		if v {
			approvals++
		}
	}

	if approvals >= 2 {
		c.stopTimerLocked(proposalID)
		delete(c.proposals, proposalID)
		opCopy := p.Op
		return true, true, &opCopy, nil
	}

	return false, false, nil, nil
}

// Cancel aborts a proposal and stops its timer.
func (c *Coordinator) Cancel(proposalID string) (*ports.Proposal, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	p, exists := c.proposals[proposalID]
	if !exists {
		return nil, false
	}

	c.stopTimerLocked(proposalID)
	delete(c.proposals, proposalID)
	return CloneProposal(p), true
}

// ActiveProposal returns the currently active unexpired proposal, if any.
func (c *Coordinator) ActiveProposal() (*ports.Proposal, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now().UTC()
	for _, p := range c.proposals {
		if now.Before(p.ExpiresAt) {
			return CloneProposal(p), true
		}
	}
	return nil, false
}

func (c *Coordinator) handleTimeout(proposalID string) {
	c.mu.Lock()
	p, exists := c.proposals[proposalID]
	if exists {
		delete(c.proposals, proposalID)
		c.stopTimerLocked(proposalID)
	}
	cb := c.onTimeout
	c.mu.Unlock()

	if exists && cb != nil {
		cb(CloneProposal(p))
	}
}

func (c *Coordinator) stopTimerLocked(id string) {
	if t, ok := c.timers[id]; ok {
		t.Stop()
		delete(c.timers, id)
	}
}

func (c *Coordinator) cleanupExpiredLocked() {
	now := time.Now().UTC()
	for id, p := range c.proposals {
		if now.After(p.ExpiresAt) {
			c.stopTimerLocked(id)
			delete(c.proposals, id)
		}
	}
}
