package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nekogravitycat/linkhub/backend/internal/database"
	"github.com/nekogravitycat/linkhub/backend/internal/handlers"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db := database.GetDBClient()
	if db == nil {
		log.Fatal("Failed to initialize database client")
	}
	defer db.Close()

	router := gin.Default()
	handlers.RegisterRoutes(router)
	router.Run()
}
