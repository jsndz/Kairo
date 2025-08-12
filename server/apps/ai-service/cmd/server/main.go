package main

import (
	"log"
	"net"

	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
	"google.golang.org/grpc"
)


type AIServer struct{
	aipb.UnimplementedAIServiceServer
}


func main(){
	lis,err :=net.Listen("tcp",":3003")
	if err!=nil{
		log.Fatalf("Failed to listen: %v", err)

	}

	grpcServer:= grpc.NewServer()

	aipb.RegisterAIServiceServer(grpcServer,AIServer{})
	log.Println("Auth gRPC server running on :3003")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}