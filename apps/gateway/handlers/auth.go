package handlers

import (
	"github.com/gin-gonic/gin"
	authpb "github.com/jsndz/kairo-proto/proto/auth"
)


type AuthHandlers struct{
	AuthClient authpb.AuthServiceClient
}


func(h *AuthHandlers) SignIn(ctx *gin.Context)  {
	var req authpb.SignInRequest
	if err := ctx.ShouldBindBodyWithJSON(&req); err!=nil{
		ctx.JSON(400,gin.H{"error":"Unable parse json"})
		return 
	}

	res,err:= h.AuthClient.SignIn(ctx,&req)

	if err!= nil{
		ctx.JSON(400,gin.H{"error":"Unable to get data"})
		return
	}
	ctx.JSON(200,res)
}

func(h *AuthHandlers) SignUp(ctx *gin.Context)  {
	var req authpb.SignUpRequest
	if err := ctx.ShouldBindBodyWithJSON(&req); err!=nil{
		ctx.JSON(400,gin.H{"error":"Unable parse json"})
		return 
	}

	res,err:= h.AuthClient.SignUp(ctx,&req)

	if err!= nil{
		ctx.JSON(400,gin.H{"error":"Unable to get data"})
		return
	}
	ctx.JSON(200,res)
}