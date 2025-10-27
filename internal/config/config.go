package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

// Config armazena todas as configurações da aplicação
type Config struct {
	DBConnectionString    string
	DBConnectionStringURL string
	LLMEndpoint           string
	LLMModel              string
	Port                  string
}

// LoadConfig lê as variáveis de ambiente e monta a string de conexão
func LoadConfig() *Config {

	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Aviso: Erro ao carregar .env: %v", err)
	}
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// Monta a string de conexão do Postgres
	connStrDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	dbPassEscaped := url.QueryEscape(dbPass)

	connStrURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassEscaped, dbHost, dbPort, dbName)
	return &Config{
		DBConnectionString:    connStrDSN,
		DBConnectionStringURL: connStrURL,
		LLMEndpoint:           os.Getenv("LLM_ENDPOINT"),
		LLMModel:              os.Getenv("LLM_MODEL"),
		Port:                  "8080", // Porta que o servidor Go vai ouvir
	}
}
