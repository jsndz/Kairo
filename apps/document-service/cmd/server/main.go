package main

import (
	"log"
	"net"
	"os"

	docpb "github.com/jsndz/kairo-proto/proto/doc"
	"github.com/jsndz/kairo/document-service/internal/app/handler"
	"github.com/jsndz/kairo/pkg/db"
	"google.golang.org/grpc"
)


type DocServer struct{
	h *handler.DocHandler
	docpb.UnimplementedDocServiceServer
}


func main(){
	dsn := os.Getenv("DOC_DB_URL")
    database,err := db.InitDB(dsn)

	lis,err:= net.Listen("tcp",":3002")

	if err!= nil{
		log.Fatalf("Failed to listen: %v", err)
	}
    h:= handler.NewDocHandler(database)
	docServer := &DocServer{h:h}
	grpcServer:= grpc.NewServer()

	docpb.RegisterDocServiceServer(grpcServer,docServer)

	log.Println("Auth gRPC server running on :3002")
	if err= grpcServer.Serve(lis);err!=nil{
        log.Fatalf("Failed to serve: %v", err)
	}

}