package ws

import (
	"crypto/rand"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

// Client represents a single active WebSocket connection.
type Client struct {
	conn      *websocket.Conn
	ID        string
	Room      *Room
	send      chan []byte
	lastSeq   uint64
	closeOnce sync.Once
	mu        sync.RWMutex
	isClosed  bool
}

// NewClient creates a new Client with a random ID.
func NewClient(conn *websocket.Conn, room *Room) *Client {
	return NewClientWithID(conn, room, rand.Text())
}

// NewClientWithID creates a new Client with a specific ID.
func NewClientWithID(conn *websocket.Conn, room *Room, id string) *Client {
	if id == "" {
		id = rand.Text()
	}
	return &Client{
		conn: conn,
		ID:   id,
		Room: room,
		send: make(chan []byte, 256),
	}
}

// Start launches the read and write pump goroutines.
func (c *Client) Start() {
	go c.writepump()
	go c.readpump()
}

func (c *Client) readpump() {
	defer c.Close()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg InMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		c.Room.HandleMessage(c, msg)
	}
}

func (c *Client) writepump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send attempts to enqueue a message to be written to the WebSocket.
func (c *Client) Send(msg []byte) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.isClosed {
		return false
	}
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

// Close gracefully tears down the client connection and notifies its room.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		if c.Room != nil {
			c.Room.Leave(c)
		}
		c.mu.Lock()
		c.isClosed = true
		close(c.send)
		c.mu.Unlock()
		_ = c.conn.Close()
	})
}
