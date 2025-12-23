package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	DatabaseDSN     string
	TestDatabaseDSN string
	IsProduction    bool
	AllowOrigins    []string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	appEnv := getEnv("APP_ENV", "production")
	isProduction := true
	if appEnv == "development" {
		isProduction = false
	}

	allowOriginsRaw := getEnv("ALLOW_ORIGINS", "")
	var allowOrigins []string
	if allowOriginsRaw != "" {
		allowOrigins = strings.Split(allowOriginsRaw, ",")
	}

	return &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseDSN:     buildDSN(getEnv("POSTGRES_DB", "linkhub")),
		TestDatabaseDSN: buildDSN(getEnv("POSTGRES_TEST_DB", "linkhub_test")),
		IsProduction:    isProduction,
		AllowOrigins:    allowOrigins,
	}, nil
}

func buildDSN(dbName string) string {
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "postgres")
	addr := getEnv("POSTGRES_ADDR", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, addr, port, dbName)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
