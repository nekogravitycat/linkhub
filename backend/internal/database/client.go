package database

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	_dbClient     *pgxpool.Pool
	_onceDBClient sync.Once
)

func GetDBClient() *pgxpool.Pool {
	_onceDBClient.Do(func() {
		dsn := os.Getenv("DATABASE_URL")

		var err error
		_dbClient, err = pgxpool.New(context.Background(), dsn)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
	})

	return _dbClient
}
