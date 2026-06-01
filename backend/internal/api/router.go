package api

import (
	"slices"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/nekogravitycat/linkhub/internal/config"
	linksHttp "github.com/nekogravitycat/linkhub/internal/links/http"
)

func NewRouter(cfg *config.Config, linkHandler *linksHttp.Handler) *gin.Engine {
	if cfg.IsProduction {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global Middleware. In production nginx already logs access, and the
	// per-request gin.Logger() is synchronous overhead on the hot redirect
	// path, so only attach it in development. Recovery is always on.
	if !cfg.IsProduction {
		r.Use(gin.Logger())
	}
	r.Use(gin.Recovery())

	// CORS Config
	corsConfig := cors.DefaultConfig()

	if cfg.IsProduction {
		// Production: Strict mode, only allow allowed origins
		corsConfig.AllowOriginFunc = func(origin string) bool {
			return slices.Contains(cfg.AllowOrigins, origin)
		}
	} else {
		// Development: Allow all origins
		corsConfig.AllowOriginFunc = func(origin string) bool {
			return true
		}
	}

	corsConfig.AllowMethods = []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS", "HEAD"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	// Register Routes
	linksHttp.RegisterRoutes(r, linkHandler)

	return r
}
