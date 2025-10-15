package clients

import (
	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
	"google.golang.org/grpc"
)

func NewAIClient() (aipb.AIServiceClient,*grpc.ClientConn) {
	conn:= CreateClient("localhost:3003")
	aiClient := aipb.NewAIServiceClient(conn)
	return aiClient,conn
}

