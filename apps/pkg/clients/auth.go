package clients

import (
	"log"

	authpb "github.com/jsndz/kairo-proto/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewAuthClient() (authpb.AuthServiceClient,*grpc.ClientConn) {
	conn:= dial("localhost:3001")
	authClient := authpb.NewAuthServiceClient(conn)
	return authClient,conn
}

func dial(target string) *grpc.ClientConn {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", target, err)
	}
	return conn
}