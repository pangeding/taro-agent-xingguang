package middleware

import (
	"github.com/gin-gonic/gin"
)

func UserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-Id")
		if userID == "" {
			userID = c.GetHeader("X-User-ID")
		}
		if userID == "" {
			cookie, err := c.Cookie("user_id")
			if err == nil {
				userID = cookie
			}
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

func RequireUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == "" {
			c.JSON(401, gin.H{"error": "X-User-Id header or user_id cookie is required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
