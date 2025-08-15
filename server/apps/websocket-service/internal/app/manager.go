package app

import "sync"


type Hub struct {
    Rooms map[uint32]*Room
    Mutex sync.Mutex
}

func (h *Hub) GetOrCreateRoom(roomId uint32) *Room{
    h.Mutex.Lock()
    defer h.Mutex.Unlock()
    if room, ok := h.Rooms[roomId]; ok {
        return room
    }
    room := &Room{
        DocId : roomId,
        clients: make(map[uint32]*Client),
    }
    h.Rooms[roomId]=room
	return room
}



func (h *Hub) DeleteRoom(roomId uint32) {
    h.Mutex.Lock()
    defer h.Mutex.Unlock()
    delete(h.Rooms,roomId) 
}
