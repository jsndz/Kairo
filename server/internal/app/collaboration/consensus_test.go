package collaboration_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"server/internal/app/collaboration"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

func makeEdit(id, kind, data string) timeline.Edit {
	rev := uint64(1)
	return timeline.Edit{
		ID:               id,
		Type:             kind,
		Payload:          json.RawMessage(data),
		ExpectedRevision: &rev,
	}
}

func TestNewCoordinator(t *testing.T) {
	c := collaboration.NewCoordinator()
	if c == nil {
		t.Fatal("expected non-nil Coordinator from New()")
	}

	if p, ok := c.ActiveProposal(); ok || p != nil {
		t.Fatalf("expected no active proposal initially, got %v", p)
	}

	c2 := collaboration.NewCoordinator()
	if c2 == nil {
		t.Fatal("expected non-nil Coordinator from NewCoordinator()")
	}
}

func TestPropose_BasicAndActive(t *testing.T) {
	c := collaboration.NewCoordinator()

	edit := makeEdit("edit-track", "add-track", `{"trackId":"video-2"}`)
	p, err := c.Propose("prop-1", "alice", edit, "Add Video Track 2", 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error proposing: %v", err)
	}

	if p.ID != "prop-1" || p.ProposerID != "alice" || p.Description != "Add Video Track 2" {
		t.Fatalf("unexpected proposal metadata: %+v", p)
	}

	if !p.Votes["alice"] {
		t.Fatalf("expected proposer alice to automatically vote true, got %v", p.Votes)
	}

	active, ok := c.ActiveProposal()
	if !ok || active == nil || active.ID != "prop-1" {
		t.Fatalf("expected prop-1 active, got %+v", active)
	}

	// Concurrent proposal while one is active should be rejected
	edit2 := makeEdit("edit-reset", "reset", `{}`)
	_, err2 := c.Propose("prop-2", "bob", edit2, "Reset project", 10*time.Second)
	if err2 == nil {
		t.Fatal("expected error proposing while another proposal is in progress")
	}
}

func TestVote_UnanimousApproval(t *testing.T) {
	c := collaboration.NewCoordinator()

	edit := makeEdit("edit-1", "add-track", `{"trackId":"video-2"}`)
	_, err := c.Propose("p1", "alice", edit, "Add Track", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Bob approves
	resolved, approved, op, err := c.Vote("p1", "bob", true)
	if err != nil {
		t.Fatalf("unexpected error voting: %v", err)
	}
	if !resolved {
		t.Fatal("expected resolution on 2/2 approvals")
	}
	if !approved {
		t.Fatal("expected proposal to be approved")
	}
	if op == nil || op.ID != "edit-1" {
		t.Fatalf("expected resolved op edit-1, got %v", op)
	}

	// Proposal should now be cleared
	if active, ok := c.ActiveProposal(); ok || active != nil {
		t.Fatalf("expected proposal to be cleared after resolution, got %v", active)
	}
}

func TestVote_PeerRejection(t *testing.T) {
	c := collaboration.NewCoordinator()

	edit := makeEdit("edit-2", "reset", `{}`)
	_, err := c.Propose("p2", "alice", edit, "Reset Project", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Bob rejects
	resolved, approved, op, err := c.Vote("p2", "bob", false)
	if err != nil {
		t.Fatalf("unexpected error voting: %v", err)
	}
	if !resolved {
		t.Fatal("expected resolution on peer rejection")
	}
	if approved {
		t.Fatal("expected proposal to be rejected")
	}
	if op == nil || op.ID != "edit-2" {
		t.Fatalf("expected op returned on rejection, got %v", op)
	}

	// Proposal should now be cleared
	if _, ok := c.ActiveProposal(); ok {
		t.Fatal("expected proposal cleared after rejection")
	}
}

func TestVote_ProposerSelfRejection(t *testing.T) {
	c := collaboration.NewCoordinator()

	edit := makeEdit("edit-3", "rename", `{"name":"New Name"}`)
	_, err := c.Propose("p3", "alice", edit, "Rename", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Proposer changes their mind and cancels by voting false
	resolved, approved, _, err := c.Vote("p3", "alice", false)
	if err != nil {
		t.Fatalf("unexpected error voting: %v", err)
	}
	if !resolved || approved {
		t.Fatalf("expected resolved=true and approved=false, got resolved=%v approved=%v", resolved, approved)
	}
}

func TestVote_NotFoundOrExpired(t *testing.T) {
	c := collaboration.NewCoordinator()

	_, _, _, err := c.Vote("non-existent", "bob", true)
	if err == nil {
		t.Fatal("expected error voting on non-existent proposal")
	}
}

func TestCancel(t *testing.T) {
	c := collaboration.NewCoordinator()

	edit := makeEdit("edit-4", "add-track", `{"trackId":"video-3"}`)
	_, err := c.Propose("p4", "alice", edit, "Add Video 3", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	p, ok := c.Cancel("p4")
	if !ok || p == nil || p.ID != "p4" {
		t.Fatalf("expected p4 canceled, got p=%v, ok=%v", p, ok)
	}

	if _, ok := c.ActiveProposal(); ok {
		t.Fatal("expected active proposal empty after cancel")
	}

	// Canceling again returns false
	if _, ok := c.Cancel("p4"); ok {
		t.Fatal("expected second cancel to return false")
	}
}

func TestTimeout_TriggersCallback(t *testing.T) {
	c := collaboration.NewCoordinator()

	timeoutCh := make(chan string, 1)
	c.SetTimeoutCallback(func(p *ports.Proposal) {
		timeoutCh <- p.ID
	})

	edit := makeEdit("edit-5", "reset", `{}`)
	_, err := c.Propose("p5", "alice", edit, "Reset", 25*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case id := <-timeoutCh:
		if id != "p5" {
			t.Fatalf("expected timed out proposal p5, got %s", id)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for timeout callback")
	}

	// Proposal should now be cleared
	if _, ok := c.ActiveProposal(); ok {
		t.Fatal("expected proposal cleared after timeout")
	}
}

func TestConcurrentVotes(t *testing.T) {
	c := collaboration.NewCoordinator()

	edit := makeEdit("edit-concurrent", "add-track", `{"trackId":"video-4"}`)
	_, err := c.Propose("p-conc", "alice", edit, "Add Track", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	const voters = 10
	wg.Add(voters)

	resolvedCount := 0
	var mu sync.Mutex

	for i := 0; i < voters; i++ {
		go func(idx int) {
			defer wg.Done()
			voter := string(rune('B' + idx))
			resolved, approved, _, _ := c.Vote("p-conc", voter, true)
			if resolved && approved {
				mu.Lock()
				resolvedCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if resolvedCount != 1 {
		t.Fatalf("expected exactly 1 vote resolution, got %d", resolvedCount)
	}
}
