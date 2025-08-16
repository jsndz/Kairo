package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/websocket"
)


type Client struct{
	Conn *websocket.Conn
	UserId uint32
	Send chan []byte
	Room *Room
}


type Message struct {
    Type    string `json:"type"`
    Payload json.RawMessage `json:"payload"`
	DocId uint32 `json:"roomId,omitempty"`
}
func (c *Client) ReadPump(h *Hub){
	defer func(){
		if c.Room.clients != nil{
			c.Send <- []byte(fmt.Sprintf("%d disconnected", c.UserId))
			c.Room.RemoveClient(c)
			if c.Room.IsEmpty() {
				h.DeleteRoom(c.Room.DocId)
				log.Printf("Room %d deleted from hub", c.Room.DocId)
			}	
			close(c.Send)
		}
		c.Conn.Close()
	}()
	for{
		_,msg,err := c.Conn.ReadMessage()
		if err!=nil{
			c.Conn.Close()
			break
		}
		var message Message 
		err= json.Unmarshal([]byte(msg),&message)
		if err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}
		fmt.Printf("Message received: %v\n", message)
		HandleEvents(message, c, h)
	}
}

func (c *Client) WritePump(){
	for msg := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println(err)
			return
		}
	}
}


func HandleEvents( message Message, c *Client, h *Hub) {
	switch message.Type {
	case "join":{
		var joinPayload struct {
			Token string `json:"token"`
			DocID uint32 `json:"doc_id"`
		}
		if err := json.Unmarshal(message.Payload, &joinPayload); err != nil {
			return
		}  
		err := Authenticate(joinPayload.Token,c,h,joinPayload.DocID)
		log.Println(err)
		
	}
	case "update":
		c.Room.Broadcast(c.UserId,message.Payload)
	default:
		log.Println("Unknown message type:", message.Type)
	}
}



func Authenticate(token string,c *Client,h *Hub,DocID uint32) (error) {
	wsjwtSecret := os.Getenv("WS_JWT_SECRET")
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(wsjwtSecret), nil
	})
	log.Println(err)
	if err != nil || !parsedToken.Valid {
		return  fmt.Errorf("invalid token")
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims);
	if !ok {
   }
   var userID string

   switch id := claims["id"].(type) {
	case string:
		userID = id
	case float64:
		userID = fmt.Sprintf("%.0f", id) 
	default:
		return  fmt.Errorf("user ID is missing or invalid")
	}
	if err != nil {
		c.Conn.WriteMessage(websocket.TextMessage, []byte("Authentication failed"))
		
	}
	parsedUserId, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		c.Conn.WriteMessage(websocket.TextMessage, []byte("Invalid user ID"))
		
	}
	c.UserId = uint32(parsedUserId)
	room:= h.GetOrCreateRoom(DocID)
	room.AddClient(c)
	c.Room=room
	return nil
}