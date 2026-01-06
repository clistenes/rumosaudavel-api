package routes

import (
	"rumosaudavel-api/internal/handlers"
	"rumosaudavel-api/internal/middleware"
	"rumosaudavel-api/internal/repositories"
	"rumosaudavel-api/internal/services"

	"github.com/joho/godotenv"

	"github.com/labstack/echo/v4"
	"database/sql"
)

func init() {
	godotenv.Load()
}

func Routes(e *echo.Echo, db *sql.DB) {
	dashboardHandler := handlers.NewDashboardHandler(db)

	api := e.Group("/rumosaudavel-api")

	api.GET("/dashboard/home", dashboardHandler.Home)
	api.GET("/dashboard/bugs", dashboardHandler.Bugs)

	userRepo := &repositories.UserRepository{DB: db}

	authService := &services.AuthService{UserRepo: userRepo}
	authHandler := &handlers.AuthHandler{AuthService: authService}
	
	api.POST("/login", authHandler.Login)
	api.POST("/register", authHandler.Register)

	participanteHandler := handlers.NewParticipanteHandler(db)

	api.POST("/participantes", participanteHandler.Criar)
	api.GET("/participantes/:id", participanteHandler.Lista)
	api.GET("/participantes/termometro/:id", participanteHandler.Termometro)
	api.GET("/participantes/termometro-cor", participanteHandler.TermometroCor)
	api.DELETE("/participantes/:id", participanteHandler.Apagar)
	api.DELETE("/participantes/empresa/:id", participanteHandler.ApagarPorEmpresa)
	api.POST("/participantes/gera-indices", participanteHandler.GeraIndices)
	api.POST("/participantes/upload", participanteHandler.UploadExcel)

	protected := api.Group("")
	protected.Use(middleware.JWTMiddleware)
	protected.GET("/me", authHandler.Me)
}
