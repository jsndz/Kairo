package handler

import (
	"context"

	"strconv"

	"github.com/jsndz/kairo/apps/auth-service/internal/app/model"
	"github.com/jsndz/kairo/apps/auth-service/internal/app/service"
	authpb "github.com/jsndz/kairo/gen/go/proto/auth"
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


func (h *UserHandler) SignUp(ctx context.Context,req *authpb.SignUpRequest) (*authpb.SignUpResponse,error){
	user:= model.User{
		Name: req.Name,
		Password: req.Password,
		Email: req.Email,
	}
	token, data, err := h.userService.Signup(user)
	if err != nil {
		return &authpb.SignUpResponse{}, err
	}

	authUser := &authpb.User{
		Name:  data.Name,
		Email: data.Email,
		Id: data.ID,
	}

	return &authpb.SignUpResponse{
		Token: token,
		User:  authUser,
	},nil
}

func (h *UserHandler) SignIn(ctx context.Context,req *authpb.SignInRequest)(*authpb.SignInResponse,error){
	

	token,data,err := h.userService.Signin(req.Email,req.Password)
	if err != nil {
		return &authpb.SignInResponse{}, err
	}

	authUser := &authpb.User{
		Name:  data.Name,
		Email: data.Email,
		Id: data.ID,
	}
	return &authpb.SignInResponse{
		Token: token,
		User: authUser,
	},err
}

func (h *UserHandler) CreateWSToken(ctx context.Context,req *authpb.CreateWSTokenRequest)(*authpb.CreateWSTokenResponse,error){
	token,err := h.userService.CreateWSToken(req.DocId,req.UserId)
	if err != nil {
		return &authpb.CreateWSTokenResponse{}, err
	}
	return &authpb.CreateWSTokenResponse{
		Token: token,
	},err
}

func (h *UserHandler) AuthenticateWS(ctx context.Context,req *authpb.AuthenticateWSRequest) (*authpb.AuthenticateWSResponse){

	user_id,doc_id,err := h.userService.AuthenticateWS(req.Token)

	if err != nil {
		return &authpb.AuthenticateWSResponse{
			Valid: false,
			UserId: 0,
			DocId: 0,
		}
	}
	return  &authpb.AuthenticateWSResponse{
		Valid: true,
		UserId: func() uint32 {
			id, _ := strconv.ParseUint(user_id, 10, 32)
			return uint32(id)
		}(),
		DocId: func() uint32 {
			id, _ := strconv.ParseUint(doc_id, 10, 32)
			return uint32(id)
		}(),
	}


}

func (h *UserHandler) Validate(ctx context.Context,req *authpb.ValidateRequest) (*authpb.ValidateResponse){
	
	
	user_id,err := h.userService.Authenticate(req.Token)

	if err != nil {
		return &authpb.ValidateResponse{
			Valid: false,
			UserId: "",
		}
	}
	return  &authpb.ValidateResponse{
		Valid: true,
		UserId: user_id,
	}

}