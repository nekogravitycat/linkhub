package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/s3bucket"
	"golang.org/x/crypto/bcrypt"
)

// Build GetResourceResponse from Resource.
// If the resource is a file, it populates the DownloadURL.
// Return an error if the download URL cannot be generated.
func toResponseWithDownloadURL(ctx context.Context, resource models.Resource) (models.GetResourceResponse, error) {
	resp := models.ToGetResourceResponse(resource)

	// If the resource is a file, populate the DownloadURL.
	if resp.Type == models.ResourceTypeFile {
		if err := populateDownloadURL(ctx, &resp, resource.File.FileUUID); err != nil {
			return models.GetResourceResponse{}, fmt.Errorf("failed to populate download URL: %w", err)
		}
	}

	return resp, nil
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
