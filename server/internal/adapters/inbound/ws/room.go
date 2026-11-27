package ws

import (
	"crypto/rand"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"server/internal/app/collaboration"
	"server/internal/domain/asset"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

// Room adapts Gorilla WebSocket connections to the transport-agnostic collaboration.RoomSession.
type Room struct {
	ID      string
	session *collaboration.RoomSession
	clients map[string]*Client
	hub     *Hub
	mu      sync.RWMutex
}

// NewRoom creates a new Room wrapping a collaboration.RoomSession.
func NewRoom(id string, ser ports.ProjectService, h *Hub) *Room {
	r := &Room{
		ID:      id,
		clients: make(map[string]*Client),
		hub:     h,
	}
	r.session = collaboration.NewRoomSession(id, ser, r)
	return r
}

// SetGraceDuration overrides the disconnect grace period.
func (r *Room) SetGraceDuration(d time.Duration) {
	r.session.SetGraceDuration(d)
}

// CanJoin checks whether client can join the room.
func (r *Room) CanJoin(clientID string) bool {
	return r.session.CanJoin(clientID)
}

// ActiveUsersCount returns the number of currently connected clients in the room.
func (r *Room) ActiveUsersCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Join adds a client connection with a random ID.
func (r *Room) Join(conn *websocket.Conn) (*Client, error) {
	return r.JoinWithID(conn, rand.Text())
}

// JoinWithID adds a client connection with a specific ID.
func (r *Room) JoinWithID(conn *websocket.Conn, clientID string) (*Client, error) {
	if clientID == "" {
		clientID = rand.Text()
	}

	c := NewClientWithID(conn, r, clientID)

	r.mu.Lock()
	r.clients[c.ID] = c
	r.mu.Unlock()

	c.Start()

	if err := r.session.Join(c.ID); err != nil {
		r.mu.Lock()
		delete(r.clients, c.ID)
		remaining := len(r.clients)
		r.mu.Unlock()
		c.Close()
		if remaining == 0 {
			r.RoomEmpty(r.ID)
		}
		return nil, err
	}

	return c, nil
}

// Leave notifies the collaboration session of a client departure.
func (r *Room) Leave(c *Client) {
	r.mu.Lock()
	delete(r.clients, c.ID)
	remaining := len(r.clients)
	r.mu.Unlock()

	r.session.Leave(c.ID)
	if remaining == 0 {
		r.RoomEmpty(r.ID)
	}
}

// HandleMessage routes incoming WebSocket messages to the collaboration session.
func (r *Room) HandleMessage(c *Client, msg collaboration.InMessage) {
	r.session.HandleMessage(c.ID, msg)
}

// Broadcast implements collaboration.SessionEmitter.
func (r *Room) Broadcast(msg collaboration.OutMessage) {
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

// BroadcastExcept implements collaboration.SessionEmitter.
func (r *Room) BroadcastExcept(excludeID string, msg collaboration.OutMessage) {
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

// SendToClient implements collaboration.SessionEmitter.
func (r *Room) SendToClient(clientID string, msg collaboration.OutMessage) {
	r.mu.RLock()
	client, ok := r.clients[clientID]
	r.mu.RUnlock()
	if ok {
		client.sendMsg(msg)
	}
}

// RoomEmpty implements collaboration.SessionEmitter.
func (r *Room) RoomEmpty(roomID string) {
	if r.hub != nil {
		r.hub.RemoveRoom(roomID)
	}
}

// BroadcastAsset forwards asset updates.
func (r *Room) BroadcastAsset(a asset.Asset) {
	r.session.BroadcastAsset(a)
}

// BroadcastAssetDeleted forwards asset deletion events.
func (r *Room) BroadcastAssetDeleted(assetID string) {
	r.session.BroadcastAssetDeleted(assetID)
}

// BroadcastOperation forwards operation commits.
func (r *Room) BroadcastOperation(projectID string, edit timeline.Edit, state ports.State, author string) {
	r.session.BroadcastOperation(projectID, edit, state, author)
}
