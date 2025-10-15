Lets Start with basics of rpc

RPC is a way to call a function a in another server
There are 2 main components:
client and server

there is simple way to create client with grpc:

```go
package clients

import (
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
	"google.golang.org/grpc"
)
//create client
func CreateClient(target string) *grpc.ClientConn {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", target, err)
	}
	return conn
}

func NewDocClient() (docpb.DocServiceClient,*grpc.ClientConn) {
	conn:= CreateClient("localhost:3002")
	docClient := docpb.NewDocServiceClient(conn)
	return docClient,conn
}



```

This creates a doc client.

Now we have client and lets initialize it in the main file.

```go
aiClient,aiconn := clients.NewAIClient()
defer aiconn.Close()

```

This will create a ai client and pass this to the handler.

the AiClient Provides Various methods to call. In the defination i mentioned about functions right the functions are accessible through AiClient

```go
type AIHandler struct{
	AiClient aipb.AIServiceClient
}
```
