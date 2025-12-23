package api

import (
	"slices"
	"strings"

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

	// Global Middleware
	r.Use(gin.Logger(), gin.Recovery())

	// CORS Config
	corsConfig := cors.DefaultConfig()

	if cfg.IsProduction {
		// Production: Strict mode, only allow exact domain match
		corsConfig.AllowOrigins = cfg.AllowOrigins
	} else {
		// Development: Use AllowOriginFunc to check dynamically
		corsConfig.AllowOriginFunc = func(origin string) bool {
			// Allow Prod Origins
			if slices.Contains(cfg.AllowOrigins, origin) {
				return true
			}
			// Allow localhost with ANY port
			if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "https://localhost") {
				return true
			}
			// Allow 127.0.0.1 with ANY port
			if strings.HasPrefix(origin, "http://127.0.0.1") || strings.HasPrefix(origin, "https://127.0.0.1") {
				return true
			}
			// Allow 192.168.*.* with ANY port
			if strings.HasPrefix(origin, "http://192.168.") || strings.HasPrefix(origin, "https://192.168.") {
				return true
			}
			return false
		}
	}

	corsConfig.AllowMethods = []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	// Register Routes
	linksHttp.RegisterRoutes(r, linkHandler)

	return r
}
