package handlers

import (
	"errors"
	"net/http"
	"net/url"
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
		private.POST("/file", initCreateFileResource)
		private.POST("/file/complete-multipart", completeMultipartUpload)
		private.GET("/resources", getResources)
		private.PATCH("/resources/:slug", updateResource)
		private.DELETE("/resources/:slug", deleteResource)
	}
}

// GET /resource/:slug
func getResource(c *gin.Context) {
	slug := c.Param("slug")
	slug = url.PathEscape(slug)

	// Validate slug format
	if err := validator.ValidateSlug(slug); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid slug", err)
		return
	}
	// Check if the resource exists and is not expired
	resource, err := database.GetResource(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, database.ErrEntryNotFound) {
			respondWithError(c, http.StatusNotFound, "Resource not found", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to get resource", err)
		return
	}
	if isExpired(resource, time.Now()) {
		respondWithError(c, http.StatusNotFound, "Resource not found", nil)
		return
	}
	// If the resource is password protected, return an forbidden error
	if resource.Entry.PasswordHash != nil {
		respondWithError(c, http.StatusForbidden, "Resource is password protected", nil)
		return
	}
	// If the resource is a file, generate a download URL
	resp, err := toResponseWithDownloadURL(c.Request.Context(), resource)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to generate download URL", err)
		return
	}
	// Return the resource response
	c.JSON(http.StatusOK, resp)
}

// POST /resource/:slug/unlock
func unlockResource(c *gin.Context) {
	slug := c.Param("slug")
	slug = url.PathEscape(slug)

	// Validate slug format
	if err := validator.ValidateSlug(slug); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid slug", err)
		return
	}
	// Build the request from the body
	var request models.UnlockResourceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Malformed request body", err)
		return
	}
	// Check if the resource exists and is not expired
	resource, err := database.GetResource(c.Request.Context(), slug)
	if err != nil {
		respondWithError(c, http.StatusNotFound, "Resource not found", err)
		return
	}
	if isExpired(resource, time.Now()) {
		respondWithError(c, http.StatusNotFound, "Resource not found", nil)
		return
	}
	// If the resource is not password protected, return an error
	if resource.Entry.PasswordHash == nil {
		respondWithError(c, http.StatusBadRequest, "Resource is not password protected", nil)
		return
	}
	// Verify the password
	ok, err := isPasswordCorrect(*resource.Entry.PasswordHash, request.Password)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to verify password", err)
		return
	} else if !ok {
		respondWithError(c, http.StatusUnauthorized, "Invalid password", nil)
		return
	}
	// If the resource is a file, generate a download URL
	resp, err := toResponseWithDownloadURL(c.Request.Context(), resource)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to generate download URL", err)
		return
	}
	// Return the resource response
	c.JSON(http.StatusOK, resp)
}

// POST /private/link
func createLinkResource(c *gin.Context) {
	// Build the request from the body
	var request models.CreateLinkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Malformed request body", err)
		return
	}
	// Validate the request parameters
	if err := validator.ValidateCreateLinkRequest(request, time.Now()); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}
	// Build the resource from the request
	resource, err := toResourceFromLink(request)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "Failed to process request", err)
		return
	}
	// Insert the resource into the database
	_, err = database.InsertResource(c.Request.Context(), resource)
	if err != nil {
		if errors.Is(err, database.ErrDuplicateSlug) {
			respondWithError(c, http.StatusConflict, "Resource with this slug already exists", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to create link resource", err)
		return
	}
	// Return success response
	c.JSON(http.StatusCreated, gin.H{"message": "Resource created successfully"})
}

// POST /private/file
func initCreateFileResource(c *gin.Context) {
	// Build the request from the body
	var request models.CreateFileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Malformed request body", err)
		return
	}
	// Validate the request parameters
	if err := validator.ValidateCreateFileRequest(request, time.Now()); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}
	// Build the resource from the request, generating a new UUID for the file
	resource, err := toResourceFromFile(request, uuid.NewString())
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "Failed to process request", err)
		return
	}
	// Insert the resource into the database
	_, err = database.InsertResource(c.Request.Context(), resource)
	if err != nil {
		if errors.Is(err, database.ErrDuplicateSlug) {
			respondWithError(c, http.StatusConflict, "Resource with this slug already exists", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to create file resource", err)
		return
	}
	// Return success response
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
