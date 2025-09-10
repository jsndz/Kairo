Fancy name, it is simple method of sending data by message framing.

Problem:
For a collaboration platform spped is essentional, so sending large json might be heavy so we need a lighter method. But the ws server also needs to understand what type of data it needs to send. like for joining "join", "update" for sending deltas.

Solution:
A simple buffer where first byte is type, and everything else is payload.

Need to mention through websocket that you are sending bytes

```go

func (c *Client) WritePump() {
	for msg := range c.Send {
        //websocket.BinaryMessage indicates data is written as Binary
		if err := c.Conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
			log.Println("write error:", err)
			return
		}
	}
}


```

Consider this example :
({
type: "join",
payload: { token, doc_id },
})
sending this will be heavy instead this:
[2,21,1212,1,21,2,4,243,544,4,5,67,7]
here first byte 2 is byte that indicates join and rest is payload.
Code where it is used
server/apps/websocket-service/internal/app/client.go
client/components/Editor.tsx
client/lib/format.ts
