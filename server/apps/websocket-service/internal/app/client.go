package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jsndz/kairo/apps/websocket-service/middleware"
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
)

type Client struct {
	Conn   *websocket.Conn
	UserId uint32
	Send   chan []byte
	Room   *Room
	  
}


func (c *Client) ReadPump(h *Hub) {
	defer func() {
		if c.Room != nil {
			c.Send <- []byte(fmt.Sprintf("%d disconnected", c.UserId))
			if c.Room.IsEmpty() {
				h.DeleteRoom(c.Room.DocId)
				log.Printf("Room %d deleted from hub", c.Room.DocId)
			}
		}
		c.Conn.Close()
	}()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}
		c.HandleEvents(uint(msg[0]), []uint8(msg[1:]), h)
	}
}

func (c *Client) WritePump() {
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("write error:", err)
			return
		}
	}
}

func (c *Client) HandleEvents(msgType uint,payload []uint8 ,h *Hub) {
	switch msgType {
	case 0:
		c.handleUpdate(payload)
	case 1:
		//will be done later for awareness
	case 2:
		c.handleJoin(payload, h)
	
	default:
		c.Send <- []uint8{}
	}
}

func (c *Client) handleJoin(message []uint8, h *Hub) {
	var payload struct {
		Token string `json:"token"`
		DocID uint32 `json:"doc_id"`
	}
	if err := json.Unmarshal(message, &payload); err != nil {
		c.SendJSON("error", "Invalid join payload", 0)
		return
	}

	userId, err := middleware.Authenticate(payload.Token)
	if err != nil {
		c.SendJSON("error", "Authentication failed", 0)
		return
	}

	parsedUserId, err := strconv.ParseUint(userId, 10, 32)
	if err != nil {
		c.SendJSON("error", "Invalid user ID", 0)
		return
	}

	c.UserId = uint32(parsedUserId)
	room := h.GetOrCreateRoom(payload.DocID)
	room.AddClient(c)
	c.Room = room

	c.SendJSON("join", fmt.Sprintf("Joined room %d", payload.DocID), payload.DocID)
	room.Broadcast(c.UserId, mustMarshal(fmt.Sprintf("%d joined the Room.", c.UserId)))
}

func (c *Client)HandleNotification(msgType uint,message string){

}

func (c *Client) handleUpdate(message []uint8) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	log.Print(c.Room.Doc)
	_, err := c.Room.Doc.CreateDelta(ctx, &docpb.CreateDeltaRequest{DocId: c.Room.DocId, Delta: message})
	if err != nil {
		log.Printf("failed to save delta for doc %d: %v", c.Room.DocId, err)
	}
	c.Room.Broadcast(c.UserId, message)
}


func mustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Println("marshal error:", err)
		return nil
	}
	return b
}
