package models

import (
	"time"
)

type PublicLink struct {
	TargetURL string `json:"target_url"` // The URL to redirect to
}

type PublicFile struct {
	DownloadURL string `json:"download_url"` // URL to download the file
	Filename    string `json:"filename"`     // Display name for the file
	MIMEType    string `json:"mime_type"`    // MIME type of the uploaded file
	Size        int64  `json:"size"`         // Size of the file in bytes
}

type GetResourceResponse struct {
	Type ResourceType `json:"type"`           // Type of resource (link or file)
	Link *PublicLink  `json:"link,omitempty"` // Optional link details
	File *PublicFile  `json:"file,omitempty"` // Optional file details
}

type UnlockResourceRequest struct {
	Password string `json:"password" binding:"required"` // Password for protected resources
}

type CreateLinkRequest struct {
	RawSlug     string     `json:"slug" binding:"required"`       // User-defined short slug
	TargetURL   string     `json:"target_url" binding:"required"` // Required: the link to redirect to
	RawPassword *string    `json:"password,omitempty"`            // Optional password
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`          // Optional expiry time
}

type CreateFileRequest struct {
	RawSlug     string     `json:"slug" binding:"required"`      // User-defined short slug
	Filename    string     `json:"filename" binding:"required"`  // Display name for the file
	MIMEType    string     `json:"mime_type" binding:"required"` // MIME type of the uploaded file
	Size        int64      `json:"size" binding:"required"`      // Size of the file in bytes
	RawPassword *string    `json:"password,omitempty"`           // Optional password
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`         // Optional expiry time
}

type UploadFileResponse struct {
	FileUUID string `json:"file_uuid"` // UUID for the uploaded file
}

type UploadFileMultipartResponse struct {
	UploadID string `json:"upload_id"` // ID for the multipart upload session
}
