package database

import (
	"fmt"
	"log"
	"os"

	sql "database/sql"
)

func ConnectMySQL() *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Erro ao verificar conexão com o banco: %v", err)
	}

	log.Println("Banco de dados conectado")
	return db
}
