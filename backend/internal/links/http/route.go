package http

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, h *Handler) {
	// Public
	public := r.Group("/public")
	{
		public.GET("/:slug", h.Redirect)
	}

	// Private
	private := r.Group("/private")
	{
		links := private.Group("/links")
		{
			links.GET("", h.List)
			links.POST("", h.Create)
			links.GET("/:slug", h.Get)
			links.PATCH("/:slug", h.Update)
			links.DELETE("/:slug", h.Delete)
		}
	}
}
