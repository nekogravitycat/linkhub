package myconfig

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

var (
	S3_BASE_ENDPOINT     string
	S3_ACCESS_KEY_ID     string
	S3_ACCESS_KEY_SECRET string
	S3_BUCKET_NAME       string
	DATABASE_URL         string
)

// ReadConfigFromEnv reads configuration from environment variables.
// It returns an error if any required variable is not set or empty.
func ReadConfigFromEnv() error {
	if err := godotenv.Load(); err != nil {
		return err
	}
	var ok bool
	if S3_BASE_ENDPOINT, ok = os.LookupEnv("S3_BASE_ENDPOINT"); !ok || S3_BASE_ENDPOINT == "" {
		return errors.New("S3_BASE_ENDPOINT is not set or empty")
	}
	if S3_ACCESS_KEY_ID, ok = os.LookupEnv("S3_ACCESS_KEY_ID"); !ok || S3_ACCESS_KEY_ID == "" {
		return errors.New("S3_ACCESS_KEY_ID is not set or empty")
	}
	if S3_ACCESS_KEY_SECRET, ok = os.LookupEnv("S3_ACCESS_KEY_SECRET"); !ok || S3_ACCESS_KEY_SECRET == "" {
		return errors.New("S3_ACCESS_KEY_SECRET is not set or empty")
	}
	if S3_BUCKET_NAME, ok = os.LookupEnv("S3_BUCKET_NAME"); !ok || S3_BUCKET_NAME == "" {
		return errors.New("S3_BUCKET_NAME is not set or empty")
	}
	if DATABASE_URL, ok = os.LookupEnv("DATABASE_URL"); !ok || DATABASE_URL == "" {
		return errors.New("DATABASE_URL is not set or empty")
	}
	return nil
}
