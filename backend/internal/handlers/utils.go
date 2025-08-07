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

func isExpired(resource models.Resource, now time.Time) bool {
	if resource.Entry.ExpiresAt == nil {
		return false
	}
	return resource.Entry.ExpiresAt.UTC().Before(now.UTC())
}

func calculatePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
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

func populateDownloadURL(ctx context.Context, resp *models.GetResourceResponse, uuid string) error {
	if resp.Type != models.ResourceTypeFile {
		return fmt.Errorf("cannot populate download URL for non-file resource")
	}
	s3Client := s3bucket.GetS3Client()
	presigner := s3bucket.NewPresigner(s3Client, 1*time.Minute)
	var err error
	resp.File.DownloadURL, err = presigner.Get(ctx, uuid)
	if err != nil {
		return fmt.Errorf("failed to generate download URL: %w", err)
	}
	return nil
}

// Create the upload response based on the file size.
// It selects single or multipart upload, initializes the upload session if needed,
// and returns the appropriate pre-signed URL(s) for uploading to S3.
func GenerateUploadResponse(ctx context.Context, request models.CreateFileRequest, fileUUID string) (models.UploadFileResponse, error) {
	resp := models.UploadFileResponse{FileUUID: fileUUID}
	// Create the S3 client and presigner
	s3Client := s3bucket.GetS3Client()
	presigner := s3bucket.NewPresigner(s3Client, 30*time.Minute)

	// Decide upload type based on the request size
	const partSize = 50 * 1024 * 1024 // 50MB

	if request.Size <= partSize {
		// Single file upload
		resp.Type = models.UploadTypeSingle

		// Generate a pre-signed URL for the single upload
		uploadURL, err := presigner.Put(ctx, fileUUID, request.MIMEType)
		if err != nil {
			return models.UploadFileResponse{}, fmt.Errorf("failed to get upload URL: %w", err)
		}

		// Add the single upload config to the response
		resp.Single = &models.SingleUploadConfig{
			UploadURL: uploadURL,
		}

	} else {
		// Multipart upload, 50MB per part
		resp.Type = models.UploadTypeMultipart

		// Create multipart upload session
		objectStorage := s3bucket.NewS3ObjectStorage(s3Client)
		createResp, err := objectStorage.CreateMultipartUpload(ctx, fileUUID, request.MIMEType)
		if err != nil || createResp.UploadId == nil {
			return models.UploadFileResponse{}, fmt.Errorf("failed to create multipart upload: %w", err)
		}

		// Generate presigned URLs for each part
		numParts := int((request.Size + partSize - 1) / partSize) // Round up to nearest part size
		partURLs, err := presigner.UploadPart(ctx, fileUUID, *createResp.UploadId, numParts)
		if err != nil {
			return models.UploadFileResponse{}, fmt.Errorf("failed to generate multipart upload URLs: %w", err)
		}

		// Build multipart upload response
		parts := []models.MultipartPart{}
		for i, url := range partURLs {
			parts = append(parts, models.MultipartPart{
				PartNumber: i + 1, // 1-based index
				UploadURL:  url,
			})
		}

		// Add multipart upload config to the response
		resp.Multipart = &models.MultipartUploadConfig{
			UploadID: *createResp.UploadId,
			Parts:    parts,
		}
	}
	return resp, nil
}
