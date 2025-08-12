package clients

import (
	authpb "github.com/jsndz/kairo/gen/go/proto/auth"
	"google.golang.org/grpc"
)

func NewAuthClient() (authpb.AuthServiceClient,*grpc.ClientConn) {
	conn:= dial("localhost:3001")
	authClient := authpb.NewAuthServiceClient(conn)
	return authClient,conn
}

