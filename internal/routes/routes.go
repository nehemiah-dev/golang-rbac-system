package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/auth"
)

func SetupRoutes(db *pgxpool.Pool, jwtUtil *auth.JWTUtil, app *gin.Engine) {
	authRepository := auth.NewRepository(db)
	authService := auth.NewService(authRepository, jwtUtil)
	authHandlers := auth.NewHandler(authService)
	app.POST("/auth/register", authHandlers.RegisterUser)
	app.POST("/auth/login", authHandlers.LoginUser)
	app.POST("/auth/refresh", authHandlers.RefreshTokens)
	app.POST("/auth/logout", authHandlers.Logout)
}
