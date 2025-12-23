package http

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, h *Handler) {
	r.GET("/redirect/:slug", h.Redirect)

	links := r.Group("/links")
	{
		links.GET("", h.List)
		links.POST("", h.Create)
		links.GET("/:slug", h.Get)
		links.PATCH("/:slug", h.Update)
		links.DELETE("/:slug", h.Delete)
	}
}
