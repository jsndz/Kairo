package ws

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"server/internal/domain/asset"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

// Hub is the global manager for all active collaboration rooms.
type Hub struct {
	Rooms    map[string]*Room
	mu       sync.RWMutex
	service  ports.ProjectService
	upgrader websocket.Upgrader
}

// NewHub constructs a new Hub with the provided project service.
func NewHub(svc ports.ProjectService) *Hub {
	return &Hub{
		Rooms:   make(map[string]*Room),
		service: svc,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// Service returns the underlying project service.
func (h *Hub) Service() ports.ProjectService {
	return h.service
}

// ServeHTTP satisfies the http.Handler interface so Hub can be directly mounted.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.ServeWs(w, r)
}

// ServeWs upgrades incoming HTTP requests to WebSocket connections and joins the project room.
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("projectId")
	if projectID == "" {
		projectID = h.service.ProjectID()
	}

	clientID := r.URL.Query().Get("clientId")
	if clientID == "" {
		clientID = r.URL.Query().Get("client_id")
	}

	room := h.GetOrCreateRoom(projectID)
	if !room.CanJoin(clientID) {
		http.Error(w, "room is full: maximum 2 users allowed", http.StatusServiceUnavailable)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	if _, err := room.JoinWithID(conn, clientID); err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, err.Error()))
		_ = conn.Close()
	}
}

// GetOrCreateRoom retrieves an existing room or creates a new one for the project.
func (h *Hub) GetOrCreateRoom(projectID string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.Rooms[projectID]
	if !exists {
		room = NewRoom(projectID, h.service, h)
		h.Rooms[projectID] = room
	}
	return room
}

// RemoveRoom cleans up an empty room from memory.
func (h *Hub) RemoveRoom(projectID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.Rooms, projectID)
}

// BroadcastAsset forwards an asset change to all active rooms.
func (h *Hub) BroadcastAsset(a asset.Asset) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, room := range h.Rooms {
		room.BroadcastAsset(a)
	}
}

// BroadcastAssetDeleted forwards an asset deletion to all active rooms.
func (h *Hub) BroadcastAssetDeleted(id string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, room := range h.Rooms {
		room.BroadcastAssetDeleted(id)
	}
}

// BroadcastOperation forwards an operation and updated state to all connected clients in the room.
func (h *Hub) BroadcastOperation(projectID string, edit timeline.Edit, state ports.State, author string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, room := range h.Rooms {
		room.BroadcastOperation(projectID, edit, state, author)
	}
}
