package models

import (
	"time"
)

// GetResourceResponse defines the response structure when accessing a resource by slug.
type GetResourceResponse struct {
	Type ResourceType `json:"type" binding:"required"` // Type of the resource: "link" or "file"
	Link *PublicLink  `json:"link,omitempty"`          // Link resource data, present iff type == "link"
	File *PublicFile  `json:"file,omitempty"`          // File resource data, present iff type == "file"
}

// PublicLink represents the response payload for a link resource present to client.
type PublicLink struct {
	TargetURL string `json:"target_url"` // Destination URL to redirect when the link is accessed
}

// PublicFile represents the response payload for a file resource present to client.
type PublicFile struct {
	DownloadURL string `json:"download_url" binding:"required"` // Signed URL for downloading the file (from S3)
	Filename    string `json:"filename" binding:"required"`     // Original display name of the uploaded file
	MIMEType    string `json:"mime_type" binding:"required"`    // File MIME type (e.g., application/pdf, image/png)
	Size        int64  `json:"size" binding:"required"`         // File size in bytes
}

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

// UploadFileResponse defines the response structure after requesting to upload a file
type UploadFileResponse struct {
	FileUUID string       `json:"file_uuid" binding:"required"` // UUID for the file, used as the S3 filename
	Upload   UploadConfig `json:"upload" binding:"required"`    // Upload configuration
}

// UploadType represents the type of upload: single or multipart
type UploadType string

const (
	UploadTypeSingle    UploadType = "single"
	UploadTypeMultipart UploadType = "multipart"
)

// UploadConfig is a polymorphic struct depending on the upload type
type UploadConfig struct {
	Type      UploadType      `json:"type" binding:"required"` // "single" or "multipart"
	UploadURL *string         `json:"upload_url,omitempty"`    // For single upload only
	UploadID  *string         `json:"upload_id,omitempty"`     // For multipart upload only
	Parts     []MultipartPart `json:"parts,omitempty"`         // For multipart upload only
}

// MultipartPart represents a single part in a multipart upload
type MultipartPart struct {
	PartNumber int    `json:"part_number" binding:"required"` // Part number (1-based index)
	UploadURL  string `json:"upload_url" binding:"required"`  // Pre-signed URL for this part
}

// CompleteMultipartUploadRequest defines the request body for completing a multipart upload
type CompleteMultipartUploadRequest struct {
	FileUUID string          `json:"file_uuid" binding:"required"` // UUID of the file being uploaded
	UploadID string          `json:"upload_id" binding:"required"` // Multipart upload ID
	Parts    []CompletedPart `json:"parts" binding:"required"`     // List of all uploaded parts with ETags
}

// CompletedPart represents a part that has been uploaded and confirmed with its ETag
type CompletedPart struct {
	PartNumber int    `json:"part_number" binding:"required"` // Part number (1-based index)
	ETag       string `json:"etag" binding:"required"`        // ETag returned from S3 for the part
}
