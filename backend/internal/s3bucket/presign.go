package s3bucket

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Presigner struct {
	client      *s3.PresignClient
	urlLifetime time.Duration
}

// NewPresigner constructs a Presigner using an s3.Client
func NewPresigner(s3Client *s3.Client, urlLifetime time.Duration) *Presigner {
	return &Presigner{
		client:      s3.NewPresignClient(s3Client),
		urlLifetime: urlLifetime,
	}
}

// getBucketAndKey returns the bucket name and object key
func getBucketAndKey(uuid string) (string, string) {
	bucket, ok := os.LookupEnv("S3_BUCKET_NAME")
	if !ok || bucket == "" {
		log.Fatal("S3_BUCKET_NAME environment variable is not set")
	}
	key := fmt.Sprintf("files/%s", uuid)
	return bucket, key
}

// Head generates a HEAD presign URL
func (p *Presigner) Head(ctx context.Context, uuid string) (string, error) {
	bucket, key := getBucketAndKey(uuid)

	output, err := p.client.PresignHeadObject(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
		func(opts *s3.PresignOptions) {
			opts.Expires = p.urlLifetime
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to presign HEAD object: %w", err)
	}
	return output.URL, nil
}

// Get generates a GET presign URL
func (p *Presigner) Get(ctx context.Context, uuid string) (string, error) {
	bucket, key := getBucketAndKey(uuid)

	output, err := p.client.PresignGetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
		func(opts *s3.PresignOptions) {
			opts.Expires = p.urlLifetime
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to presign GET object: %w", err)
	}
	return output.URL, nil
}

// Put generates a PUT presign URL
func (p *Presigner) Put(ctx context.Context, uuid string, mime string) (string, error) {
	bucket, key := getBucketAndKey(uuid)

	output, err := p.client.PresignPutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			ContentType: aws.String(mime),
		},
		func(opts *s3.PresignOptions) {
			opts.Expires = p.urlLifetime
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to presign PUT object: %w", err)
	}
	return output.URL, nil
}

// UploadPart generates presign URLs for each part of multipart upload
func (p *Presigner) UploadPart(ctx context.Context, uuid string, uploadID string, parts int32) ([]string, error) {
	if parts < 2 {
		return nil, fmt.Errorf("number of parts must be at least 2")
	}

	bucket, key := getBucketAndKey(uuid)
	urls := make([]string, 0, parts)

	for idx := range parts {
		output, err := p.client.PresignUploadPart(
			ctx,
			&s3.UploadPartInput{
				Bucket:     aws.String(bucket),
				Key:        aws.String(key),
				PartNumber: aws.Int32(idx + 1),
				UploadId:   aws.String(uploadID),
			},
			func(opts *s3.PresignOptions) {
				opts.Expires = p.urlLifetime
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to presign multipart upload (part %d): %w", idx+1, err)
		}
		urls = append(urls, output.URL)
	}

	return urls, nil
}
