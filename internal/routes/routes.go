package routes

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/docs"
	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/auth"
)

func SetupRoutes(db *pgxpool.Pool, jwtUtil *auth.JWTUtil, app *gin.Engine) {
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080", "http://127.0.0.1:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	app.GET("/docs", func(c *gin.Context) {
		c.Redirect(301, "/docs/index.html")
	})
	app.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authRepository := auth.NewRepository(db)
	authService := auth.NewService(authRepository, jwtUtil)
	authHandlers := auth.NewHandler(authService)
	app.POST("/auth/register", authHandlers.RegisterUser)
	app.POST("/auth/login", authHandlers.LoginUser)
	app.POST("/auth/refresh", authHandlers.RefreshTokens)
	app.POST("/auth/logout", authHandlers.Logout)
}
