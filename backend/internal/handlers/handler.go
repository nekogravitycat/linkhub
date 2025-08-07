package handlers

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nekogravitycat/linkhub/backend/internal/database"
	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
)

// Set up the HTTP routes for the application
func RegisterRoutes(router *gin.Engine) {
	// Public routes
	public := router.Group("/public")
	{
		public.GET("/resources/:slug", getResource)
		public.POST("/resources/:slug/unlock", unlockResource)
	}
	// Private routes
	private := router.Group("/private")
	{
		// Resources
		private.GET("/resources", listResources)
		private.DELETE("/resources/:slug", deleteResource)
		// Entries
		private.PATCH("/entries/:slug", updateEntryMeta)
		// Links
		private.POST("/links", createLinkResource)
		private.PATCH("/links/:slug/target-url", updateLinkTargetURL)
		// Files
		private.POST("/files", createFileResource)
		private.PATCH("/files/:slug/mark-uploaded", markFileResourceUploaded)
	}
}

// GET /public/resources/:slug
func getResource(c *gin.Context) {
	slug := c.Param("slug")
	slug = url.PathEscape(slug)
	// Validate slug format
	if err := validator.ValidateSlug(slug); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid slug", err)
		return
	}
	// Fetch the resource from the database
	resource, err := database.GetResource(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, database.ErrRowNotFound) {
			respondWithError(c, http.StatusNotFound, "Resource not found", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to get resource", err)
		return
	}
	// If the resource is expired, return a not found error
	if isExpired(resource, time.Now()) {
		respondWithError(c, http.StatusNotFound, "Resource not found", nil)
		return
	}
	// If the resource is password protected, return an forbidden error
	if resource.Entry.PasswordHash != nil {
		respondWithError(c, http.StatusForbidden, "Resource is password protected", nil)
		return
	}
	// If the resource is a file, check if it is uploaded
	if resource.Entry.Type == models.ResourceTypeFile {
		if resource.File == nil {
			respondWithError(c, http.StatusInternalServerError, "File entry exists but file is missing", nil)
			return
		}
		if resource.File.Pending {
			respondWithError(c, http.StatusConflict, "File is not uploaded yet", nil)
			return
		}
	}
	// Build the response, populate download URL if needed
	resp, err := toResponseWithDownloadURL(c.Request.Context(), resource)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to generate download URL", err)
		return
	}
	// Return the resource response
	c.JSON(http.StatusOK, resp)
}

// POST /public/resources/:slug/unlock
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

// GET /private/resources
func listResources(c *gin.Context) {
	page, errPage := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, errLimit := strconv.Atoi(c.DefaultQuery("limit", "20"))
	// Validate pagination parameters
	if errPage != nil || errLimit != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid pagination parameters, page and limit must be integers.", nil)
		return
	}
	if page < 1 || limit < 1 || limit > 100 {
		respondWithError(c, http.StatusBadRequest, "Invalid pagination parameters, page and limit must be positive and limit must not exceed 100.", nil)
		return
	}
	// List resources from the database
	offset := (page - 1) * limit
	resources, err := database.ListResources(c.Request.Context(), offset, limit)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to list resources", err)
		return
	}
	// Redact password hashes from the resources
	redactedPasswordHash := "REDACTED"
	for i := range resources {
		if resources[i].Entry.PasswordHash != nil {
			resources[i].Entry.PasswordHash = &redactedPasswordHash
		}
	}
	// Return the list of resources
	c.JSON(http.StatusOK, resources)
}

// DELETE /private/resources/:slug
func deleteResource(c *gin.Context) {
	slug := c.Param("slug")
	slug = url.PathEscape(slug)
	// Validate slug format
	if err := validator.ValidateSlug(slug); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid slug", err)
		return
	}
	// Delete the resource from the database
	if err := database.DeleteResourceBySlug(c.Request.Context(), slug); err != nil {
		if errors.Is(err, database.ErrRowNotFound) {
			respondWithError(c, http.StatusNotFound, "Resource not found", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to delete resource", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Resource deleted successfully"})
}

// PATCH /private/entries/:slug
func updateEntryMeta(c *gin.Context) {
	slug := c.Param("slug")
	slug = url.PathEscape(slug)
	// Validate slug format
	if err := validator.ValidateSlug(slug); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid slug", err)
		return
	}
	// Build the request from the body
	var request models.UpdateEntryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Malformed request body", err)
		return
	}
	// Validate the request parameters
	if err := validator.ValidateUpdateEntryRequest(request, time.Now()); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}
	// Build the entry update fields from the request
	entryUpdate := toEntryUpdate(request)
	// Update the entry in the database
	if err := database.UpdateEntry(c.Request.Context(), slug, entryUpdate); err != nil {
		if errors.Is(err, database.ErrRowNotFound) {
			respondWithError(c, http.StatusNotFound, "Entry not found", nil)
			return
		}
		if errors.Is(err, database.ErrDuplicateSlug) {
			respondWithError(c, http.StatusConflict, "Entry with this slug already exists", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to update entry", err)
		return
	}
	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Entry updated successfully"})
}

// POST /private/links
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
	if _, err := database.InsertResource(c.Request.Context(), resource); err != nil {
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

// PATCH /private/links/:slug/target-url
func updateLinkTargetURL(c *gin.Context) {
	slug := c.Param("slug")
	slug = url.PathEscape(slug)
	// Validate slug format
	if err := validator.ValidateSlug(slug); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid slug", err)
		return
	}
	// Build the request from the body
	var request models.UpdateLinkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Malformed request body", err)
		return
	}
	// Validate the request parameters
	if err := validator.ValidateUpdateLinkRequest(request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}
	// Check if the resource exists and is link type
	resource, err := database.GetResource(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, database.ErrRowNotFound) {
			respondWithError(c, http.StatusNotFound, "Resource not found", err)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to get resource", err)
		return
	}
	if resource.Entry.Type != models.ResourceTypeLink {
		respondWithError(c, http.StatusBadRequest, "Resource is not a link", nil)
		return
	}
	// Build the link from the request
	link := models.Link{
		EntryID:   resource.Entry.ID,
		TargetURL: request.TargetURL,
	}
	// Update the link in the database
	if err := database.UpdateLink(c.Request.Context(), link); err != nil {
		if errors.Is(err, database.ErrRowNotFound) {
			respondWithError(c, http.StatusInternalServerError, "Resource exists but link not found", err)
			return
		}
		if errors.Is(err, database.ErrDuplicateSlug) {
			respondWithError(c, http.StatusConflict, "Link with this slug already exists", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to update link", err)
		return
	}
	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Link updated successfully"})
}

// POST /private/files
func createFileResource(c *gin.Context) {
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
	fileUUID := uuid.NewString()
	resource, err := toResourceFromFile(request, fileUUID)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "Failed to process request", err)
		return
	}
	// Insert the resource into the database
	entryID, err := database.InsertResource(c.Request.Context(), resource)
	if err != nil {
		if errors.Is(err, database.ErrDuplicateSlug) {
			respondWithError(c, http.StatusConflict, "Resource with this slug already exists", nil)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to create file resource", err)
		return
	}
	// Generate the upload response
	resp, err := generateUploadResponse(c.Request.Context(), request, fileUUID)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to generate upload response", err)
		// Clean up the resource if upload fails
		if err := database.DeleteResourceByID(c.Request.Context(), entryID); err != nil {
			log.Printf("[ERROR] Failed to clean up file resource (%d) after generate upload response failure: %v", entryID, err)
		}
		return
	}
	// Return the response
	c.JSON(http.StatusOK, resp)
}

// PATCH /private/files/:slug/mark-uploaded
func markFileResourceUploaded(c *gin.Context) {
	slug := c.Param("slug")
	slug = url.PathEscape(slug)
	// Validate slug format
	if err := validator.ValidateSlug(slug); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid slug", err)
		return
	}
	// Build the request from the body
	var request models.UploadFileCompleteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Malformed request body", err)
		return
	}
	// Validate the request parameters
	if err := validator.ValidateUploadFileCompleteRequest(request); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}
	// Check if the resource exists and is file type
	resource, err := database.GetResource(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, database.ErrRowNotFound) {
			respondWithError(c, http.StatusNotFound, "Resource not found", err)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to get resource", err)
		return
	}
	if resource.Entry.Type != models.ResourceTypeFile {
		respondWithError(c, http.StatusBadRequest, "Resource is not a file", nil)
		return
	}
	if resource.File == nil {
		respondWithError(c, http.StatusInternalServerError, "File entry exists but file is missing", nil)
		return
	}
	// If the file is already marked as uploaded, return a conflict error
	if !resource.File.Pending {
		respondWithError(c, http.StatusConflict, "File is already marked as uploaded", nil)
		return
	}
	// If the request is multipart upload, complete the multipart upload
	if request.Type == models.UploadTypeMultipart {
		if err := completeMultipartUpload(c.Request.Context(), request.FileUUID, *request.Multipart); err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to complete multipart upload", err)
			return
		}
	}
	// Mark the file as uploaded in the database
	if err := database.MarkFileAsUploaded(c.Request.Context(), resource.Entry.ID); err != nil {
		if errors.Is(err, database.ErrRowNotFound) {
			respondWithError(c, http.StatusInternalServerError, "Resource exists but file not found", err)
			return
		}
		respondWithError(c, http.StatusInternalServerError, "Failed to mark file as uploaded", err)
		return
	}
	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "File marked as uploaded successfully"})
}
