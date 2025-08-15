package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jsndz/kairo/apps/gateway/handlers"
	"github.com/jsndz/kairo/apps/gateway/route"
	"github.com/jsndz/kairo/pkg/clients"
)

func main(){
	router := gin.New()
	router.Use(gin.Recovery())

	authClient,authconn:= clients.NewAuthClient()
	defer authconn.Close()
	authHandlers := handlers.AuthHandlers{
		AuthClient: authClient,
	}

	docClient,docconn:= clients.NewDocClient()
	defer docconn.Close()
	docHandlers := handlers.DocHandlers{
		DocClient: docClient,
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
	
	doc := api.Group("/doc")
	doc.Use(authHandlers.ValidationMiddleware())
	route.DocRoute(doc,docHandlers)
	

	
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}