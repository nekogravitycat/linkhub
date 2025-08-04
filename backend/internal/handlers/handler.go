package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nekogravitycat/linkhub/backend/internal/database"
	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// GET /resource/:slug
func GetResource(c *gin.Context) {
	slug := c.Param("slug")

	resource, err := database.GetResource(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	if resource.Entry.PasswordHash != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Resource is password protected"})
		return
	}

	resp, err := toResponseWithDownloadURL(c.Request.Context(), resource)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download URL"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /resource/:slug/unlock
func UnlockResource(c *gin.Context) {
	slug := c.Param("slug")

	var request models.UnlockResourceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resource, err := database.GetResource(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	if resource.Entry.PasswordHash == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Resource is not password protected"})
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(*resource.Entry.PasswordHash),
		[]byte(request.Password),
	); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password"})
			return
		}
	}

	resp, err := toResponseWithDownloadURL(c.Request.Context(), resource)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download URL"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /private/link
func CreateResource(c *gin.Context) {
	var request models.CreateLinkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handler logic for creating a resource
	c.JSON(http.StatusCreated, gin.H{"message": "Resource created"})
}

// POST /private/file
func CreateFileResource(c *gin.Context) {
	var request models.CreateFileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handler logic for creating a file resource
	c.JSON(http.StatusCreated, gin.H{"message": "File resource created"})
}

// POST /private/file/complete-multipart
func CompleteMultipartUpload(c *gin.Context) {
	// Handler logic for completing a multipart upload
	// This would typically involve finalizing the file upload process
	c.JSON(http.StatusOK, gin.H{"message": "Multipart upload completed"})
}

// GET /private/resources
func GetResources(c *gin.Context) {
	// Handler logic for retrieving all resources
	c.JSON(200, gin.H{"message": "Resources retrieved"})
}

// PATCH /private/resources/:slug
func UpdateResource(c *gin.Context) {
	// Handler logic for updating a resource by slug
	slug := c.Param("slug")
	c.JSON(200, gin.H{"slug": slug, "message": "Resource updated"})
}

// DELETE /private/resources/:slug
func DeleteResource(c *gin.Context) {
	// Handler logic for deleting a resource by slug
	slug := c.Param("slug")
	c.JSON(200, gin.H{"slug": slug, "message": "Resource deleted"})
}
