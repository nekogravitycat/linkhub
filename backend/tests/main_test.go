package tests

import (
	"context"
	"fmt"
	"log"

	"os"

	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nekogravitycat/linkhub/internal/config"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	if err := runTestMain(m); err != nil {
		log.Printf("%v\n", err)
		os.Exit(1)
	}
}

func runTestMain(m *testing.M) error {
	// 0. Load .env file if it exists
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	ctx := context.Background()

	// 1. Get Test DB Configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	testDB := cfg.TestDatabaseDSN
	if testDB == "" {
		log.Fatal("Cannot run integration tests: TestDatabaseDSN not set (or constructed empty)")
	}

	// 2. Connect to Database
	poolConfig, err := pgxpool.ParseConfig(testDB)
	if err != nil {
		return fmt.Errorf("unable to parse database config: %w", err)
	}

	testPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}
	defer testPool.Close()

	if err := testPool.Ping(ctx); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	// 3. Run Tests
	code := m.Run()

	// 4. Cleanup
	if err := clearDatabase(ctx, testPool); err != nil {
		log.Printf("failed to clear database: %v\n", err)
	}

	if code != 0 {
		os.Exit(code)
	}

	return nil
}

func clearDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	// Truncate tables to clear data but keep schema
	_, err := pool.Exec(ctx, "TRUNCATE links CASCADE;")
	if err != nil {
		return fmt.Errorf("failed to truncate links table: %w", err)
	}
	return nil
}
