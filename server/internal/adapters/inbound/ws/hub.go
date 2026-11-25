package ws

import (
	"context"
	"fmt"
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
	manager  ports.ProjectManager
	upgrader websocket.Upgrader
}

// NewHub constructs a new Hub powered by a dynamic ProjectManager.
func NewHub(manager ports.ProjectManager) *Hub {
	return &Hub{
		Rooms:   make(map[string]*Room),
		manager: manager,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// ServeWs upgrades incoming HTTP requests to WebSocket connections and joins the project room.
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	if projectID == "" {
		projectID = r.URL.Query().Get("projectId")
	}
	if projectID == "" {
		http.Error(w, "missing projectId", http.StatusBadRequest)
		return
	}

	clientID := r.URL.Query().Get("clientId")

	room, err := h.GetOrCreateRoom(projectID)
	if err != nil {
		http.Error(w, "project not found: "+err.Error(), http.StatusNotFound)
		return
	}

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
func (h *Hub) GetOrCreateRoom(projectID string) (*Room, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}

	// 1. Fast-path: check if room already exists with read-lock
	h.mu.RLock()
	room, exists := h.Rooms[projectID]
	h.mu.RUnlock()
	if exists {
		return room, nil
	}

	// 2. Load the project instance outside of the Hub lock to prevent blocking other rooms during I/O
	if h.manager == nil {
		return nil, fmt.Errorf("project manager not configured")
	}
	inst, err := h.manager.Get(context.Background(), projectID)
	if err != nil {
		return nil, err
	}

	// 3. Acquire write-lock and register
	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check pattern in case another connection created it concurrently
	if existing, alreadyExists := h.Rooms[projectID]; alreadyExists {
		return existing, nil
	}

	room = NewRoom(projectID, inst, h)
	h.Rooms[projectID] = room
	return room, nil
}

// RemoveRoom cleans up an empty room from memory.
func (h *Hub) RemoveRoom(projectID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.Rooms, projectID)
}

// IsBusy returns true if the project currently has connected users.
func (h *Hub) IsBusy(projectID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, exists := h.Rooms[projectID]
	if !exists {
		return false
	}
	return room.ActiveUsersCount() > 0
}

// BroadcastProjectAsset forwards an asset change strictly to the specific project room.
func (h *Hub) BroadcastProjectAsset(projectID string, a asset.Asset) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if room, exists := h.Rooms[projectID]; exists {
		room.BroadcastAsset(a)
	}
}

// BroadcastProjectAssetDeleted forwards an asset deletion to a specific project room.
func (h *Hub) BroadcastProjectAssetDeleted(projectID, assetID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if room, exists := h.Rooms[projectID]; exists {
		room.BroadcastAssetDeleted(assetID)
	}
}

// BroadcastOperation forwards an operation and updated state strictly to connected clients in that project's room.
func (h *Hub) BroadcastOperation(projectID string, edit timeline.Edit, state ports.State, author string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if room, exists := h.Rooms[projectID]; exists {
		room.BroadcastOperation(projectID, edit, state, author)
	}
}
