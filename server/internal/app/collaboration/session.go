package collaboration

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

// Cursor represents cursor coordinates on the timeline.
type Cursor struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// InMessage is the envelope for incoming messages from a collaboration client.
type InMessage struct {
	Type             string         `json:"type"`
	ReqID            string         `json:"req_id,omitempty"`
	Op               *timeline.Edit `json:"op,omitempty"`
	ExpectedRevision *uint64        `json:"expected_revision,omitempty"`
	Cursor           *Cursor        `json:"cursor,omitempty"`
	Playhead         *int           `json:"playhead,omitempty"`
	Selected         *string        `json:"selected,omitempty"`
	LastSeenSeq      *uint64        `json:"last_seen_seq,omitempty"`

	// Stage 3 fields
	ClipIDs     []string `json:"clip_ids,omitempty"`
	ClipID      string   `json:"clip_id,omitempty"`
	Approved    *bool    `json:"approved,omitempty"`
	ProposalID  string   `json:"proposal_id,omitempty"`
	Description string   `json:"description,omitempty"`
}

// UnmarshalJSON handles both nested and flat op formats.
func (m *InMessage) UnmarshalJSON(data []byte) error {
	type InMessageShadow InMessage
	var shadow InMessageShadow
	if err := json.Unmarshal(data, &shadow); err != nil {
		return err
	}
	*m = InMessage(shadow)

	var aux struct {
		RawOp json.RawMessage `json:"op,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err == nil && aux.RawOp != nil {
		if m.Op == nil {
			var edit timeline.Edit
			if err := json.Unmarshal(aux.RawOp, &edit); err == nil {
				m.Op = &edit
			}
		}
		if m.Op != nil && (len(m.Op.Payload) == 0 || string(m.Op.Payload) == "null") {
			var flat map[string]json.RawMessage
			if err := json.Unmarshal(aux.RawOp, &flat); err == nil {
				delete(flat, "id")
				delete(flat, "type")
				delete(flat, "expectedRevision")
				if len(flat) > 0 {
					m.Op.Payload, _ = json.Marshal(flat)
				} else {
					m.Op.Payload = json.RawMessage("{}")
				}
			}
		}
	}
	return nil
}

// OutMessage is the envelope for outgoing messages sent to collaboration clients.
type OutMessage struct {
	Type         string            `json:"type"`
	ReqID        string            `json:"req_id,omitempty"`
	Seq          uint64            `json:"seq,omitempty"`
	Status       string            `json:"status,omitempty"`
	Code         string            `json:"code,omitempty"`
	Message      string            `json:"message,omitempty"`
	CurrentSeq   uint64            `json:"current_seq,omitempty"`
	Op           *timeline.Edit    `json:"op,omitempty"`
	Author       string            `json:"author,omitempty"`
	ClientID     string            `json:"client_id,omitempty"`
	Cursor       *Cursor           `json:"cursor,omitempty"`
	Playhead     *int              `json:"playhead,omitempty"`
	Selected     *string           `json:"selected,omitempty"`
	Asset        *asset.Asset      `json:"asset,omitempty"`
	AssetID      string            `json:"asset_id,omitempty"`
	Ops          []record.Record   `json:"ops,omitempty"`
	State        *ports.State      `json:"state,omitempty"`

	// Stage 3 fields
	Granted      []string          `json:"granted,omitempty"`
	Rejected     []string          `json:"rejected,omitempty"`
	ClipIDs      []string          `json:"clip_ids,omitempty"`
	ClipID       string            `json:"clip_id,omitempty"`
	FromClientID string            `json:"from_client_id,omitempty"`
	TimeoutMS    int               `json:"timeout_ms,omitempty"`
	Approved     *bool             `json:"approved,omitempty"`
	NewOwner     *string           `json:"new_owner,omitempty"`
	ProposalID   string            `json:"proposal_id,omitempty"`
	ProposerID   string            `json:"proposer_id,omitempty"`
	Description  string            `json:"description,omitempty"`
	Owners       map[string]string `json:"owners,omitempty"`
}

// SessionEmitter abstracts network delivery of collaboration events.
type SessionEmitter interface {
	Broadcast(msg OutMessage)
	BroadcastExcept(excludeClientID string, msg OutMessage)
	SendToClient(clientID string, msg OutMessage)
	RoomEmpty(roomID string)
}

type pendingPermission struct {
	reqID        string
	fromClientID string
	ownerID      string
	clipID       string
	timer        *time.Timer
}

// RoomSession coordinates multi-user collaboration business logic, consensus, and lock ownership.
type RoomSession struct {
	ID                 string
	service            ports.ProjectService
	ownership          *Engine
	consensus          *Coordinator
	clients            map[string]bool
	reclaimTimers      map[string]*time.Timer
	pendingPermissions map[string]*pendingPermission
	graceDuration      time.Duration
	emitter            SessionEmitter
	mu                 sync.RWMutex
}

// NewRoomSession constructs a pure collaboration session with domain business rules.
func NewRoomSession(id string, svc ports.ProjectService, emitter SessionEmitter) *RoomSession {
	s := &RoomSession{
		ID:                 id,
		service:            svc,
		ownership:          New(),
		consensus:          NewCoordinator(),
		clients:            make(map[string]bool),
		reclaimTimers:      make(map[string]*time.Timer),
		pendingPermissions: make(map[string]*pendingPermission),
		graceDuration:      15 * time.Second,
		emitter:            emitter,
	}

	s.consensus.SetTimeoutCallback(func(p *Proposal) {
		approved := false
		s.Broadcast(OutMessage{
			Type:       "consensus_resolved",
			ProposalID: p.ID,
			Approved:   &approved,
			Message:    "Proposal timed out after 30 seconds",
		})
	})

	return s
}

// SetGraceDuration overrides the disconnect grace period.
func (s *RoomSession) SetGraceDuration(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.graceDuration = d
}

// Ownership returns the underlying ownership engine.
func (s *RoomSession) Ownership() *Engine {
	return s.ownership
}

// Consensus returns the underlying consensus coordinator.
func (s *RoomSession) Consensus() *Coordinator {
	return s.consensus
}

// CanJoin checks whether client can join the room.
func (s *RoomSession) CanJoin(clientID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if clientID != "" {
		if s.clients[clientID] {
			return false
		}
		if _, hasGrace := s.reclaimTimers[clientID]; hasGrace {
			return true
		}
	}
	return len(s.clients) < 2
}

// Join admits a client ID to the session, cancels active grace timers, and sends initial greeting.
func (s *RoomSession) Join(clientID string) error {
	s.mu.Lock()

	// If reconnecting during grace period, cancel reclaim timer
	if clientID != "" {
		if t, ok := s.reclaimTimers[clientID]; ok {
			t.Stop()
			delete(s.reclaimTimers, clientID)
		}
	}

	if len(s.clients) >= 2 {
		s.mu.Unlock()
		return errors.New("room is full: maximum 2 users allowed")
	}

	s.clients[clientID] = true
	s.mu.Unlock()

	// Send initial greeting including current active ownership map
	currentState := s.service.Get()
	if s.emitter != nil {
		s.emitter.SendToClient(clientID, OutMessage{
			Type:       "init",
			ClientID:   clientID,
			CurrentSeq: currentState.Revision,
			State:      &currentState,
			Owners:     s.ownership.AllOwners(),
		})

		// Tell connecting client about existing peers
		s.mu.RLock()
		for id := range s.clients {
			if id != clientID {
				s.emitter.SendToClient(clientID, OutMessage{
					Type:     "peer_joined",
					ClientID: id,
				})
			}
		}
		s.mu.RUnlock()

		// Notify existing peer that client joined
		s.emitter.BroadcastExcept(clientID, OutMessage{
			Type:     "peer_joined",
			ClientID: clientID,
		})
	}

	return nil
}

// Leave removes a client from the session and starts disconnect grace timer.
func (s *RoomSession) Leave(clientID string) {
	s.mu.Lock()
	delete(s.clients, clientID)
	remaining := len(s.clients)

	// Clean up pending permission requests involving departed client
	for reqID, req := range s.pendingPermissions {
		if req.fromClientID == clientID || req.ownerID == clientID {
			if req.timer != nil {
				req.timer.Stop()
			}
			delete(s.pendingPermissions, reqID)
		}
	}

	// Start disconnect grace timer for locks
	if _, exists := s.reclaimTimers[clientID]; !exists {
		dur := s.graceDuration
		timer := time.AfterFunc(dur, func() {
			s.handleGraceExpiration(clientID)
		})
		s.reclaimTimers[clientID] = timer
	}
	s.mu.Unlock()

	if remaining > 0 && s.emitter != nil {
		s.emitter.Broadcast(OutMessage{
			Type:     "peer_left",
			ClientID: clientID,
		})
	}
}

func (s *RoomSession) handleGraceExpiration(clientID string) {
	s.mu.Lock()
	delete(s.reclaimTimers, clientID)
	remainingClients := len(s.clients)
	remainingTimers := len(s.reclaimTimers)
	s.mu.Unlock()

	released := s.ownership.ReleaseClient(clientID)
	if len(released) > 0 && s.emitter != nil {
		s.emitter.Broadcast(OutMessage{
			Type:    "release_broadcast",
			ClipIDs: released,
		})
	}

	if remainingClients == 0 && remainingTimers == 0 && s.emitter != nil {
		s.emitter.RoomEmpty(s.ID)
	}
}

// Broadcast sends an OutMessage to all session clients.
func (s *RoomSession) Broadcast(msg OutMessage) {
	if s.emitter != nil {
		s.emitter.Broadcast(msg)
	}
}

// BroadcastExcept sends an OutMessage to all session clients except excludeClientID.
func (s *RoomSession) BroadcastExcept(excludeClientID string, msg OutMessage) {
	if s.emitter != nil {
		s.emitter.BroadcastExcept(excludeClientID, msg)
	}
}

// BroadcastAsset forwards asset updates.
func (s *RoomSession) BroadcastAsset(a asset.Asset) {
	s.Broadcast(OutMessage{
		Type:  "asset_updated",
		Asset: &a,
	})
}

// BroadcastAssetDeleted forwards asset deletion events.
func (s *RoomSession) BroadcastAssetDeleted(assetID string) {
	s.Broadcast(OutMessage{
		Type:    "asset_deleted",
		AssetID: assetID,
	})
}

// BroadcastOperation forwards operation commits.
func (s *RoomSession) BroadcastOperation(projectID string, edit timeline.Edit, state ports.State, author string) {
	if s.ID == projectID || projectID == "" {
		s.BroadcastExcept(author, OutMessage{
			Type:   "op_broadcast",
			Seq:    state.Revision,
			Op:     &edit,
			Author: author,
			State:  &state,
		})
	}
}

// HandleMessage routes incoming client actions according to domain collaboration rules.
func (s *RoomSession) HandleMessage(clientID string, msg InMessage) {
	switch msg.Type {
	case "op_submit":
		s.handleOpSubmit(clientID, msg)
	case "claim_request":
		s.handleClaimRequest(clientID, msg)
	case "release_request":
		s.handleReleaseRequest(clientID, msg)
	case "permission_request":
		s.handlePermissionRequest(clientID, msg)
	case "permission_response":
		s.handlePermissionResponse(clientID, msg)
	case "consensus_propose":
		s.handleConsensusPropose(clientID, msg)
	case "consensus_vote":
		s.handleConsensusVote(clientID, msg)
	case "presence":
		s.BroadcastExcept(clientID, OutMessage{
			Type:     "presence_broadcast",
			ClientID: clientID,
			Cursor:   msg.Cursor,
			Playhead: msg.Playhead,
			Selected: msg.Selected,
		})
	case "sync_req":
		s.handleSyncReq(clientID, msg)
	}
}

func (s *RoomSession) handleOpSubmit(clientID string, msg InMessage) {
	if msg.Op == nil {
		s.sendMsg(clientID, OutMessage{
			Type:    "op_error",
			ReqID:   msg.ReqID,
			Code:    "invalid",
			Message: "missing operation payload",
		})
		return
	}

	if msg.Op.ID == "" && msg.ReqID != "" {
		msg.Op.ID = msg.ReqID
	}
	if msg.Op.ExpectedRevision == nil && msg.ExpectedRevision != nil {
		msg.Op.ExpectedRevision = msg.ExpectedRevision
	}

	// Structural operations require consensus when a peer is present
	s.mu.RLock()
	hasPeer := len(s.clients) >= 2
	s.mu.RUnlock()
	if hasPeer && (msg.Op.Type == "add-track" || msg.Op.Type == "delete-track" || msg.Op.Type == "reset") {
		s.sendMsg(clientID, OutMessage{
			Type:    "op_error",
			ReqID:   msg.ReqID,
			Code:    "conflict",
			Message: "structural operations require consensus proposal",
		})
		return
	}

	// Authorize edit against clip ownership
	state, err := s.service.CommitAuthor(*msg.Op, clientID, s.ownership)
	if err != nil {
		code := "invalid"
		if errors.Is(err, domErrors.ErrConflict) {
			code = "conflict"
		}
		s.sendMsg(clientID, OutMessage{
			Type:       "op_error",
			ReqID:      msg.ReqID,
			Code:       code,
			CurrentSeq: state.Revision,
			Message:    err.Error(),
		})
		return
	}

	// Auto-claim newly added clips or release deleted clips
	if clipID, ok := timeline.ExtractClipID(*msg.Op); ok {
		if msg.Op.Type == "add" {
			s.ownership.Claim(clientID, []string{clipID})
			s.Broadcast(OutMessage{
				Type:     "claim_broadcast",
				ClientID: clientID,
				ClipIDs:  []string{clipID},
			})
		} else if msg.Op.Type == "delete" {
			s.ownership.Release(clientID, []string{clipID})
			s.Broadcast(OutMessage{
				Type:    "release_broadcast",
				ClipIDs: []string{clipID},
			})
		}
	}

	// Ack author and broadcast to peer
	s.sendMsg(clientID, OutMessage{
		Type:   "op_ack",
		ReqID:  msg.ReqID,
		Seq:    state.Revision,
		Status: "ok",
		State:  &state,
	})

	s.BroadcastExcept(clientID, OutMessage{
		Type:   "op_broadcast",
		Seq:    state.Revision,
		Op:     msg.Op,
		Author: clientID,
		State:  &state,
	})
}

func (s *RoomSession) handleClaimRequest(clientID string, msg InMessage) {
	granted, rejected := s.ownership.Claim(clientID, msg.ClipIDs)

	s.sendMsg(clientID, OutMessage{
		Type:     "claim_ack",
		ReqID:    msg.ReqID,
		Granted:  granted,
		Rejected: rejected,
	})

	if len(granted) > 0 {
		s.Broadcast(OutMessage{
			Type:     "claim_broadcast",
			ClientID: clientID,
			ClipIDs:  granted,
		})
	}
}

func (s *RoomSession) handleReleaseRequest(clientID string, msg InMessage) {
	released := s.ownership.Release(clientID, msg.ClipIDs)
	if len(released) > 0 {
		s.Broadcast(OutMessage{
			Type:    "release_broadcast",
			ClipIDs: released,
		})
	}
}

func (s *RoomSession) handlePermissionRequest(clientID string, msg InMessage) {
	ownerID, exists := s.ownership.GetOwner(msg.ClipID)

	// If unassigned or already owned by requester, grant immediately
	if !exists || ownerID == clientID {
		s.ownership.Claim(clientID, []string{msg.ClipID})
		approved := true
		newOwner := clientID
		s.Broadcast(OutMessage{
			Type:     "permission_resolved",
			ReqID:    msg.ReqID,
			ClipID:   msg.ClipID,
			Approved: &approved,
			NewOwner: &newOwner,
		})
		s.Broadcast(OutMessage{
			Type:     "claim_broadcast",
			ClientID: clientID,
			ClipIDs:  []string{msg.ClipID},
		})
		return
	}

	// Clip is owned by peer
	s.mu.RLock()
	isConnected := s.clients[ownerID]
	s.mu.RUnlock()

	if !isConnected {
		// Peer is disconnected (in grace period)
		approved := false
		s.sendMsg(clientID, OutMessage{
			Type:     "permission_resolved",
			ReqID:    msg.ReqID,
			ClipID:   msg.ClipID,
			Approved: &approved,
		})
		return
	}

	// Register pending request with 30s timeout
	timer := time.AfterFunc(30*time.Second, func() {
		s.mu.Lock()
		req, ok := s.pendingPermissions[msg.ReqID]
		if ok {
			delete(s.pendingPermissions, msg.ReqID)
		}
		s.mu.Unlock()

		if ok {
			approved := false
			s.Broadcast(OutMessage{
				Type:     "permission_resolved",
				ReqID:    req.reqID,
				ClipID:   req.clipID,
				Approved: &approved,
			})
		}
	})

	s.mu.Lock()
	s.pendingPermissions[msg.ReqID] = &pendingPermission{
		reqID:        msg.ReqID,
		fromClientID: clientID,
		ownerID:      ownerID,
		clipID:       msg.ClipID,
		timer:        timer,
	}
	s.mu.Unlock()

	// Send interactive prompt to owner
	s.sendMsg(ownerID, OutMessage{
		Type:         "permission_prompt",
		ReqID:        msg.ReqID,
		FromClientID: clientID,
		ClipID:       msg.ClipID,
		TimeoutMS:    30000,
	})
}

func (s *RoomSession) handlePermissionResponse(clientID string, msg InMessage) {
	s.mu.Lock()
	req, exists := s.pendingPermissions[msg.ReqID]
	if exists {
		if req.timer != nil {
			req.timer.Stop()
		}
		delete(s.pendingPermissions, msg.ReqID)
	}
	s.mu.Unlock()

	if !exists {
		return
	}

	approved := msg.Approved != nil && *msg.Approved
	var newOwner *string
	if approved {
		_ = s.ownership.Transfer(req.clipID, req.fromClientID)
		newOwner = &req.fromClientID
	}

	s.Broadcast(OutMessage{
		Type:     "permission_resolved",
		ReqID:    req.reqID,
		ClipID:   req.clipID,
		Approved: &approved,
		NewOwner: newOwner,
	})

	if approved {
		s.Broadcast(OutMessage{
			Type:     "claim_broadcast",
			ClientID: req.fromClientID,
			ClipIDs:  []string{req.clipID},
		})
	}
}

func (s *RoomSession) handleConsensusPropose(clientID string, msg InMessage) {
	if msg.Op == nil {
		s.sendMsg(clientID, OutMessage{
			Type:    "op_error",
			ReqID:   msg.ReqID,
			Code:    "invalid",
			Message: "missing op for consensus proposal",
		})
		return
	}

	if msg.Op.ID == "" {
		msg.Op.ID = msg.ProposalID
	}
	if len(msg.Op.Payload) == 0 {
		msg.Op.Payload = json.RawMessage("{}")
	}

	p, err := s.consensus.Propose(msg.ProposalID, clientID, *msg.Op, msg.Description, 30*time.Second)
	if err != nil {
		s.sendMsg(clientID, OutMessage{
			Type:    "op_error",
			ReqID:   msg.ReqID,
			Code:    "conflict",
			Message: err.Error(),
		})
		return
	}

	// Prompt peer to vote
	s.BroadcastExcept(clientID, OutMessage{
		Type:        "consensus_prompt",
		ProposalID:  p.ID,
		ProposerID:  clientID,
		Op:          &p.Op,
		Description: p.Description,
		TimeoutMS:   30000,
	})
}

func (s *RoomSession) handleConsensusVote(clientID string, msg InMessage) {
	approve := msg.Approved != nil && *msg.Approved
	resolved, approved, op, err := s.consensus.Vote(msg.ProposalID, clientID, approve)
	if err != nil {
		s.sendMsg(clientID, OutMessage{
			Type:    "op_error",
			ReqID:   msg.ReqID,
			Code:    "invalid",
			Message: err.Error(),
		})
		return
	}

	if !resolved {
		return
	}

	if approved && op != nil {
		if op.ID == "" {
			op.ID = msg.ProposalID
		}
		if len(op.Payload) == 0 {
			op.Payload = json.RawMessage("{}")
		}
		currState := s.service.Get()
		op.ExpectedRevision = &currState.Revision

		committedState, commitErr := s.service.Commit(*op)
		if commitErr != nil {
			log.Printf("[Room %s] Consensus commit failed for proposal %s (%s): %v", s.ID, msg.ProposalID, op.Type, commitErr)
			approved = false
			s.Broadcast(OutMessage{
				Type:       "consensus_resolved",
				ProposalID: msg.ProposalID,
				Approved:   &approved,
				Message:    commitErr.Error(),
			})
			return
		}

		// Release ownership of any cascaded deleted clips
		prevClipMap := make(map[string]bool)
		for _, clip := range currState.Project.Clips {
			prevClipMap[clip.ID] = true
		}
		for _, clip := range committedState.Project.Clips {
			delete(prevClipMap, clip.ID)
		}
		if len(prevClipMap) > 0 {
			deletedClipIDs := make([]string, 0, len(prevClipMap))
			for deletedID := range prevClipMap {
				deletedClipIDs = append(deletedClipIDs, deletedID)
				if owner, ok := s.ownership.GetOwner(deletedID); ok {
					s.ownership.Release(owner, []string{deletedID})
				}
			}
			s.Broadcast(OutMessage{
				Type:    "release_broadcast",
				ClipIDs: deletedClipIDs,
			})
		}

		s.Broadcast(OutMessage{
			Type:       "consensus_resolved",
			ProposalID: msg.ProposalID,
			Approved:   &approved,
			Op:         op,
		})

		s.Broadcast(OutMessage{
			Type:   "op_broadcast",
			Seq:    committedState.Revision,
			Op:     op,
			Author: clientID,
			State:  &committedState,
		})
	} else {
		s.Broadcast(OutMessage{
			Type:       "consensus_resolved",
			ProposalID: msg.ProposalID,
			Approved:   &approved,
		})
	}
}

func (s *RoomSession) handleSyncReq(clientID string, msg InMessage) {
	since := uint64(0)
	if msg.LastSeenSeq != nil {
		since = *msg.LastSeenSeq
	}
	ops, _ := s.service.GetOpsSince(since)
	currState := s.service.Get()
	s.sendMsg(clientID, OutMessage{
		Type:       "sync_resp",
		CurrentSeq: currState.Revision,
		Ops:        ops,
		State:      &currState,
	})
}

func (s *RoomSession) sendMsg(clientID string, msg OutMessage) {
	if s.emitter != nil {
		s.emitter.SendToClient(clientID, msg)
	}
}
