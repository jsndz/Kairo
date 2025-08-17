package app

import (
	"sync"
)



type Room struct{
	DocId uint32
	clients map[uint32]*Client 
	mutex sync.Mutex
	updates [][]byte
}

func (r *Room) AddClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.clients[client.UserId]= client
}


func (r *Room) RemoveClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.clients,client.UserId)
}


func (r *Room) Broadcast(userID uint32,msg []byte) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for id,client:= range r.clients{
		if id!= userID{
			client.Send <- msg
		}
	} 
}


func (r *Room) IsEmpty() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return len(r.clients) == 0
}