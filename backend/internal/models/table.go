package models

import "time"

/*
CREATE TABLE entries (
	id SERIAL PRIMARY KEY,
	slug TEXT UNIQUE NOT NULL,
	type TEXT NOT NULL CHECK (type IN ('link', 'file')),
	password_hash TEXT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
	expires_at TIMESTAMPTZ NULL
);

CREATE TABLE links (
	entry_id INTEGER PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
	target_url TEXT NOT NULL
);

CREATE TABLE files (
	entry_id INTEGER PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
	file_uuid CHAR(36) NOT NULL UNIQUE,
	filename TEXT NOT NULL,
	mime_type TEXT NOT NULL,
	size BIGINT NOT NULL
);
*/

type ResourceType string

const (
	ResourceTypeLink ResourceType = "link"
	ResourceTypeFile ResourceType = "file"
)

type Resource struct {
	Entry Entry `json:"entry"`          // The entry associated with the resource
	Link  *Link `json:"link,omitempty"` // Optional link details
	File  *File `json:"file,omitempty"` // Optional file details
}

type Entry struct {
	ID           int64        `json:"id"`                      // Unique identifier for the resource
	Slug         string       `json:"slug"`                    // User-defined short slug
	Type         ResourceType `json:"type"`                    // Type of resource (link or file)
	PasswordHash *string      `json:"password_hash,omitempty"` // Optional password hash
	CreatedAt    time.Time    `json:"created_at"`              // Creation timestamp
	ExpiresAt    *time.Time   `json:"expires_at,omitempty"`    // Optional expiry timestamp
}

type Link struct {
	EntryID   int64  `json:"entry_id"`   // Foreign key to entry table
	TargetURL string `json:"target_url"` // The URL to redirect to
}

type File struct {
	EntryID  int64  `json:"entry_id"`  // Foreign key to entry table
	FileUUID string `json:"file_uuid"` // File UUID (filename in s3)
	Filename string `json:"filename"`  // Display name for the file
	MIMEType string `json:"mime_type"` // MIME type of the uploaded file
	Size     int64  `json:"size"`      // File size in bytes
}
