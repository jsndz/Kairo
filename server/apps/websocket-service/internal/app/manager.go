package app

import (
	"sync"

	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
)


type Hub struct {
    Rooms map[uint32]*Room
    Mutex sync.Mutex
    Doc docpb.DocServiceClient
    AI aipb.AIServiceClient
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
        Doc:     h.Doc,  
    }
    h.Rooms[roomId]=room
	return room
}



func (h *Hub) DeleteRoom(roomId uint32) {
    h.Mutex.Lock()
    defer h.Mutex.Unlock()
    delete(h.Rooms,roomId) 
}
