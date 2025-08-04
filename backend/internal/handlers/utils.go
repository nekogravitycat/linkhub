package handlers

import (
	"context"
	"fmt"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/s3bucket"
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
