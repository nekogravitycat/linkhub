package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nekogravitycat/linkhub/internal/links"
)

type Handler struct {
	service links.Service
}

func NewHandler(service links.Service) *Handler {
	return &Handler{service: service}
}

// Public: Redirect
func (h *Handler) Redirect(c *gin.Context) {
	var uri BySlug
	if err := c.ShouldBindUri(&uri); err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if err := ValidateSlug(uri.Slug); err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	link, err := h.service.Get(c.Request.Context(), uri.Slug)
	if err != nil {
		if errors.Is(err, links.ErrLinkNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if !link.IsActive {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Redirect(http.StatusFound, link.URL)
}

// Private: List
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	opts := links.ListOptions{
		ListParams: req.ListParams,
		SortBy:     req.SortBy,
	}

	list, err := h.service.List(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Private: Get
func (h *Handler) Get(c *gin.Context) {
	var uri BySlug
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ValidateSlug(uri.Slug); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	link, err := h.service.Get(c.Request.Context(), uri.Slug)
	if err != nil {
		if errors.Is(err, links.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, link)
}

// Private: Create
func (h *Handler) Create(c *gin.Context) {
	var req CreateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.Create(c.Request.Context(), req.Slug, req.URL)
	if err != nil {
		if errors.Is(err, links.ErrSlugTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "slug already taken"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// Private: Update
func (h *Handler) Update(c *gin.Context) {
	var uri BySlug
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ValidateSlug(uri.Slug); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req UpdateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.Update(c.Request.Context(), uri.Slug, req.URL, req.IsActive)
	if err != nil {
		if errors.Is(err, links.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// Private: Delete
func (h *Handler) Delete(c *gin.Context) {
	var uri BySlug
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ValidateSlug(uri.Slug); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.Delete(c.Request.Context(), uri.Slug)
	if err != nil {
		if errors.Is(err, links.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
