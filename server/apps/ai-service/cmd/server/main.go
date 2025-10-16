package main

import (
	"log"
	"net"

	"github.com/jsndz/kairo/apps/ai-service/internal/app"
	"github.com/jsndz/kairo/apps/ai-service/internal/app/handler"
	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)





func main(){
	lis,err :=net.Listen("tcp",":3003")
	if err!=nil{
		log.Fatalf("Failed to listen: %v", err)
	}

	h := handler.NewAiHandler()
	aiserver := app.NewAiServer(h)
	grpcServer:= grpc.NewServer()

	aipb.RegisterAIServiceServer(grpcServer,aiserver)
	reflection.Register(grpcServer)
	log.Println("AI gRPC server running on :3003")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}