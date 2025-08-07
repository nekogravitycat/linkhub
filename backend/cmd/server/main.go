package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/nekogravitycat/linkhub/backend/internal/database"
	"github.com/nekogravitycat/linkhub/backend/internal/handlers"
	"github.com/nekogravitycat/linkhub/backend/internal/myconfig"
	"github.com/nekogravitycat/linkhub/backend/internal/s3bucket"
	"github.com/nekogravitycat/linkhub/backend/internal/syncer"
)

func main() {
	if err := myconfig.ReadConfigFromEnv(); err != nil {
		log.Fatalf("Error reading configuration: %v", err)
	}

	db := database.GetDBClient()
	if db == nil {
		log.Fatal("Failed to initialize database client")
	}
	defer db.Close()

	s3Client := s3bucket.GetS3Client()
	if s3Client == nil {
		log.Fatal("Failed to initialize S3 client")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go syncer.StartPeriodicSync(ctx, db, s3Client)

	router := gin.Default()
	handlers.RegisterRoutes(router)
	router.Run()
}
