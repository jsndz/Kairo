package main

import (
	"context"
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
	"google.golang.org/grpc/status"
)

// Unary interceptor for logging
func loggingUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	log.Printf("[gRPC] Unary call - Method: %s, Req: %+v", info.FullMethod, req)
	resp, err = handler(ctx, req)
	if err != nil {
		log.Printf("[gRPC] Error in method %s: %v", info.FullMethod, err)
	} else {
		log.Printf("[gRPC] Response: %+v", resp)
	}
	return resp, err
}

// Stream interceptor for logging (for streaming calls)
func loggingStreamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	log.Printf("[gRPC] Stream call - Method: %s, IsClientStream: %v, IsServerStream: %v",
		info.FullMethod, info.IsClientStream, info.IsServerStream)
	err := handler(srv, ss)
	if err != nil {
		st, _ := status.FromError(err)
		log.Printf("[gRPC] Stream error in %s: %v", info.FullMethod, st.Message())
	}
	return err
}

func main() {
	env.Loadenv()

	dsn := os.Getenv("DOC_DB_URL")
	database, err := db.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to connect DB: %v", err)
	}
	db.MigrateDB(database, model.Document{}, model.DocumentUpdate{})

	lis, err := net.Listen("tcp", ":3002")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	h := handler.NewDocHandler(database)
	uh := handler.NewDocUpdateHandler(database)
	docServer := app.NewDocServer(h, uh)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingUnaryInterceptor),
		grpc.StreamInterceptor(loggingStreamInterceptor),
	)

	docpb.RegisterDocServiceServer(grpcServer, docServer)

	log.Println("Doc gRPC server running on :3002")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
