package handlers

import (
	"log"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func respondWithError(c *gin.Context, status int, publicMessage string, internalErr error) {
	if internalErr != nil {
		log.Printf("[REQUEST ERROR] %s: %s", publicMessage, internalErr.Error())
	}
	c.JSON(status, ErrorResponse{Message: publicMessage})
}
