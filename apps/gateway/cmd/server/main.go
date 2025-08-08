package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jsndz/kairo/gateway/handlers"
	"github.com/jsndz/kairo/gateway/route"
	"github.com/jsndz/kairo/pkg/clients"
)

func main(){
	router := gin.New()
	router.Use(gin.Recovery())
	authClient,conn:= clients.NewAuthClient()
	defer conn.Close()
	authHandlers := handlers.AuthHandlers{
		AuthClient: authClient,
	}
	
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	api := router.Group("/api/v1")
	auth := api.Group("/auth")
	route.AuthRoute(auth,authHandlers)
	
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}