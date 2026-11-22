package ws

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/gorilla/websocket"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	"server/internal/service"
)

// Room manages real-time collaboration for a single project (max 2 users).
type Room struct {
	ID      string
	clients map[string]*Client
	service *service.Service
	hub     *Hub
	mu      sync.RWMutex
}

// NewRoom creates a new Room for the given project.
func NewRoom(id string, ser *service.Service, h *Hub) *Room {
	return &Room{
		ID:      id,
		clients: make(map[string]*Client),
		service: ser,
		hub:     h,
	}
}

// CanJoin checks if the room has capacity for a new client.
func (r *Room) CanJoin() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients) < 2
}

// Join adds a client connection to the room, rejecting if capacity is reached.
func (r *Room) Join(conn *websocket.Conn) (*Client, error) {
	r.mu.Lock()
	if len(r.clients) >= 2 {
		r.mu.Unlock()
		return nil, errors.New("room is full: maximum 2 users allowed")
	}

	c := NewClient(conn, r)
	r.clients[c.ID] = c
	r.mu.Unlock()

	c.Start()

	// Send initial greeting to the connecting client
	currentState := r.service.Get()
	c.lastSeq = currentState.Revision
	c.sendMsg(OutMessage{
		Type:       "init",
		ClientID:   c.ID,
		CurrentSeq: currentState.Revision,
		State:      &currentState,
	})

	// Tell the new client about any existing peer in the room
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

	// Notify existing peer that someone joined
	r.BroadcastExcept(c.ID, OutMessage{
		Type:     "peer_joined",
		ClientID: c.ID,
	})

	return c, nil
}

// Leave removes a client from the room and disposes the room if empty.
func (r *Room) Leave(c *Client) {
	r.mu.Lock()
	delete(r.clients, c.ID)
	remaining := len(r.clients)
	r.mu.Unlock()

	if remaining > 0 {
		r.Broadcast(OutMessage{
			Type:     "peer_left",
			ClientID: c.ID,
		})
	} else if r.hub != nil {
		r.hub.RemoveRoom(r.ID)
	}
}

// HandleMessage routes incoming WebSocket frames from a client.
func (r *Room) HandleMessage(c *Client, msg InMessage) {
	switch msg.Type {
	case "op_submit":
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

		state, err := r.service.Commit(*msg.Op)
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

		// Ack the author
		c.sendMsg(OutMessage{
			Type:   "op_ack",
			ReqID:  msg.ReqID,
			Seq:    state.Revision,
			Status: "ok",
			State:  &state,
		})

		// Broadcast committed edit and updated state to the peer
		r.BroadcastExcept(c.ID, OutMessage{
			Type:   "op_broadcast",
			Seq:    state.Revision,
			Op:     msg.Op,
			Author: c.ID,
			State:  &state,
		})

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
