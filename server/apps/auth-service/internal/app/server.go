package app

import (
	"context"
	"log"

	"github.com/jsndz/kairo/apps/auth-service/internal/app/handler"
	authpb "github.com/jsndz/kairo/gen/go/proto/auth"
)

type AuthServer struct {
	h *handler.UserHandler
	authpb.UnimplementedAuthServiceServer
}


func NewAuthServer(h *handler.UserHandler) *AuthServer{
	return &AuthServer{
		h:h,
	}
}


func (s *AuthServer) SignUp(ctx context.Context, req *authpb.SignUpRequest) (*authpb.SignUpResponse, error) {
	resp,err := s.h.SignUp(ctx, req)
	if err!=nil{
		log.Println(err)
	}
	return resp, nil
}

func (s *AuthServer) SignIn(ctx context.Context, req *authpb.SignInRequest) (*authpb.SignInResponse, error) {
	resp,err := s.h.SignIn(ctx, req)
	if err!=nil{
		log.Println(err)
	}
    return resp, nil
}

func (s *AuthServer) Validate(ctx context.Context, req *authpb.ValidateRequest) (*authpb.ValidateResponse, error) {
	 res := s.h.Validate(ctx,req)
    return res, nil
}


func (s *AuthServer) CreateWSToken(ctx context.Context, req *authpb.CreateWSTokenRequest) (*authpb.CreateWSTokenResponse, error) {
	res ,err:= s.h.CreateWSToken(ctx,req)
	if err!=nil{
		log.Println(err)
	}
   return res, nil
}

func (s *AuthServer) AuthenticateWS(ctx context.Context, req *authpb.AuthenticateWSRequest) (*authpb.AuthenticateWSResponse, error) {
	res := s.h.AuthenticateWS(ctx,req)
   return res, nil
}