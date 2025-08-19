package handlers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	authpb "github.com/jsndz/kairo/gen/go/proto/auth"
)

type AuthHandlers struct{
	AuthClient authpb.AuthServiceClient
}

func(h *AuthHandlers) SignIn(ctx *gin.Context)  {
	var req authpb.SignInRequest
	if err := ctx.ShouldBindBodyWithJSON(&req); err!=nil{
		ctx.JSON(400,gin.H{"error":"Unable parse json"})
		ctx.Abort()
		return 
	}

	res,err:= h.AuthClient.SignIn(ctx.Request.Context(),&req)
	ctx.SetCookie("kairo_token",(res.Token),86400,"/","",false,true)
	if err!= nil{
		ctx.JSON(400,gin.H{"error":"Unable to get data"})
		ctx.Abort()
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
	res,err:= h.AuthClient.SignUp(ctx.Request.Context(),&req)

	if err!= nil{
		ctx.JSON(400,gin.H{"error":"Unable to get data"})
		ctx.Abort()
		return
	}


	ctx.JSON(200,res)
}

func (h *AuthHandlers) ValidationMiddleware() gin.HandlerFunc {
    return func(ctx *gin.Context) {

        token, err := ctx.Cookie("kairo_token")
        if err != nil {
            ctx.JSON(401, gin.H{"error": "Token not found"})
            ctx.Abort()
            return
        }

        req := authpb.ValidateRequest{Token: token}

        res, err := h.AuthClient.Validate(ctx.Request.Context(), &req)
        if err != nil {
            ctx.JSON(401, gin.H{"error": "Invalid token"})
            ctx.Abort()
            return
        }

        if res.UserId == "" {
            ctx.JSON(401, gin.H{"error": "No user ID in validation response"})
            ctx.Abort()
            return
        }

        ctx.Set("user_id", res.UserId)

        ctx.Next()
    }
}

func ValidateToken(authClient authpb.AuthServiceClient, token string) (*authpb.ValidateResponse, error) {
    req := &authpb.ValidateRequest{Token: token}
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    return authClient.Validate(ctx, req)
}
