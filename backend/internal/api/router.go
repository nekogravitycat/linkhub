package api

import (
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
	corsConfig := cors.DefaultConfig()

	if cfg.IsProduction {
		// Production: Strict mode, only allow exact domain match
		corsConfig.AllowOrigins = cfg.AllowOrigins
	} else {
		// Development: Allow all origins
		corsConfig.AllowAllOrigins = true
	}

	corsConfig.AllowMethods = []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}

	r.Use(cors.New(corsConfig))

	// Register Routes
	linksHttp.RegisterRoutes(r, linkHandler)

	return r
}
