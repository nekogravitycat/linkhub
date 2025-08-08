package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func getPasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func isPasswordCorrect(hash string, password string) (bool, error) {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, fmt.Errorf("failed to compare password hash: %w", err)
	}
	return true, nil
}

func populateDownloadURL(ctx context.Context, resp *models.GetResourceResponse, uuid string) error {
	if resp.Type != models.ResourceTypeFile {
		return fmt.Errorf("cannot populate download URL for non-file resource")
	}
	s3Client := s3bucket.GetS3Client()
	presigner := s3bucket.NewPresigner(s3Client, 1*time.Minute)

	url, err := presigner.Get(ctx, uuid)
	if err != nil {
		return fmt.Errorf("failed to generate download URL: %w", err)
	}
	resp.File.DownloadURL = url
	return nil
}

// partSize defines the maximum size of a single part in a multipart upload.
// It is also the threshold for deciding between single and multipart uploads.
const partSize = 50 * 1024 * 1024 // 50MB

// Create the upload response based on the file size.
// It selects single or multipart upload, initializes the upload session if needed,
// and returns the appropriate pre-signed URL(s) for uploading to S3.
func generateUploadResponse(ctx context.Context, request models.CreateFileRequest, fileUUID string) (models.UploadFileResponse, error) {
	resp := models.UploadFileResponse{
		FileUUID: fileUUID,
	}

	// Decide upload type based on the request size
	if request.Size <= partSize {
		// Single file upload
		resp.Type = models.UploadTypeSingle
		info, err := createSingleUpload(ctx, fileUUID, request.MIMEType)
		if err != nil {
			return models.UploadFileResponse{}, fmt.Errorf("failed to create single upload: %w", err)
		}
		resp.Single = &info
	} else {
		// Multipart upload, 50MB per part
		resp.Type = models.UploadTypeMultipart
		info, err := createMultipartUpload(ctx, fileUUID, request.MIMEType, request.Size)
		if err != nil {
			return models.UploadFileResponse{}, fmt.Errorf("failed to create multipart upload: %w", err)
		}
		resp.Multipart = &info
	}

	return resp, nil
}

func createSingleUpload(ctx context.Context, uuid string, mime string) (models.SingleUploadInfo, error) {
	s3Client := s3bucket.GetS3Client()
	presigner := s3bucket.NewPresigner(s3Client, 30*time.Minute)

	// Generate a pre-signed URL for the single upload
	uploadURL, err := presigner.Put(ctx, uuid, mime)
	if err != nil {
		return models.SingleUploadInfo{}, fmt.Errorf("failed to get upload URL: %w", err)
	}

	// Return the single upload info
	return models.SingleUploadInfo{
		UploadURL: uploadURL,
	}, nil
}

func createMultipartUpload(ctx context.Context, uuid string, mime string, size int64) (models.MultipartUploadInfo, error) {
	s3Client := s3bucket.GetS3Client()
	objectStorage := s3bucket.NewS3ObjectStorage(s3Client)
	presigner := s3bucket.NewPresigner(s3Client, 30*time.Minute)

	// Create multipart upload session
	createResp, err := objectStorage.CreateMultipartUpload(ctx, uuid, mime)
	if err != nil || createResp.UploadId == nil {
		return models.MultipartUploadInfo{}, fmt.Errorf("failed to create multipart upload: %w", err)
	}

	// Generate presigned URLs for each part
	numParts := int32((size + partSize - 1) / partSize) // Round up to nearest part size
	partURLs, err := presigner.UploadPart(ctx, uuid, *createResp.UploadId, numParts)
	if err != nil {
		return models.MultipartUploadInfo{}, fmt.Errorf("failed to generate multipart upload URLs: %w", err)
	}

	// Build multipart upload part
	parts := []models.MultipartUploadPart{}
	for i, url := range partURLs {
		parts = append(parts, models.MultipartUploadPart{
			PartNumber: int32(i + 1), // 1-based index
			UploadURL:  url,
		})
	}

	// Build and return multipart upload info
	return models.MultipartUploadInfo{
		UploadID: *createResp.UploadId,
		Parts:    parts,
	}, nil
}

func completeMultipartUpload(ctx context.Context, uuid string, info models.MultipartCompleteInfo) error {
	s3Client := s3bucket.GetS3Client()
	objectStorage := s3bucket.NewS3ObjectStorage(s3Client)
	if _, err := objectStorage.CompleteMultipartUpload(ctx, uuid, info); err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}
	return nil
}

var ErrS3NotFound = fmt.Errorf("file not found in S3")

func headS3File(ctx context.Context, file models.File) (models.S3HeadResponse, error) {
	s3Client := s3bucket.GetS3Client()
	objectStorage := s3bucket.NewS3ObjectStorage(s3Client)

	head, err := objectStorage.HeadObject(ctx, file.FileUUID)
	if err != nil || head == nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return models.S3HeadResponse{}, ErrS3NotFound
		}
		return models.S3HeadResponse{}, fmt.Errorf("failed to head S3 file: %w", err)
	}

	if head.ContentType == nil || head.ContentLength == nil {
		return models.S3HeadResponse{}, fmt.Errorf("missing required S3 head response fields")
	}

	return models.S3HeadResponse{
		MIMEType: *head.ContentType,
		Size:     *head.ContentLength,
	}, nil
}
