package main

import (
	"log"
	"net"
	"os"

	"github.com/jsndz/kairo/apps/auth-service/internal/app"
	"github.com/jsndz/kairo/apps/auth-service/internal/app/handler"
	"github.com/jsndz/kairo/apps/auth-service/internal/app/model"
	authpb "github.com/jsndz/kairo/gen/go/proto/auth"
	"github.com/jsndz/kairo/pkg/db"
	"github.com/jsndz/kairo/pkg/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main(){  
    env.Loadenv()
    dsn := os.Getenv("AUTH_DB_URL")
    database,err := db.InitDB(dsn)
    db.MigrateDB(database, &model.User{})

	lis, err := net.Listen("tcp", ":3001")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    h:= handler.NewUserHandler(database)
    authServer:= app.NewAuthServer(h)
    grpcServer := grpc.NewServer()

    authpb.RegisterAuthServiceServer(grpcServer,authServer)
    reflection.Register(grpcServer)
    log.Println("Auth gRPC server running on :3001")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }

}