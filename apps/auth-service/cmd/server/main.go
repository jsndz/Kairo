package main

import (
	"log"
	"net"
	"os"

	authpb "github.com/jsndz/kairo-proto/proto/auth"
	"github.com/jsndz/kairo/auth-service/internal/app/handler"
	"github.com/jsndz/kairo/pkg/db"
	"github.com/jsndz/kairo/pkg/env"
	"google.golang.org/grpc"
)


type AuthServer struct {
	h *handler.UserHandler
	authpb.UnimplementedAuthServiceServer
}

func main(){  
    env.Loadenv()
    dsn := os.Getenv("AUTH_DB_URL")
    database,err := db.InitDB(dsn)
	lis, err := net.Listen("tcp", ":3001")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    h:= handler.NewUserHandler(database)
    authServer := &AuthServer{h: h}
    grpcServer := grpc.NewServer()

    authpb.RegisterAuthServiceServer(grpcServer,authServer)

    log.Println("Auth gRPC server running on :3001")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }

}