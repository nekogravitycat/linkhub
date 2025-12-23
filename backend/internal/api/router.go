package api

import (
	"slices"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/nekogravitycat/linkhub/internal/config"
	linksHttp "github.com/nekogravitycat/linkhub/internal/links/http"
)

func NewRouter(cfg *config.Config, linkHandler *linksHttp.Handler) *gin.Engine {
	if cfg.IsProduction {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// CORS Config
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	if !cfg.IsProduction {
		corsConfig.AllowOriginFunc = func(origin string) bool {
			// Check if it matches allowed origins from env
			if slices.Contains(cfg.AllowOrigins, origin) {
				return true
			}
			// Check for localhost (any port)
			// Matches http://localhost, https://localhost, http://localhost:PORT, https://localhost:PORT
			return origin == "http://localhost" ||
				origin == "https://localhost" ||
				strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "https://localhost:")
		}
	} else {
		corsConfig.AllowOrigins = cfg.AllowOrigins
	}

	r.Use(cors.New(corsConfig))

	// Register Routes
	linksHttp.RegisterRoutes(r, linkHandler)

	return r
}
