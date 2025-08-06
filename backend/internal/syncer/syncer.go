package syncer

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func StartPeriodicSync(ctx context.Context, db *pgxpool.Pool, s3Client *s3.Client) {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[SYNCER] Syncer stopped.")
			return
		case <-ticker.C:
			log.Println("[SYNCER] Running periodic sync task...")
			if err := checkSync(ctx, db, s3Client); err != nil {
				log.Printf("[SYNCER ERROR] %v\n", err)
			}
		}
	}
}

// checkSync would contain your actual logic to compare DB and R2
func checkSync(ctx context.Context, db *pgxpool.Pool, s3Client *s3.Client) error {
	// TODO: implement your logic here
	return nil
}
