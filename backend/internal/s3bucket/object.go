package s3bucket

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/nekogravitycat/linkhub/backend/internal/models"
)

func toAWSCompletedPart(part models.MultipartCompletePart) types.CompletedPart {
	return types.CompletedPart{
		PartNumber: aws.Int32(part.PartNumber),
		ETag:       aws.String(part.ETag),
	}
}

type S3ObjectStorage struct {
	client     *s3.Client
	bucketName string
}

func NewS3ObjectStorage(client *s3.Client) *S3ObjectStorage {
	bucket, ok := os.LookupEnv("S3_BUCKET_NAME")
	if !ok || bucket == "" {
		log.Fatal("S3_BUCKET_NAME environment variable is not set")
	}
	return &S3ObjectStorage{
		client:     client,
		bucketName: bucket,
	}
}

func (s *S3ObjectStorage) HeadObject(ctx context.Context, uuid string) (*s3.HeadObjectOutput, error) {
	objectKey := "files/" + uuid

	return s.client.HeadObject(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(objectKey),
		},
	)
}

func (s *S3ObjectStorage) DeleteObject(ctx context.Context, uuid string) (*s3.DeleteObjectOutput, error) {
	objectKey := "files/" + uuid

	return s.client.DeleteObject(
		ctx,
		&s3.DeleteObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(objectKey),
		},
	)
}

func (s *S3ObjectStorage) CreateMultipartUpload(ctx context.Context, uuid string, mime string) (*s3.CreateMultipartUploadOutput, error) {
	objectKey := "files/" + uuid

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
	objectKey := "files/" + uuid

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
