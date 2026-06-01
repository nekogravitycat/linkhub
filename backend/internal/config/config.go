package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	DatabaseDSN     string
	TestDatabaseDSN string
	IsProduction    bool
	AllowOrigins    []string
	RedirectDomain  string
	PprofAddr       string

	// Database connection pool tuning.
	DBMaxConns int32
	DBMinConns int32

	// Redirect-path cache. Size is the max number of cached links; TTL bounds
	// how long a cached link can be stale before it is refreshed from the DB.
	CacheSize int
	CacheTTL  time.Duration
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
		origins := strings.SplitSeq(allowOriginsRaw, ",")
		for origin := range origins {
			allowOrigins = append(allowOrigins, strings.TrimSpace(origin))
		}
	}

	return &Config{
		Port:            getEnv("PORT", "8001"),
		DatabaseDSN:     buildDSN(getEnv("POSTGRES_DB", "linkhub")),
		TestDatabaseDSN: buildDSN(getEnv("POSTGRES_TEST_DB", "linkhub_test")),
		IsProduction:    isProduction,
		AllowOrigins:    allowOrigins,
		RedirectDomain:  getEnv("REDIRECT_DOMAIN", "localhost:8003"),
		PprofAddr:       getEnv("PPROF_ADDR", ""),
		DBMaxConns:      int32(getEnvInt("POSTGRES_MAX_CONNS", 25)),
		DBMinConns:      int32(getEnvInt("POSTGRES_MIN_CONNS", 5)),
		CacheSize:       getEnvInt("REDIRECT_CACHE_SIZE", 100000),
		CacheTTL:        getEnvDuration("REDIRECT_CACHE_TTL", 60*time.Second),
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

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			return parsed
		}
		log.Printf("invalid int for %s=%q, using fallback %d", key, value, fallback)
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := time.ParseDuration(strings.TrimSpace(value)); err == nil {
			return parsed
		}
		log.Printf("invalid duration for %s=%q, using fallback %s", key, value, fallback)
	}
	return fallback
}
