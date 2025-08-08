package handler

import (
	"context"
	"strings"

	authpb "github.com/jsndz/kairo-proto/proto/auth"
	"github.com/jsndz/kairo/auth-service/internal/app/model"
	"github.com/jsndz/kairo/auth-service/internal/app/service"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

type UserHandler struct{
	userService *service.UserService
}

func NewUserHandler(db *gorm.DB) * UserHandler {
	return &UserHandler{
		userService: service.NewUserService(db),
	}
}


func (h *UserHandler) SignUp(ctx context.Context,req *authpb.SignUpRequest) (authpb.SignUpResponse,error){
	user:= model.User{
		Name: req.Name,
		Password: req.Password,
		Email: req.Email,
	}
	token, data, err := h.userService.Signup(user)
	if err != nil {
		return authpb.SignUpResponse{}, err
	}

	authUser := &authpb.User{
		Name:  data.Name,
		Email: data.Email,
	}

	return authpb.SignUpResponse{
		Token: token,
		User:  authUser,
	},nil
}

func (h *UserHandler) SignIn(ctx context.Context,req *authpb.SignInRequest)(authpb.SignInResponse,error){
	

	token,data,err := h.userService.Signin(req.Email,req.Password)
	if err != nil {
		return authpb.SignInResponse{}, err
	}

	authUser := &authpb.User{
		Name:  data.Name,
		Email: data.Email,
	}
	return authpb.SignInResponse{
		Token: token,
		User: authUser,
	},err
}


func (h *UserHandler) Validate(ctx context.Context,req *authpb.ValidateRequest) (authpb.ValidateResponse){
	md,ok:= metadata.FromIncomingContext(ctx)
	if !ok{
		return authpb.ValidateResponse{
			Valid: false,
			UserId: "",
		} 
	}
	authHeader := md["authorization"]
	if len(authHeader) == 0 {
		return authpb.ValidateResponse{
			Valid: false,
			UserId: "",
		}
	}
	token := strings.TrimPrefix(authHeader[0], "Bearer ")
	user_id,err := h.userService.Authenticate(token)

	if err != nil {
		return authpb.ValidateResponse{
			Valid: false,
			UserId: "",
		}
	}
	return  authpb.ValidateResponse{
		Valid: true,
		UserId: user_id,
	}

}