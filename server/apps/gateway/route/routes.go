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
	router.PUT("/update/:id",h.UpdateDoc)
	router.PUT("/update/name/:id",h.ChangeDocName)

	router.GET("/doc/:id",h.GetDoc)
	router.GET("/:id",h.GetUserDocs)

	router.GET("/save/:id",h.Save)
}


func AiRoute(router *gin.RouterGroup, h handlers.AIHandler) {
	router.GET("/summarize/:id",)
}