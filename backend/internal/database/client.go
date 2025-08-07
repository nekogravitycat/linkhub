package database

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicateSlug = errors.New("duplicate slug")
var ErrRowNotFound = errors.New("entry not found")

var (
	_dbClient     *pgxpool.Pool
	_onceDBClient sync.Once
)

// Initialize and returns a singleton database client.
// It reads the database connection string from environment variable and ensures the client is created only once.
func GetDBClient() *pgxpool.Pool {
	_onceDBClient.Do(func() {
		dsn, ok := os.LookupEnv("DATABASE_URL")
		if !ok || dsn == "" {
			log.Fatal("DATABASE_URL environment variable is not set")
		}

		var err error
		_dbClient, err = pgxpool.New(context.Background(), dsn)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
	})

	return _dbClient
}
