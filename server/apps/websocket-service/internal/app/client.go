package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jsndz/kairo/apps/websocket-service/middleware"
	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
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
		if err := c.Conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
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
		
	case 2:
		c.handleJoin(payload, h)
	case 3:
		c.handleSummarize()
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
		c.HandleNotification(2,"error in extracting token ")
		return
	}

	userId, err := middleware.Authenticate(payload.Token)
	if err != nil {
		c.HandleNotification(2,"Authentication failed") 
		return
	}

	parsedUserId, err := strconv.ParseUint(userId, 10, 32)
	if err != nil {
		c.HandleNotification(2,"Invalid user ID") 
		return
	}

	c.UserId = uint32(parsedUserId)
	room := h.GetOrCreateRoom(payload.DocID)
	room.AddClient(c)
	c.Room = room

	c.HandleNotification(2,"Joined Room successfully") 
	room.Broadcast(c.UserId, mustMarshal(fmt.Sprintf("%d joined the Room.", c.UserId)),2)
}

func (c *Client)HandleNotification(msgType uint,message string){
	msgBytes := []byte(message)
	payload := make([]byte, 1+len(msgBytes))
	payload[0]= byte(msgType)
	copy(payload[1:], msgBytes)
	c.Send<- payload
}

func (c *Client) handleUpdate(message []uint8 ) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if c.Room == nil || c.Room.Doc == nil {
        log.Printf("handleUpdate called but Room or Doc is nil (user %d)", c.UserId)
        return
    }
	log.Print(c.Room.Doc)
	_, err := c.Room.Doc.CreateDelta(ctx, &docpb.CreateDeltaRequest{DocId: c.Room.DocId, Delta: message})
	if err != nil {
		log.Printf("failed to save delta for doc %d: %v", c.Room.DocId, err)
	}
	c.Room.Broadcast(c.UserId, message,0)
}


func (c *Client) handleSummarize(){
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stream,err := c.Room.AI.Summarize(ctx,&aipb.SummarizeRequest{DocId:c.Room.DocId })
	if err != nil {
		log.Printf("failed to summarize %d: %v", c.Room.DocId, err)
	}
	for {
		data ,err  := stream.Recv()
		if err == io.EOF {
			log.Println("Summarization complete")
			break
		}
		if err != nil {
			log.Printf("Stream receive error: %v", err)
			break
		}
		c.Room.Broadcast(c.UserId, []byte(data.Summary), 0)

		if data.Done {
			log.Println("Summarization done signal received")
			break
		}
	}
	
}

func mustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Println("marshal error:", err)
		return nil
	}
	return b
}
