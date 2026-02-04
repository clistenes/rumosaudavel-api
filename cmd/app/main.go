package main

import (
	"log"
	"os"

	"rumosaudavel-api/internal/config/database"
	"rumosaudavel-api/internal/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

func main() {
	_ = godotenv.Load()

	db := database.ConnectMySQL()

	e := echo.New()
	routes.Routes(e, db)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "9443"
	}

	log.Println("Servidor rodando na porta " + port)
	e.Logger.Fatal(e.Start(":" + port))
}