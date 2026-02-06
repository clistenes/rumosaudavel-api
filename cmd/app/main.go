package main

import (
	"fmt"
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
		port = "443"
	}

	certFile := os.Getenv("SSL_CERT_FILE")
	keyFile := os.Getenv("SSL_KEY_FILE")

	if certFile == "" || keyFile == "" {
		fmt.Println("Certificado SSL ou chave não fornecidos. Iniciando em modo HTTP.")
	}

	log.Println("Servidor HTTPS rodando na porta " + port)
	if os.Getenv("ENV") == "DEV" {
		e.Logger.Fatal(e.Start(":" + port))
	} else {
		e.Logger.Fatal(e.StartTLS(":"+port, certFile, keyFile))
	}
}
