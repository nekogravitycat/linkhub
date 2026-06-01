package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof handlers on http.DefaultServeMux
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

const (
	SERVER_SHUTDOWN_TIMEOUT = 5 * time.Second

	// HTTP server timeouts. Without these a slow or idle client can pin a
	// goroutine indefinitely. WriteTimeout comfortably exceeds the slowest
	// handler (list); redirects/lookups are sub-millisecond.
	SERVER_READ_HEADER_TIMEOUT = 5 * time.Second
	SERVER_READ_TIMEOUT        = 10 * time.Second
	SERVER_WRITE_TIMEOUT       = 15 * time.Second
	SERVER_IDLE_TIMEOUT        = 60 * time.Second
)

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

	// Initialize Layers. The repository is wrapped with an in-memory cache for
	// the hot single-slug lookup path (redirect/get); writes invalidate it.
	linkRepo := links.NewRepository(pool)
	linkRepo = links.NewCachedRepository(linkRepo, cfg.CacheSize, cfg.CacheTTL)
	linkService := links.NewService(linkRepo, cfg.RedirectDomain)
	linkHandler := linksHttp.NewHandler(linkService)

	// Start pprof debug server when explicitly enabled (dev only).
	// Served on its own listener/mux so profiling endpoints never touch the
	// main API router or its port.
	if cfg.PprofAddr != "" {
		go func() {
			log.Printf("Starting pprof server on %s (http://%s/debug/pprof/)", cfg.PprofAddr, cfg.PprofAddr)
			if err := http.ListenAndServe(cfg.PprofAddr, nil); err != nil && err != http.ErrServerClosed {
				log.Printf("pprof server stopped: %v", err)
			}
		}()
	}

	// Setup Server
	r := api.NewRouter(cfg, linkHandler)

	// Setup HTTP Server
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: SERVER_READ_HEADER_TIMEOUT,
		ReadTimeout:       SERVER_READ_TIMEOUT,
		WriteTimeout:      SERVER_WRITE_TIMEOUT,
		IdleTimeout:       SERVER_IDLE_TIMEOUT,
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
