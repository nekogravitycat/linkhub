package models

import "time"

/*
CREATE TABLE resources (
	id SERIAL PRIMARY KEY,
	slug VARCHAR(512) UNIQUE NOT NULL,
	type VARCHAR(10) NOT NULL CHECK (type IN ('link', 'file')),
	password_hash TEXT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
	expires_at TIMESTAMPTZ NULL
);

CREATE TABLE links (
	resource_id INTEGER PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
	target_url TEXT NOT NULL
);

CREATE TABLE files (
	resource_id INTEGER PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
	sha256 CHAR(64) NOT NULL UNIQUE,
	original_filename TEXT NOT NULL,
	mime_type TEXT NOT NULL,
	size BIGINT NOT NULL
);
*/

type CreateLinkRequest struct {
	Slug      string     `json:"slug" binding:"required"`       // User-defined short slug
	TargetURL string     `json:"target_url" binding:"required"` // Required: the link to redirect to
	Password  *string    `json:"password,omitempty"`            // Optional password
	ExpiresAt *time.Time `json:"expires_at,omitempty"`          // Optional expiry time
}

type CreateFileRequest struct {
	Slug             string     `json:"slug" binding:"required"`              // User-defined short slug
	SHA256           string     `json:"sha256" binding:"required"`            // File content hash (pre-uploaded to R2)
	OriginalFilename string     `json:"original_filename" binding:"required"` // Display name for the file
	MIMEType         string     `json:"mime_type" binding:"required"`         // MIME type of the uploaded file
	Size             int64      `json:"size" binding:"required"`              // File size in bytes
	Password         *string    `json:"password,omitempty"`                   // Optional password
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`                 // Optional expiry time
}
