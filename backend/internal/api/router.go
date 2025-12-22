package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/nekogravitycat/linkhub/internal/config"
	linksHttp "github.com/nekogravitycat/linkhub/internal/links/http"
)

func NewRouter(cfg *config.Config, linkHandler *linksHttp.Handler) *gin.Engine {
	r := gin.Default()

	// CORS Config
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Tweak as needed for production
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Register Routes
	linksHttp.RegisterRoutes(r, linkHandler)

	return r
}
