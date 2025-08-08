package models

import (
	"time"
)

// UnlockResourceRequest is used to unlock a password-protected resource.
type UnlockResourceRequest struct {
	Password string `json:"password" binding:"required"` // Plain-text password provided by the user
}

// UpdateEntryRequest defines the request payload for updating an entry's metadata.
type UpdateEntryRequest struct {
	RawSlug        *string    `json:"slug,omitempty"`                     // Optional unescaped slug identifier (must be unique)
	RawPassword    *string    `json:"password,omitempty"`                 // Optional plain-text password to protect access
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`               // Optional expiration time
	UpdatePassword *bool      `json:"update_password" binding:"required"` // Whether to update the password hash
}

// CreateLinkRequest defines the request payload for creating a new link resource.
type CreateLinkRequest struct {
	RawSlug     string     `json:"slug" binding:"required"`       // Unescaped slug identifier (must be unique)
	TargetURL   string     `json:"target_url" binding:"required"` // Target URL to redirect when the slug is accessed
	RawPassword *string    `json:"password,omitempty"`            // Optional plain-text password to protect access
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`          // Optional expiration time
}

// UpdateLinkRequest defines the request payload for updating an existing link resource.
type UpdateLinkRequest struct {
	TargetURL string `json:"target_url" binding:"required"` // Target URL
}

// CreateFileRequest defines the request payload for initiating a new file resource.
type CreateFileRequest struct {
	RawSlug     string     `json:"slug" binding:"required"`      // Unescaped slug identifier (must be unique)
	Filename    string     `json:"filename" binding:"required"`  // Display name for the file (shown to the user)
	MIMEType    string     `json:"mime_type" binding:"required"` // MIME type of the file
	Size        int64      `json:"size" binding:"required"`      // Declared file size in bytes
	RawPassword *string    `json:"password,omitempty"`           // Optional plain-text password to protect the file
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`         // Optional expiration time
}

// UploadFileCompleteRequest defines the request payload for completing a file upload.
type UploadFileCompleteRequest struct {
	Type      UploadType             `json:"type" binding:"required"` // Type of upload: "single" or "multipart"
	Multipart *MultipartCompleteInfo `json:"multipart,omitempty"`     // Multipart upload completion details, present iff type == "multipart"
}

// MultipartCompleteInfo defines the request body for completing a multipart file upload.
type MultipartCompleteInfo struct {
	UploadID string                  `json:"upload_id" binding:"required"` // Multipart upload ID
	Parts    []MultipartCompletePart `json:"parts" binding:"required"`     // List of all uploaded parts with ETags
}

// MultipartCompletePart represents a single uploaded part of a multipart upload.
type MultipartCompletePart struct {
	PartNumber int32  `json:"part_number" binding:"required"` // Part number (1-based index)
	ETag       string `json:"etag" binding:"required"`        // ETag returned from S3 for the part
}
