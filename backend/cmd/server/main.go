package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nekogravitycat/linkhub/internal/api"
	"github.com/nekogravitycat/linkhub/internal/config"
	"github.com/nekogravitycat/linkhub/internal/database"
	"github.com/nekogravitycat/linkhub/internal/links"
	linksHttp "github.com/nekogravitycat/linkhub/internal/links/http"
)

const SERVER_SHUTDOWN_TIMEOUT = 5 * time.Second

func main() {
	// Setup Context for Gracedful Shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect Database
	pool, err := database.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Initialize Layers
	linkRepo := links.NewRepository(pool)
	linkService := links.NewService(linkRepo, cfg.RedirectDomain)
	linkHandler := linksHttp.NewHandler(linkService)

	// Setup Server
	r := api.NewRouter(cfg, linkHandler)

	// Setup HTTP Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Start Server
	go func() {
		log.Printf("Starting server on port %s...", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for Interrupt Signal
	<-ctx.Done()
	log.Println("Shutdown signal received")

	// Graceful Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), SERVER_SHUTDOWN_TIMEOUT)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server exited gracefully")
	}
}
