package ws

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"server/internal/consensus"
	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	"server/internal/domain/timeline"
	"server/internal/ownership"
	"server/internal/service"
)

type pendingPermission struct {
	reqID        string
	fromClientID string
	ownerID      string
	clipID       string
	timer        *time.Timer
}

// Room manages real-time collaboration for a single project (max 2 users).
type Room struct {
	ID                 string
	clients            map[string]*Client
	service            *service.Service
	hub                *Hub
	ownership          *ownership.Engine
	consensus          *consensus.Coordinator
	reclaimTimers      map[string]*time.Timer
	pendingPermissions map[string]*pendingPermission
	graceDuration      time.Duration
	mu                 sync.RWMutex
}

// NewRoom creates a new Room for the given project.
func NewRoom(id string, ser *service.Service, h *Hub) *Room {
	r := &Room{
		ID:                 id,
		clients:            make(map[string]*Client),
		service:            ser,
		hub:                h,
		ownership:          ownership.New(),
		consensus:          consensus.New(),
		reclaimTimers:      make(map[string]*time.Timer),
		pendingPermissions: make(map[string]*pendingPermission),
		graceDuration:      15 * time.Second,
	}

	r.consensus.SetTimeoutCallback(func(p *consensus.Proposal) {
		approved := false
		r.Broadcast(OutMessage{
			Type:       "consensus_resolved",
			ProposalID: p.ID,
			Approved:   &approved,
			Message:    "Proposal timed out after 30 seconds",
		})
	})

	return r
}

// SetGraceDuration overrides the disconnect grace period (useful in tests).
func (r *Room) SetGraceDuration(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.graceDuration = d
}

// CanJoin checks if the room has capacity for a new or reconnecting client.
func (r *Room) CanJoin(clientID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if clientID != "" {
		if _, isConnected := r.clients[clientID]; isConnected {
			return false
		}
		if _, hasGrace := r.reclaimTimers[clientID]; hasGrace {
			return true
		}
	}
	return len(r.clients) < 2
}

// Join adds a client connection with a random ID.
func (r *Room) Join(conn *websocket.Conn) (*Client, error) {
	return r.JoinWithID(conn, "")
}

// JoinWithID adds a client connection with a specific ID or recovers session during grace period.
func (r *Room) JoinWithID(conn *websocket.Conn, clientID string) (*Client, error) {
	r.mu.Lock()

	// If reconnecting during grace period, cancel reclaim timer
	if clientID != "" {
		if t, ok := r.reclaimTimers[clientID]; ok {
			t.Stop()
			delete(r.reclaimTimers, clientID)
		}
	}

	if len(r.clients) >= 2 {
		r.mu.Unlock()
		return nil, errors.New("room is full: maximum 2 users allowed")
	}

	c := NewClientWithID(conn, r, clientID)
	r.clients[c.ID] = c
	r.mu.Unlock()

	c.Start()

	// Send initial greeting including current active ownership map
	currentState := r.service.Get()
	c.lastSeq = currentState.Revision
	c.sendMsg(OutMessage{
		Type:       "init",
		ClientID:   c.ID,
		CurrentSeq: currentState.Revision,
		State:      &currentState,
		Owners:     r.ownership.AllOwners(),
	})

	// Tell connecting client about existing peer
	r.mu.RLock()
	for id := range r.clients {
		if id != c.ID {
			c.sendMsg(OutMessage{
				Type:     "peer_joined",
				ClientID: id,
			})
		}
	}
	r.mu.RUnlock()

	// Notify peer that client joined
	r.BroadcastExcept(c.ID, OutMessage{
		Type:     "peer_joined",
		ClientID: c.ID,
	})

	return c, nil
}

// Leave removes a client from the room and starts grace timer for lock reclamation.
func (r *Room) Leave(c *Client) {
	r.mu.Lock()
	delete(r.clients, c.ID)
	remaining := len(r.clients)

	// Clean up pending permission requests involving departed client
	for reqID, req := range r.pendingPermissions {
		if req.fromClientID == c.ID || req.ownerID == c.ID {
			if req.timer != nil {
				req.timer.Stop()
			}
			delete(r.pendingPermissions, reqID)
		}
	}

	// Start disconnect grace timer for locks
	if _, exists := r.reclaimTimers[c.ID]; !exists {
		dur := r.graceDuration
		timer := time.AfterFunc(dur, func() {
			r.handleGraceExpiration(c.ID)
		})
		r.reclaimTimers[c.ID] = timer
	}
	r.mu.Unlock()

	if remaining > 0 {
		r.Broadcast(OutMessage{
			Type:     "peer_left",
			ClientID: c.ID,
		})
	}
}

func (r *Room) handleGraceExpiration(clientID string) {
	r.mu.Lock()
	delete(r.reclaimTimers, clientID)
	remainingClients := len(r.clients)
	remainingTimers := len(r.reclaimTimers)
	r.mu.Unlock()

	released := r.ownership.ReleaseClient(clientID)
	if len(released) > 0 {
		r.Broadcast(OutMessage{
			Type:    "release_broadcast",
			ClipIDs: released,
		})
	}

	if remainingClients == 0 && remainingTimers == 0 && r.hub != nil {
		r.hub.RemoveRoom(r.ID)
	}
}

// HandleMessage routes incoming WebSocket frames from a client.
func (r *Room) HandleMessage(c *Client, msg InMessage) {
	switch msg.Type {
	case "op_submit":
		r.handleOpSubmit(c, msg)

	case "claim_request":
		r.handleClaimRequest(c, msg)

	case "release_request":
		r.handleReleaseRequest(c, msg)

	case "permission_request":
		r.handlePermissionRequest(c, msg)

	case "permission_response":
		r.handlePermissionResponse(c, msg)

	case "consensus_propose":
		r.handleConsensusPropose(c, msg)

	case "consensus_vote":
		r.handleConsensusVote(c, msg)

	case "presence":
		r.BroadcastExcept(c.ID, OutMessage{
			Type:     "presence_broadcast",
			ClientID: c.ID,
			Cursor:   msg.Cursor,
			Playhead: msg.Playhead,
			Selected: msg.Selected,
		})

	case "sync_req":
		since := uint64(0)
		if msg.LastSeenSeq != nil {
			since = *msg.LastSeenSeq
		}
		ops, _ := r.service.GetOpsSince(since)
		currState := r.service.Get()
		c.sendMsg(OutMessage{
			Type:       "sync_resp",
			CurrentSeq: currState.Revision,
			Ops:        ops,
			State:      &currState,
		})
	}
}

func (r *Room) handleOpSubmit(c *Client, msg InMessage) {
	if msg.Op == nil {
		c.sendMsg(OutMessage{
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
	r.mu.RLock()
	hasPeer := len(r.clients) >= 2
	r.mu.RUnlock()
	if hasPeer && (msg.Op.Type == "add-track" || msg.Op.Type == "delete-track" || msg.Op.Type == "reset") {
		c.sendMsg(OutMessage{
			Type:    "op_error",
			ReqID:   msg.ReqID,
			Code:    "conflict",
			Message: "structural operations require consensus proposal",
		})
		return
	}

	// Authorize edit against clip ownership
	state, err := r.service.CommitAuthor(*msg.Op, c.ID, r.ownership)
	if err != nil {
		code := "invalid"
		if errors.Is(err, domErrors.ErrConflict) {
			code = "conflict"
		}
		c.sendMsg(OutMessage{
			Type:       "op_error",
			ReqID:      msg.ReqID,
			Code:       code,
			CurrentSeq: state.Revision,
			Message:    err.Error(),
		})
		return
	}

	c.lastSeq = state.Revision

	// Auto-claim newly added clips or release deleted clips
	if clipID, ok := timeline.ExtractClipID(*msg.Op); ok {
		if msg.Op.Type == "add" {
			r.ownership.Claim(c.ID, []string{clipID})
			r.Broadcast(OutMessage{
				Type:     "claim_broadcast",
				ClientID: c.ID,
				ClipIDs:  []string{clipID},
			})
		} else if msg.Op.Type == "delete" {
			r.ownership.Release(c.ID, []string{clipID})
			r.Broadcast(OutMessage{
				Type:    "release_broadcast",
				ClipIDs: []string{clipID},
			})
		}
	}

	// Ack author and broadcast to peer
	c.sendMsg(OutMessage{
		Type:   "op_ack",
		ReqID:  msg.ReqID,
		Seq:    state.Revision,
		Status: "ok",
		State:  &state,
	})

	r.BroadcastExcept(c.ID, OutMessage{
		Type:   "op_broadcast",
		Seq:    state.Revision,
		Op:     msg.Op,
		Author: c.ID,
		State:  &state,
	})
}

func (r *Room) handleClaimRequest(c *Client, msg InMessage) {
	granted, rejected := r.ownership.Claim(c.ID, msg.ClipIDs)

	c.sendMsg(OutMessage{
		Type:     "claim_ack",
		ReqID:    msg.ReqID,
		Granted:  granted,
		Rejected: rejected,
	})

	if len(granted) > 0 {
		r.Broadcast(OutMessage{
			Type:     "claim_broadcast",
			ClientID: c.ID,
			ClipIDs:  granted,
		})
	}
}

func (r *Room) handleReleaseRequest(c *Client, msg InMessage) {
	released := r.ownership.Release(c.ID, msg.ClipIDs)
	if len(released) > 0 {
		r.Broadcast(OutMessage{
			Type:    "release_broadcast",
			ClipIDs: released,
		})
	}
}

func (r *Room) handlePermissionRequest(c *Client, msg InMessage) {
	ownerID, exists := r.ownership.GetOwner(msg.ClipID)

	// If unassigned or already owned by requester, grant immediately
	if !exists || ownerID == c.ID {
		r.ownership.Claim(c.ID, []string{msg.ClipID})
		approved := true
		newOwner := c.ID
		r.Broadcast(OutMessage{
			Type:     "permission_resolved",
			ReqID:    msg.ReqID,
			ClipID:   msg.ClipID,
			Approved: &approved,
			NewOwner: &newOwner,
		})
		r.Broadcast(OutMessage{
			Type:     "claim_broadcast",
			ClientID: c.ID,
			ClipIDs:  []string{msg.ClipID},
		})
		return
	}

	// Clip is owned by peer
	r.mu.RLock()
	ownerClient, isConnected := r.clients[ownerID]
	r.mu.RUnlock()

	if !isConnected {
		// Peer is disconnected (in grace period)
		approved := false
		c.sendMsg(OutMessage{
			Type:     "permission_resolved",
			ReqID:    msg.ReqID,
			ClipID:   msg.ClipID,
			Approved: &approved,
		})
		return
	}

	// Register pending request with 30s timeout
	timer := time.AfterFunc(30*time.Second, func() {
		r.mu.Lock()
		req, ok := r.pendingPermissions[msg.ReqID]
		if ok {
			delete(r.pendingPermissions, msg.ReqID)
		}
		r.mu.Unlock()

		if ok {
			approved := false
			r.Broadcast(OutMessage{
				Type:     "permission_resolved",
				ReqID:    req.reqID,
				ClipID:   req.clipID,
				Approved: &approved,
			})
		}
	})

	r.mu.Lock()
	r.pendingPermissions[msg.ReqID] = &pendingPermission{
		reqID:        msg.ReqID,
		fromClientID: c.ID,
		ownerID:      ownerID,
		clipID:       msg.ClipID,
		timer:        timer,
	}
	r.mu.Unlock()

	// Send interactive prompt to owner
	ownerClient.sendMsg(OutMessage{
		Type:         "permission_prompt",
		ReqID:        msg.ReqID,
		FromClientID: c.ID,
		ClipID:       msg.ClipID,
		TimeoutMS:    30000,
	})
}

func (r *Room) handlePermissionResponse(c *Client, msg InMessage) {
	r.mu.Lock()
	req, exists := r.pendingPermissions[msg.ReqID]
	if exists {
		if req.timer != nil {
			req.timer.Stop()
		}
		delete(r.pendingPermissions, msg.ReqID)
	}
	r.mu.Unlock()

	if !exists {
		return
	}

	approved := msg.Approved != nil && *msg.Approved
	var newOwner *string
	if approved {
		_ = r.ownership.Transfer(req.clipID, req.fromClientID)
		newOwner = &req.fromClientID
	}

	r.Broadcast(OutMessage{
		Type:     "permission_resolved",
		ReqID:    req.reqID,
		ClipID:   req.clipID,
		Approved: &approved,
		NewOwner: newOwner,
	})

	if approved {
		r.Broadcast(OutMessage{
			Type:     "claim_broadcast",
			ClientID: req.fromClientID,
			ClipIDs:  []string{req.clipID},
		})
	}
}

func (r *Room) handleConsensusPropose(c *Client, msg InMessage) {
	if msg.Op == nil {
		c.sendMsg(OutMessage{
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

	p, err := r.consensus.Propose(msg.ProposalID, c.ID, *msg.Op, msg.Description, 30*time.Second)
	if err != nil {
		c.sendMsg(OutMessage{
			Type:    "op_error",
			ReqID:   msg.ReqID,
			Code:    "conflict",
			Message: err.Error(),
		})
		return
	}

	// Prompt peer to vote
	r.BroadcastExcept(c.ID, OutMessage{
		Type:        "consensus_prompt",
		ProposalID:  p.ID,
		ProposerID:  c.ID,
		Op:          &p.Op,
		Description: p.Description,
		TimeoutMS:   30000,
	})
}

func (r *Room) handleConsensusVote(c *Client, msg InMessage) {
	approve := msg.Approved != nil && *msg.Approved
	resolved, approved, op, err := r.consensus.Vote(msg.ProposalID, c.ID, approve)
	if err != nil {
		c.sendMsg(OutMessage{
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
		currState := r.service.Get()
		op.ExpectedRevision = &currState.Revision

		committedState, commitErr := r.service.Commit(*op)
		if commitErr != nil {
			log.Printf("[Room %s] Consensus commit failed for proposal %s (%s): %v", r.ID, msg.ProposalID, op.Type, commitErr)
			approved = false
			r.Broadcast(OutMessage{
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
				if owner, ok := r.ownership.GetOwner(deletedID); ok {
					r.ownership.Release(owner, []string{deletedID})
				}
			}
			r.Broadcast(OutMessage{
				Type:    "release_broadcast",
				ClipIDs: deletedClipIDs,
			})
		}

		r.Broadcast(OutMessage{
			Type:       "consensus_resolved",
			ProposalID: msg.ProposalID,
			Approved:   &approved,
			Op:         op,
		})

		r.Broadcast(OutMessage{
			Type:   "op_broadcast",
			Seq:    committedState.Revision,
			Op:     op,
			Author: c.ID,
			State:  &committedState,
		})
	} else {
		r.Broadcast(OutMessage{
			Type:       "consensus_resolved",
			ProposalID: msg.ProposalID,
			Approved:   &approved,
		})
	}
}

// Broadcast sends a message to all connected clients in the room.
func (r *Room) Broadcast(msg OutMessage) {
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, client := range r.clients {
		client.Send(raw)
	}
}

// BroadcastExcept sends a message to all clients except the specified client ID.
func (r *Room) BroadcastExcept(excludeID string, msg OutMessage) {
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, client := range r.clients {
		if id != excludeID {
			client.Send(raw)
		}
	}
}

// BroadcastAsset sends an asset update to all connected clients.
func (r *Room) BroadcastAsset(a asset.Asset) {
	r.Broadcast(OutMessage{
		Type:  "asset_updated",
		Asset: &a,
	})
}

// BroadcastAssetDeleted sends an asset deletion event to all connected clients.
func (r *Room) BroadcastAssetDeleted(assetID string) {
	r.Broadcast(OutMessage{
		Type:    "asset_deleted",
		AssetID: assetID,
	})
}

func (c *Client) sendMsg(msg OutMessage) {
	if raw, err := json.Marshal(msg); err == nil {
		c.Send(raw)
	}
}
