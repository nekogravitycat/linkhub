package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/s3bucket"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

func isExpired(resource models.Resource, now time.Time) bool {
	if resource.Entry.ExpiresAt == nil {
		return false
	}
	return resource.Entry.ExpiresAt.UTC().Before(now.UTC())
}

func isPasswordCorrect(hash string, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return false, err
}

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

func populateDownloadURL(ctx context.Context, resp *models.GetResourceResponse, uuid string) error {
	if resp.Type != models.ResourceTypeFile {
		return fmt.Errorf("cannot populate download URL for non-file resource")
	}
	s3 := s3bucket.GetS3Client()
	var err error
	resp.File.DownloadURL, err = s3bucket.NewPresigner(s3).Get(ctx, uuid)
	if err != nil {
		return fmt.Errorf("failed to generate download URL: %w", err)
	}
	return nil
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

func toResourceFromLink(request models.CreateLinkRequest) (models.Resource, error) {
	slug := url.PathEscape(request.RawSlug)

	var passwordHash *string = nil
	if request.RawPassword != nil {
		if validator.ValidateRawPassword(*request.RawPassword) != nil {
			return models.Resource{}, fmt.Errorf("invalid password format")
		}
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(*request.RawPassword), 12)
		if err != nil {
			return models.Resource{}, fmt.Errorf("failed to hash password: %w", err)
		}
		hash := string(hashBytes)
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

func toResourceFromFile(request models.CreateFileRequest, uuid string) (models.Resource, error) {
	slug := url.PathEscape(request.RawSlug)

	var passwordHash *string = nil
	if request.RawPassword != nil {
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(*request.RawPassword), 12)
		if err != nil {
			return models.Resource{}, fmt.Errorf("failed to hash password: %w", err)
		}
		hash := string(hashBytes)
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
		},
	}, nil
}
