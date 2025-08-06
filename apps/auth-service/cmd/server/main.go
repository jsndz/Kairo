package main

import (
	"log"
	"net"

	authpb "github.com/jsndz/kairo-proto/proto/auth"
	"google.golang.org/grpc"
)


type AuthServer struct{
	authpb.UnimplementedAuthServiceServer
}

func main(){
	lis, err := net.Listen("tcp", ":3001")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    grpcServer := grpc.NewServer()

    authpb.RegisterAuthServiceServer(grpcServer, &AuthServer{})

    log.Println("Auth gRPC server running on :3001")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }

}