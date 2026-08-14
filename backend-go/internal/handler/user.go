package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func InitUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		existingUserID := c.GetString("user_id")
		if existingUserID != "" {
			c.JSON(200, gin.H{"user_id": existingUserID})
			return
		}

		newUserID := uuid.New().String()
		c.SetCookie("user_id", newUserID, 60*60*24*365, "/", "", false, true)
		c.JSON(200, gin.H{"user_id": newUserID})
	}
}
