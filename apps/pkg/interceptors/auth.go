package interceptors

import (
	"context"
	"errors"
	"strings"

	authpb "github.com/jsndz/kairo-proto/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)


func AuthServerInterceptor(authClient authpb.AuthServiceClient) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md,ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil,errors.New("missing metadata")
		}
		tokens := md["authorization"]
		if len(tokens) == 0 {
			return nil,errors.New("No auth token")
		}
		token := strings.TrimPrefix(tokens[0], "Bearer ")
		user_id,err := authClient.Validate(ctx,&authpb.ValidateRequest{Token: token})
		ctx = context.WithValue(ctx,"user_id",user_id)
		return handler(ctx,req)
	} 
}