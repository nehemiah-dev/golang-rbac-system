package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/auth"
	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/users"
)

func SetupRoutes(db *pgxpool.Pool, app *gin.Engine) {
	authRepository := users.NewRepository(db)
	authService := auth.NewService(authRepository)
	authHandlers := auth.NewHandler(authService)
	app.POST("/auth/register", authHandlers.RegisterUser)
	app.POST("/auth/login", authHandlers.LoginUser)
}
