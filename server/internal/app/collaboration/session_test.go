package collaboration_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"server/internal/app/collaboration"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

type mockEmitter struct {
	mu           sync.Mutex
	broadcasts   []collaboration.OutMessage
	clientMsgs   map[string][]collaboration.OutMessage
	roomEmptied  bool
}

func newMockEmitter() *mockEmitter {
	return &mockEmitter{
		clientMsgs: make(map[string][]collaboration.OutMessage),
	}
}

func (m *mockEmitter) Broadcast(msg collaboration.OutMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcasts = append(m.broadcasts, msg)
}

func (m *mockEmitter) BroadcastExcept(excludeClientID string, msg collaboration.OutMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcasts = append(m.broadcasts, msg)
}

func (m *mockEmitter) SendToClient(clientID string, msg collaboration.OutMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientMsgs[clientID] = append(m.clientMsgs[clientID], msg)
}

func (m *mockEmitter) RoomEmpty(_ string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.roomEmptied = true
}

type mockProjectService struct {
	mu    sync.Mutex
	state ports.State
}

func newMockProjectService() *mockProjectService {
	s := &mockProjectService{}
	s.state.ID = "proj-test"
	s.state.Revision = 1
	return s
}

func (m *mockProjectService) Get() ports.State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *mockProjectService) GetOpsSince(_ uint64) ([]record.Record, error) {
	return nil, nil
}

func (m *mockProjectService) ProjectID() string {
	return "proj-test"
}

func (m *mockProjectService) Metadata() record.ProjectMetadata {
	m.mu.Lock()
	defer m.mu.Unlock()
	return record.ProjectMetadata{
		ID:       m.state.ID,
		Name:     m.state.Project.Name,
		Revision: m.state.Revision,
	}
}

func (m *mockProjectService) Commit(edit timeline.Edit) (ports.State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Revision++
	return m.state, nil
}

func (m *mockProjectService) CommitAuthor(edit timeline.Edit, _ string, _ ports.OwnershipChecker) (ports.State, error) {
	return m.Commit(edit)
}

func TestRoomSession_Capacity(t *testing.T) {
	emitter := newMockEmitter()
	svc := newMockProjectService()
	session := collaboration.NewRoomSession("room-1", svc, emitter)

	if !session.CanJoin("c1") {
		t.Fatal("expected c1 can join")
	}
	if err := session.Join("c1"); err != nil {
		t.Fatalf("c1 join: %v", err)
	}

	if !session.CanJoin("c2") {
		t.Fatal("expected c2 can join")
	}
	if err := session.Join("c2"); err != nil {
		t.Fatalf("c2 join: %v", err)
	}

	// Third client cannot join
	if session.CanJoin("c3") {
		t.Fatal("expected c3 cannot join")
	}
	if err := session.Join("c3"); err == nil {
		t.Fatal("expected error on c3 join, got nil")
	}
}

func TestRoomSession_Ownership(t *testing.T) {
	emitter := newMockEmitter()
	svc := newMockProjectService()
	session := collaboration.NewRoomSession("room-1", svc, emitter)

	_ = session.Join("c1")
	_ = session.Join("c2")

	// c1 claims clip-a
	session.HandleMessage("c1", collaboration.InMessage{
		Type:    "claim_request",
		ReqID:   "req-1",
		ClipIDs: []string{"clip-a"},
	})

	emitter.mu.Lock()
	c1Msgs := emitter.clientMsgs["c1"]
	emitter.mu.Unlock()

	var claimAck *collaboration.OutMessage
	for _, m := range c1Msgs {
		if m.Type == "claim_ack" && m.ReqID == "req-1" {
			cp := m
			claimAck = &cp
			break
		}
	}
	if claimAck == nil || len(claimAck.Granted) != 1 || claimAck.Granted[0] != "clip-a" {
		t.Fatalf("expected claim_ack granting clip-a, got %+v", claimAck)
	}

	// c2 tries to claim clip-a -> rejected
	session.HandleMessage("c2", collaboration.InMessage{
		Type:    "claim_request",
		ReqID:   "req-2",
		ClipIDs: []string{"clip-a"},
	})

	emitter.mu.Lock()
	c2Msgs := emitter.clientMsgs["c2"]
	emitter.mu.Unlock()

	var c2ClaimAck *collaboration.OutMessage
	for _, m := range c2Msgs {
		if m.Type == "claim_ack" && m.ReqID == "req-2" {
			cp := m
			c2ClaimAck = &cp
			break
		}
	}
	if c2ClaimAck == nil || len(c2ClaimAck.Rejected) != 1 || c2ClaimAck.Rejected[0] != "clip-a" {
		t.Fatalf("expected claim_ack rejecting clip-a for c2, got %+v", c2ClaimAck)
	}
}

func TestRoomSession_Consensus(t *testing.T) {
	emitter := newMockEmitter()
	svc := newMockProjectService()
	session := collaboration.NewRoomSession("room-1", svc, emitter)

	_ = session.Join("c1")
	_ = session.Join("c2")

	// Propose structural op
	edit := timeline.Edit{
		ID:      "op-1",
		Type:    "add-track",
		Payload: json.RawMessage(`{"kind":"video"}`),
	}
	session.HandleMessage("c1", collaboration.InMessage{
		Type:        "consensus_propose",
		ReqID:       "req-p",
		ProposalID:  "prop-1",
		Op:          &edit,
		Description: "Add video track",
	})

	// Peer c2 votes approve
	approved := true
	session.HandleMessage("c2", collaboration.InMessage{
		Type:       "consensus_vote",
		ReqID:      "req-v",
		ProposalID: "prop-1",
		Approved:   &approved,
	})

	emitter.mu.Lock()
	broadcasts := emitter.broadcasts
	emitter.mu.Unlock()

	var resolved *collaboration.OutMessage
	for _, m := range broadcasts {
		if m.Type == "consensus_resolved" && m.ProposalID == "prop-1" {
			cp := m
			resolved = &cp
			break
		}
	}
	if resolved == nil || resolved.Approved == nil || !*resolved.Approved {
		t.Fatalf("expected consensus_resolved approved=true, got %+v", resolved)
	}
}

func TestRoomSession_GracePeriodReclaim(t *testing.T) {
	emitter := newMockEmitter()
	svc := newMockProjectService()
	session := collaboration.NewRoomSession("room-1", svc, emitter)
	session.SetGraceDuration(20 * time.Millisecond)

	_ = session.Join("c1")
	session.HandleMessage("c1", collaboration.InMessage{
		Type:    "claim_request",
		ReqID:   "req-1",
		ClipIDs: []string{"clip-x"},
	})

	// c1 leaves -> starts 20ms grace timer
	session.Leave("c1")

	// Immediately, clip-x is still claimed
	if owner, ok := session.Ownership().GetOwner("clip-x"); !ok || owner != "c1" {
		t.Fatalf("expected clip-x still owned during grace period, got %s, %v", owner, ok)
	}

	// Wait for grace period to expire
	time.Sleep(50 * time.Millisecond)

	// Locks should be released
	if _, ok := session.Ownership().GetOwner("clip-x"); ok {
		t.Fatal("expected clip-x ownership to be released after grace period")
	}
}
