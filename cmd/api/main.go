// Command rbac-system is the entrypoint for the RBAC service.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/initializers"
	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/routes"
)

func main() {
	router := gin.Default()
	router.GET("/health", healthHandler)
	err := initializers.LoadConfig()
	if err != nil {
		log.Println("Failed to load config", err.Error())
		return
	}
	PORT := ":" + os.Getenv("PORT")
	addr := PORT
	ctx := context.Background()
	pool, err := initializers.ConnectToDB(ctx)
	if err != nil {
		log.Println("Failed to connect to DB", err.Error())
	}
	// DI here
	err = pool.Ping(ctx)
	if err != nil {
		log.Println("Database is unreachable or offline:", err.Error())
		pool.Close()
		return
	}

	routes.SetupRoutes(pool, router)

	log.Println("Successfully connected to the database")
	cleanAddr := strings.NewReplacer("\r", "", "\n", "").Replace(addr)
	log.Println("server listening on", cleanAddr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
