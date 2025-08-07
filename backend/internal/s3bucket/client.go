package s3bucket

import (
	"context"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nekogravitycat/linkhub/backend/internal/myconfig"
)

// singleton S3 client
var (
	_s3Client     *s3.Client
	_onceS3Client sync.Once
)

// Initialize and returns a singleton S3 client.
// It reads configuration from environment variables and ensures the client is created only once.
func GetS3Client() *s3.Client {
	_onceS3Client.Do(func() {
		cfg, err := config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion("auto"),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(myconfig.S3_ACCESS_KEY_ID, myconfig.S3_ACCESS_KEY_SECRET, ""),
			),
		)
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		_s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(myconfig.S3_BASE_ENDPOINT)
		})

		if _s3Client == nil {
			log.Fatal("Failed to create S3 client")
		}
	})

	return _s3Client
}
