// Command rbac-system is the entrypoint for the RBAC service.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/initializers"
	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/routes"
)

// @title           Golang RBAC System API
// @version         1.0.0
// @description     Authentication and Role-Based Access Control REST API service.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and your JWT access token.
func main() {
	router := gin.Default()
	router.GET("/health", healthHandler)

	if err := initializers.LoadConfig(); err != nil {
		log.Fatalln("Failed to load config:", err.Error())
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if !initializers.IsValidPort(port) {
		log.Fatalf("invalid PORT value")
	}
	addr := ":" + port

	ctx := context.Background()

	pool, err := initializers.ConnectToDB(ctx)
	if err != nil {
		log.Fatalln("Failed to connect to DB:", err.Error())
	}
	defer pool.Close()

	err = pool.Ping(ctx)
	if err != nil {
		log.Println("Database is unreachable or offline:", err.Error())
		pool.Close()
		return // Using return allows defers to execute cleanly instead of log.Fatalln
	}

	jwtUtil, err := initializers.InitJWT()
	if err != nil {
		log.Println("Failed to initialize JWT:", err.Error())
		return
	}

	routes.SetupRoutes(pool, jwtUtil, router)

	log.Println("Successfully connected to the database")
	// #nosec G706 -- addr is set strictly via system environment and validated
	log.Printf("server listening on: %s\n", addr)

	if err := router.Run(addr); err != nil {
		log.Printf("server failed: %v", err)
		return
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
