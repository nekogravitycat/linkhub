package s3bucket

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/myconfig"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
)

func toAWSCompletedPart(part models.MultipartCompletePart) types.CompletedPart {
	return types.CompletedPart{
		PartNumber: aws.Int32(part.PartNumber),
		ETag:       aws.String(part.ETag),
	}
}

func getObjectKey(uuid string) (string, error) {
	if err := validator.ValidateUUID(uuid); err != nil {
		return "", fmt.Errorf("invalid UUID: %w", err)
	}
	return "files/" + uuid, nil
}

type S3ObjectStorage struct {
	client     *s3.Client
	bucketName string
}

func NewS3ObjectStorage(client *s3.Client) *S3ObjectStorage {
	return &S3ObjectStorage{
		client:     client,
		bucketName: myconfig.S3_BUCKET_NAME,
	}
}

func (s *S3ObjectStorage) HeadObject(ctx context.Context, uuid string) (*s3.HeadObjectOutput, error) {
	objectKey, err := getObjectKey(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get object key: %w", err)
	}

	return s.client.HeadObject(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(objectKey),
		},
	)
}

func (s *S3ObjectStorage) DeleteObject(ctx context.Context, uuid string) (*s3.DeleteObjectOutput, error) {
	objectKey, err := getObjectKey(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get object key: %w", err)
	}

	return s.client.DeleteObject(
		ctx,
		&s3.DeleteObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(objectKey),
		},
	)
}

func (s *S3ObjectStorage) CreateMultipartUpload(ctx context.Context, uuid string, mime string) (*s3.CreateMultipartUploadOutput, error) {
	objectKey, err := getObjectKey(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get object key: %w", err)
	}

	return s.client.CreateMultipartUpload(
		ctx,
		&s3.CreateMultipartUploadInput{
			Bucket:      aws.String(s.bucketName),
			Key:         aws.String(objectKey),
			ContentType: aws.String(mime),
		},
	)
}

func (s *S3ObjectStorage) CompleteMultipartUpload(ctx context.Context, uuid string, info models.MultipartCompleteInfo) (*s3.CompleteMultipartUploadOutput, error) {
	objectKey, err := getObjectKey(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get object key: %w", err)
	}

	var awsCompletedParts []types.CompletedPart
	for _, p := range info.Parts {
		awsCompletedParts = append(awsCompletedParts, toAWSCompletedPart(p))
	}

	return s.client.CompleteMultipartUpload(
		ctx,
		&s3.CompleteMultipartUploadInput{
			Bucket:   aws.String(s.bucketName),
			Key:      aws.String(objectKey),
			UploadId: aws.String(info.UploadID),
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: awsCompletedParts,
			},
		},
	)
}
