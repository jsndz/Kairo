package clients

import (
	docpb "github.com/jsndz/kairo-proto/proto/doc"
	"google.golang.org/grpc"
)

func NewDocClient() (docpb.DocServiceClient,*grpc.ClientConn) {
	conn:= dial("localhost:3001")
	docClient := docpb.NewDocServiceClient(conn)
	return docClient,conn
}

