Problem:
For getting the data of a specific doc we needed to receive Doc data through HTTP.

```go
type Document struct {
    ID            uint32   `db:"id"`
    Title         string    `db:"title"`
    UserID        uint32   `db:"user_id"`
    CurrentState  []byte    `db:"current_state"`
    CreatedAt     time.Time `db:"created_at"`
    UpdatedAt     time.Time `db:"updated_at"`
}

```

But the problem is the current state it is a []byte type. When sending through json it would be converted to hex automatically.

```ts
const binaryData = Uint8Array.from(atob(response.current_state), (c) =>
  c.charCodeAt(0)
);
```

The problem with this approach is that decoding hex will hamper performance of client (probably) and sending large data in JSON is hard.

Solution:
So i useda two request approach
Basically from frontend i changed the /api/doc/:id route into two kinds of request

1. header : {accept:"application/json"}
   Which is for meta data. That is everything except Current state

2. header : {accept:"application/octate-stream"}
   Which is for current_state

Code:
server/apps/gateway/handlers/doc.go
client/services/doc.ts
