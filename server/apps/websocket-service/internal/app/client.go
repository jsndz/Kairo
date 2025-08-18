package app

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/jsndz/kairo/apps/websocket-service/middleware"
)

type Client struct {
	Conn   *websocket.Conn
	UserId uint32
	Send   chan []byte
	Room   *Room
}

type Message struct {
	Type   string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	DocId  uint32          `json:"roomId,omitempty"`
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

		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}

		c.HandleEvents(message, h)
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

func (c *Client) HandleEvents(message Message, h *Hub) {
	switch message.Type {
	case "join":
		c.handleJoin(message, h)
	case "update":
		c.handleUpdate(message)
	default:
		c.SendJSON("error", "Unknown message type", 0)
	}
}

func (c *Client) handleJoin(message Message, h *Hub) {
	var payload struct {
		Token string `json:"token"`
		DocID uint32 `json:"doc_id"`
	}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
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

	room.mutex.Lock()
	for _, u := range room.updates {
		c.Send <- u
	}
	room.mutex.Unlock()

	room.Broadcast(c.UserId, mustMarshal(fmt.Sprintf("%d joined the Room.", c.UserId)))
}

func (c *Client) handleUpdate(message Message) {
	c.Room.mutex.Lock()
	c.Room.updates = append(c.Room.updates, message.Payload)
	c.Room.mutex.Unlock()

	
	c.Room.Broadcast(c.UserId, message.Payload)
}

func (c *Client) SendJSON(msgType string, payload interface{}, roomId uint32) {
	msg, err := json.Marshal(Message{
		Type:    msgType,
		Payload: mustMarshal(payload),
		DocId:   roomId,
	})
	if err != nil {
		log.Println("failed to marshal message:", err)
		return
	}
	c.Send <- msg
}

func mustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Println("marshal error:", err)
		return nil
	}
	return b
}
