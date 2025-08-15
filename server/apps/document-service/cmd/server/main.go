package main

import (
	"log"
	"net"
	"os"

	"github.com/jsndz/kairo/apps/document-service/internal/app"
	"github.com/jsndz/kairo/apps/document-service/internal/app/handler"
	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
	"github.com/jsndz/kairo/pkg/db"
	"github.com/jsndz/kairo/pkg/env"
	"google.golang.org/grpc"
)



func main(){
	env.Loadenv()

	dsn := os.Getenv("DOC_DB_URL")
    database,err := db.InitDB(dsn)
	db.MigrateDB(database,model.Document{},model.DocumentUpdate{})
	lis,err:= net.Listen("tcp",":3002")
	

	if err!= nil{
		log.Fatalf("Failed to listen: %v", err)
	}
    h:= handler.NewDocHandler(database)
	docServer := app.NewDocServer(h)
	grpcServer:= grpc.NewServer()

	docpb.RegisterDocServiceServer(grpcServer,docServer)

	log.Println("Doc gRPC server running on :3002")
	if err= grpcServer.Serve(lis);err!=nil{
        log.Fatalf("Failed to serve: %v", err)
	}

}