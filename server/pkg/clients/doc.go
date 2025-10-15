package clients

import (
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
	"google.golang.org/grpc"
)

func NewDocClient() (docpb.DocServiceClient,*grpc.ClientConn) {
	conn:= CreateClient("localhost:3002")
	docClient := docpb.NewDocServiceClient(conn)
	return docClient,conn
}

