package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nekogravitycat/linkhub/backend/internal/database"
	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
)

func RegisterRoutes(router *gin.Engine) {
	// Public routes
	router.GET("/resource/:slug", getResource)
	router.POST("/resource/:slug/unlock", unlockResource)

	// Private routes
	private := router.Group("/private")
	{
		private.POST("/link", createLinkResource)
		private.POST("/file", createFileResource)
		private.POST("/file/complete-multipart", completeMultipartUpload)
		private.GET("/resources", getResources)
		private.PATCH("/resources/:slug", updateResource)
		private.DELETE("/resources/:slug", deleteResource)
	}
}

// GET /resource/:slug
func getResource(c *gin.Context) {
	slug := c.Param("slug")

	if err := validator.ValidateSlug(slug); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid slug", err)
		return
	}

	resource, err := database.GetResource(c.Request.Context(), slug)
	if err != nil {
		respondWithError(c, http.StatusNotFound, "Resource not found", err)
		return
	}

	if isExpired(resource, time.Now()) {
		respondWithError(c, http.StatusNotFound, "Resource not found", nil)
		return
	}

	if resource.Entry.PasswordHash != nil {
		respondWithError(c, http.StatusForbidden, "Resource is password protected", nil)
		return
	}

	resp, err := toResponseWithDownloadURL(c.Request.Context(), resource)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to generate download URL", err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /resource/:slug/unlock
func unlockResource(c *gin.Context) {
	slug := c.Param("slug")

	if err := validator.ValidateSlug(slug); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid slug", err)
		return
	}

	var request models.UnlockResourceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Malformed request body", err)
		return
	}

	resource, err := database.GetResource(c.Request.Context(), slug)
	if err != nil {
		respondWithError(c, http.StatusNotFound, "Resource not found", err)
		return
	}

	if isExpired(resource, time.Now()) {
		respondWithError(c, http.StatusNotFound, "Resource not found", nil)
		return
	}

	if resource.Entry.PasswordHash == nil {
		respondWithError(c, http.StatusBadRequest, "Resource is not password protected", nil)
		return
	}

	ok, err := isPasswordCorrect(*resource.Entry.PasswordHash, request.Password)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to verify password", err)
		return
	} else if !ok {
		respondWithError(c, http.StatusUnauthorized, "Invalid password", nil)
		return
	}

	resp, err := toResponseWithDownloadURL(c.Request.Context(), resource)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to generate download URL", err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /private/link
func createLinkResource(c *gin.Context) {
	var request models.CreateLinkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Malformed request body", err)
		return
	}

	if err := validator.ValidateCreateLinkRequest(request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	resource, err := models.ToResourceFromLink(request)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "Failed to process request", err)
		return
	}

	_, err = database.InsertResource(c.Request.Context(), resource)
	if err != nil {
		if errors.Is(err, database.ErrDuplicateSlug) {
			respondWithError(c, http.StatusConflict, "Resource with this slug already exists", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to create link resource", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Resource created successfully"})
}

// POST /private/file
func createFileResource(c *gin.Context) {
	var request models.CreateFileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Malformed request body", err)
		return
	}

	if err := validator.ValidateCreateFileRequest(request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	resource, err := models.ToResourceFromFile(request, uuid.NewString())
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "Failed to process request", err)
		return
	}

	_, err = database.InsertResource(c.Request.Context(), resource)
	if err != nil {
		if errors.Is(err, database.ErrDuplicateSlug) {
			respondWithError(c, http.StatusConflict, "Resource with this slug already exists", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to create file resource", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "File resource created successfully"})
}

// POST /private/file/complete-multipart
func completeMultipartUpload(c *gin.Context) {
	// Handler logic for completing a multipart upload
	// This would typically involve finalizing the file upload process
	c.JSON(http.StatusOK, gin.H{"message": "Multipart upload completed"})
}

// GET /private/resources
func getResources(c *gin.Context) {
	// Handler logic for retrieving all resources
	c.JSON(200, gin.H{"message": "Resources retrieved"})
}

// PATCH /private/resources/:slug
func updateResource(c *gin.Context) {
	// Handler logic for updating a resource by slug
	slug := c.Param("slug")
	c.JSON(200, gin.H{"slug": slug, "message": "Resource updated"})
}

// DELETE /private/resources/:slug
func deleteResource(c *gin.Context) {
	// Handler logic for deleting a resource by slug
	slug := c.Param("slug")
	c.JSON(200, gin.H{"slug": slug, "message": "Resource deleted"})
}
