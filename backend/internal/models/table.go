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
	size BIGINT NOT NULL,
	pending BOOLEAN DEFAULT TRUE NOT NULL
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
	ID           int64        `json:"id"`                      // Unique identifier assigned by the database
	Slug         string       `json:"slug"`                    // URL-escaped slug identifier (must be unique)
	Type         ResourceType `json:"type"`                    // Type of resource: either "link" or "file"
	PasswordHash *string      `json:"password_hash,omitempty"` // Optional bcrypt hash of the password, if password-protected
	CreatedAt    time.Time    `json:"created_at"`              // Timestamp when the resource was created (set automatically)
	ExpiresAt    *time.Time   `json:"expires_at,omitempty"`    // Optional expiration timestamp; after this, the resource is considered expired
}

type Link struct {
	EntryID   int64  `json:"entry_id"`   // Foreign key referencing the associated entry
	TargetURL string `json:"target_url"` // The external URL to redirect to when accessing the resource
}

type File struct {
	EntryID  int64  `json:"entry_id"`  // Foreign key referencing the associated entry
	FileUUID string `json:"file_uuid"` // Unique identifier for the file, also used as the filename in S3
	Filename string `json:"filename"`  // Original file name (used for display or download)
	MIMEType string `json:"mime_type"` // MIME type of the file (e.g., application/pdf)
	Size     int64  `json:"size"`      // File size in bytes
	Pending  bool   `json:"pending"`   // True if file is awaiting upload; false once confirmed uploaded
}

type EntryUpdate struct {
	Slug           *string    `json:"slug,omitempty"`          // Optional new slug (must be unique)
	PasswordHash   *string    `json:"password_hash,omitempty"` // Optional new password hash
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`    // Optional new expiration time
	UpdatePassword bool       `json:"update_password"`         // Whether to update the password hash
}

type LinkUpdate struct {
	TargetURL string `json:"target_url,omitempty"` // New target URL for the link
}
