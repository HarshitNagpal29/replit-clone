package main

import (
	"log"
	"net/http"
	"os"

	"github.com/HarshitNagpal29/replit-clone/backend/src/http"
	"github.com/HarshitNagpal29/replit-clone/backend/src/ws"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	// Create a Gin router
	router := gin.Default()

	// Enable CORS
	router.Use(corsMiddleware())

	// Initialize HTTP routes
	http.InitHttp(router) // Correctly call InitHttp from the http package

	// Create the HTTP server
	httpServer := &http.Server{
		Addr:    ":" + getPort(),
		Handler: router,
	}

	// Initialize WebSocket server
	ws.InitWs(httpServer) // Correctly call InitWs from the ws package

	// Start the server
	log.Printf("listening on *:%s\n", getPort())
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// Middleware to enable CORS
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, X-Auth-Token")
		c.Next()
	}
}

// Retrieve port from environment variables or default to 3001
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	return port
}
