package models

import (
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	Slug      string     `json:"slug" binding:"required"`       // User-defined short slug
	TargetURL string     `json:"target_url" binding:"required"` // Required: the link to redirect to
	Password  *string    `json:"password,omitempty"`            // Optional password
	ExpiresAt *time.Time `json:"expires_at,omitempty"`          // Optional expiry time
}

type CreateFileRequest struct {
	Slug      string     `json:"slug" binding:"required"`      // User-defined short slug
	Filename  string     `json:"filename" binding:"required"`  // Display name for the file
	MIMEType  string     `json:"mime_type" binding:"required"` // MIME type of the uploaded file
	Size      int64      `json:"size" binding:"required"`      // Size of the file in bytes
	Password  *string    `json:"password,omitempty"`           // Optional password
	ExpiresAt *time.Time `json:"expires_at,omitempty"`         // Optional expiry time
}

type UploadFileResponse struct {
	FileUUID string `json:"file_uuid"` // UUID for the uploaded file
}

type UploadFileMultipartResponse struct {
	UploadID string `json:"upload_id"` // ID for the multipart upload session
}

// If the resource is a file, DownloadURL will be empty and need to populate.
func ToGetResourceResponse(resource Resource) GetResourceResponse {
	resp := GetResourceResponse{
		Type: resource.Entry.Type,
	}

	switch resource.Entry.Type {
	case ResourceTypeLink:
		if resource.Link != nil {
			resp.Link = &PublicLink{
				TargetURL: resource.Link.TargetURL,
			}
		}

	case ResourceTypeFile:
		if resource.File != nil {
			resp.File = &PublicFile{
				DownloadURL: "",
				Filename:    resource.File.Filename,
				MIMEType:    resource.File.MIMEType,
				Size:        resource.File.Size,
			}
		}
	}

	return resp
}

func ToResourceFromLink(request CreateLinkRequest) (Resource, error) {
	var passwordHash *string = nil
	if request.Password != nil {
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(*request.Password), 12)
		if err != nil {
			return Resource{}, fmt.Errorf("failed to hash password: %w", err)
		}
		hash := string(hashBytes)
		passwordHash = &hash
	}

	return Resource{
		Entry: Entry{
			ID:           0, // ID will be set by the database
			Slug:         request.Slug,
			Type:         ResourceTypeLink,
			PasswordHash: passwordHash,
			CreatedAt:    time.Now(), // CreatedAt will be set by the database
			ExpiresAt:    request.ExpiresAt,
		},
		Link: &Link{
			EntryID:   0, // EntryID will be set by the database
			TargetURL: request.TargetURL,
		},
	}, nil
}

func ToResourceFromFile(request CreateFileRequest, uuid string) (Resource, error) {
	var passwordHash *string = nil
	if request.Password != nil {
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(*request.Password), 12)
		if err != nil {
			return Resource{}, fmt.Errorf("failed to hash password: %w", err)
		}
		hash := string(hashBytes)
		passwordHash = &hash
	}

	return Resource{
		Entry: Entry{
			ID:           0, // ID will be set by the database
			Slug:         request.Slug,
			Type:         ResourceTypeFile,
			PasswordHash: passwordHash,
			CreatedAt:    time.Now(), // CreatedAt will be set by the database
			ExpiresAt:    request.ExpiresAt,
		},
		File: &File{
			EntryID:  0, // EntryID will be set by the database
			FileUUID: uuid,
			Filename: request.Filename,
			MIMEType: request.MIMEType,
			Size:     request.Size,
		},
	}, nil
}
