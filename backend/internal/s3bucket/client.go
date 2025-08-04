package s3bucket

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// singleton S3 client
var (
	_s3Client     *s3.Client
	_onceS3Client sync.Once
)

func GetS3Client() *s3.Client {
	_onceS3Client.Do(func() {
		// Initialize the S3 client with credentials
		var baseEndpoint = os.Getenv("S3_BASE_ENDPOINT")
		var accessKeyId = os.Getenv("S3_ACCESS_KEY_ID")
		var accessKeySecret = os.Getenv("S3_ACCESS_KEY_SECRET")

		cfg, err := config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion("auto"),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, ""),
			),
		)
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		_s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(baseEndpoint)
		})

		if _s3Client == nil {
			log.Fatal("Failed to create S3 client")
		}
	})

	return _s3Client
}
