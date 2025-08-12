package route

import (
	"github.com/gin-gonic/gin"
	"github.com/jsndz/kairo/apps/gateway/handlers"
)

func AuthRoute(router *gin.RouterGroup, h handlers.AuthHandlers){
	router.POST("/signin",h.SignIn)
	router.POST("/signup",h.SignUp)
	router.GET("/")
}

func DocRoute(router *gin.RouterGroup, h handlers.DocHandlers){
	router.POST("/create",h.CreateDoc)
}