package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jsndz/kairo/apps/websocket-service/internal/app"
	"github.com/jsndz/kairo/pkg/clients"
	"github.com/jsndz/kairo/pkg/env"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:3000" {
			return true
		}else {
			return false
		}
	},
}

func wsHandler(h *app.Hub) http.HandlerFunc{
	return func (w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		client := &app.Client{
			Conn: conn,
			Send: make(chan []byte,256),
		}
		go client.ReadPump(h)
		go client.WritePump()
	}
}


func main(){
	env.Loadenv()
	docClient,docconn:= clients.NewDocClient()
	defer docconn.Close()
	hub := app.Hub{
		Rooms: make(map[uint32]*app.Room),
		Doc: docClient,
	}

	
	
	
	http.HandleFunc("/ws",wsHandler(&hub))
	log.Println("Web Socket Server started on :3004")
    err := http.ListenAndServe(":3004", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}