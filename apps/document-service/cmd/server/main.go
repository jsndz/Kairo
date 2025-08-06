package main

import (
	"log"
	"net"

	docpb "github.com/jsndz/kairo-proto/proto/doc"
	"google.golang.org/grpc"
)


type DocServer struct{
	docpb.UnimplementedDocServiceServer
}


func main(){
	lis,err:= net.Listen("tcp",":3002")

	if err!= nil{
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer:= grpc.NewServer()

	docpb.RegisterDocServiceServer(grpcServer,DocServer{})

	log.Println("Auth gRPC server running on :3002")
	if err= grpcServer.Serve(lis);err!=nil{
        log.Fatalf("Failed to serve: %v", err)
	}

}