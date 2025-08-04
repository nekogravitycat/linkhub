package s3bucket

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type CompletedPart struct {
	ETag       string
	PartNumber int32
}

func (p CompletedPart) ToAWSType() types.CompletedPart {
	return types.CompletedPart{
		ETag:       aws.String(p.ETag),
		PartNumber: aws.Int32(p.PartNumber),
	}
}

type S3ObjectStorage struct {
	client     *s3.Client
	bucketName string
}

func NewS3ObjectStorage(client *s3.Client) *S3ObjectStorage {
	bucket := os.Getenv("S3_BUCKET_NAME")
	return &S3ObjectStorage{
		client:     client,
		bucketName: bucket,
	}
}

func (s *S3ObjectStorage) HeadObject(uuid string) (*s3.HeadObjectOutput, error) {
	objectKey := "files/" + uuid

	return s.client.HeadObject(
		context.TODO(),
		&s3.HeadObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(objectKey),
		},
	)
}

func (s *S3ObjectStorage) DeleteObject(uuid string) (*s3.DeleteObjectOutput, error) {
	objectKey := "files/" + uuid

	return s.client.DeleteObject(
		context.TODO(),
		&s3.DeleteObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(objectKey),
		},
	)
}

func (s *S3ObjectStorage) CreateMultipartUpload(uuid string, mime string) (*s3.CreateMultipartUploadOutput, error) {
	objectKey := "files/" + uuid

	return s.client.CreateMultipartUpload(
		context.TODO(),
		&s3.CreateMultipartUploadInput{
			Bucket:      aws.String(s.bucketName),
			Key:         aws.String(objectKey),
			ContentType: aws.String(mime),
		},
	)
}

func (s *S3ObjectStorage) CompleteMultipartUpload(uuid string, uploadID string, parts []CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {
	objectKey := "files/" + uuid

	var awsCompletedParts []types.CompletedPart
	for _, p := range parts {
		awsCompletedParts = append(awsCompletedParts, p.ToAWSType())
	}

	return s.client.CompleteMultipartUpload(
		context.TODO(),
		&s3.CompleteMultipartUploadInput{
			Bucket:   aws.String(s.bucketName),
			Key:      aws.String(objectKey),
			UploadId: aws.String(uploadID),
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: awsCompletedParts,
			},
		},
	)
}
