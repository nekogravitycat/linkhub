package handlers

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
)

// If the resource is a file, DownloadURL will be empty and need to populate.
func toGetResourceResponse(resource models.Resource) models.GetResourceResponse {
	resp := models.GetResourceResponse{
		Type: resource.Entry.Type,
	}

	switch resource.Entry.Type {
	case models.ResourceTypeLink:
		if resource.Link != nil {
			resp.Link = &models.PublicLink{
				TargetURL: resource.Link.TargetURL,
			}
		}

	case models.ResourceTypeFile:
		if resource.File != nil {
			resp.File = &models.PublicFile{
				DownloadURL: "",
				Filename:    resource.File.Filename,
				MIMEType:    resource.File.MIMEType,
				Size:        resource.File.Size,
			}
		}
	}

	return resp
}

// Build GetResourceResponse from Resource.
// If the resource is a file, it populates the DownloadURL.
// Return an error if the download URL cannot be generated.
func toResponseWithDownloadURL(ctx context.Context, resource models.Resource) (models.GetResourceResponse, error) {
	resp := toGetResourceResponse(resource)

	// If the resource is a file, populate the DownloadURL.
	if resp.Type == models.ResourceTypeFile {
		if err := populateDownloadURL(ctx, &resp, resource.File.FileUUID); err != nil {
			return models.GetResourceResponse{}, fmt.Errorf("failed to populate download URL: %w", err)
		}
	}

	return resp, nil
}

// Build Resource from CreateLinkRequest.
// The slug is URL-escaped.
// If the password is provided, it is hashed using bcrypt.
func toResourceFromLink(request models.CreateLinkRequest) (models.Resource, error) {
	slug := url.PathEscape(request.RawSlug)

	var passwordHash *string = nil
	if request.RawPassword != nil {
		if validator.ValidateRawPassword(*request.RawPassword) != nil {
			return models.Resource{}, fmt.Errorf("invalid password format")
		}
		hash, err := calculatePasswordHash(*request.RawPassword)
		if err != nil {
			return models.Resource{}, fmt.Errorf("failed to hash password: %w", err)
		}
		passwordHash = &hash
	}

	return models.Resource{
		Entry: models.Entry{
			ID:           0, // ID will be set by the database
			Slug:         slug,
			Type:         models.ResourceTypeLink,
			PasswordHash: passwordHash,
			CreatedAt:    time.Now(), // CreatedAt will be set by the database
			ExpiresAt:    request.ExpiresAt,
		},
		Link: &models.Link{
			EntryID:   0, // EntryID will be set by the database
			TargetURL: request.TargetURL,
		},
	}, nil
}

// Build Resource from CreateFileRequest and file UUID.
// The slug is URL-escaped.
// If the password is provided, it is hashed using bcrypt.
// Pending will be set to true by default.
func toResourceFromFile(request models.CreateFileRequest, uuid string) (models.Resource, error) {
	slug := url.PathEscape(request.RawSlug)

	var passwordHash *string = nil
	if request.RawPassword != nil {
		hash, err := calculatePasswordHash(*request.RawPassword)
		if err != nil {
			return models.Resource{}, fmt.Errorf("failed to hash password: %w", err)
		}
		passwordHash = &hash
	}

	return models.Resource{
		Entry: models.Entry{
			ID:           0, // ID will be set by the database
			Slug:         slug,
			Type:         models.ResourceTypeFile,
			PasswordHash: passwordHash,
			CreatedAt:    time.Now(), // CreatedAt will be set by the database
			ExpiresAt:    request.ExpiresAt,
		},
		File: &models.File{
			EntryID:  0, // EntryID will be set by the database
			FileUUID: uuid,
			Filename: request.Filename,
			MIMEType: request.MIMEType,
			Size:     request.Size,
			Pending:  true, // Pending will be set by the database
		},
	}, nil
}

func toEntryUpdate(request models.UpdateEntryRequest) models.EntryUpdate {
	fields := models.EntryUpdate{}

	if request.RawSlug != nil {
		slug := url.PathEscape(*request.RawSlug)
		fields.Slug = &slug
	}
	if request.RawPassword != nil {
		hash, _ := calculatePasswordHash(*request.RawPassword)
		fields.PasswordHash = &hash
	}
	fields.ExpiresAt = request.ExpiresAt
	fields.UpdatePassword = request.UpdatePassword

	return fields
}
