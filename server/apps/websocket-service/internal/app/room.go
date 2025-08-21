package app

import (
	"sync"

	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
)

type Room struct {
	DocId   uint32
	clients map[uint32]*Client
	mutex   sync.Mutex
	Doc 	docpb.DocServiceClient

}

func (r *Room) AddClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.clients[client.UserId] = client
}

func (r *Room) RemoveClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.clients, client.UserId)
}

func (r *Room) Broadcast(userID uint32, msg []byte,msgType uint) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	payload := make([]byte, 1+len(msg))
	payload[0]= byte(msgType)
	copy(payload[1:], msg)
	for id, client := range r.clients {
		if id != userID {
			client.Send <- payload
		}
	}
}

func (r *Room) IsEmpty() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return len(r.clients) == 0
}
