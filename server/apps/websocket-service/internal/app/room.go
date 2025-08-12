package app

import (
	"sync"
)



type Room struct{
	ID string
	clients map[string]*Client 
	mutex sync.Mutex
}

func (r *Room) AddClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.clients[client.ID]= client
}


func (r *Room) RemoveCLient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.clients,client.ID)
}


func (r *Room) Broadcast(userID string,msg []byte) {
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