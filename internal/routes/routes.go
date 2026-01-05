package routes

import (
	"rumosaudavel-api/internal/handlers"
	"rumosaudavel-api/internal/middleware"
	"rumosaudavel-api/internal/repositories"
	"rumosaudavel-api/internal/services"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func RegisterRoutes(e *echo.Echo, db *gorm.DB) {
	userRepo := &repositories.UserRepository{DB: db}
	userService := &services.UserService{Repo: userRepo}
	userHandler := &handlers.UserHandler{Service: userService}

	authService := &services.AuthService{UserRepo: userRepo}
	authHandler := &handlers.AuthHandler{AuthService: authService}

	api := e.Group("/rumosaudavel-api")

	api.POST("/login", authHandler.Login)
	api.POST("/register", authHandler.Register)

	api.POST("/users", userHandler.Create)
	api.GET("/users", userHandler.List)
	api.GET("/users/:id", userHandler.Get)
	api.PUT("/users/:id", userHandler.Update)

	protected := api.Group("")
	protected.Use(middleware.JWTMiddleware)
	protected.GET("/me", authHandler.Me)
}
