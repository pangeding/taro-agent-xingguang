package handler

import (
	"strconv"

	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

func CreateConversation(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		conv, err := svc.CreateConversation(userID)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, conv)
	}
}

func ListConversations(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		conversations, err := svc.ListConversations(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		type ConvSummary struct {
			ID        uint      `json:"id"`
			Title     string    `json:"title"`
			UpdatedAt string    `json:"updated_at"`
		}

		summaries := make([]ConvSummary, len(conversations))
		for i, conv := range conversations {
			summaries[i] = ConvSummary{
				ID:        conv.ID,
				Title:     conv.Title,
				UpdatedAt: conv.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}

		c.JSON(200, summaries)
	}
}

func GetConversation(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}

		conv, err := svc.GetConversation(uint(id), userID)
		if err != nil {
			c.JSON(404, gin.H{"error": "conversation not found"})
			return
		}
		c.JSON(200, conv)
	}
}

func DeleteConversation(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}

		if err := svc.DeleteConversation(uint(id), userID); err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"success": true})
	}
}

func UpdateConversationTitle(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}

		var req struct {
			Title string `json:"title" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "title is required"})
			return
		}

		if err := svc.UpdateTitle(uint(id), userID, req.Title); err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"success": true})
	}
}

func GetMessages(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}

		limitStr := c.DefaultQuery("limit", "50")
		offsetStr := c.DefaultQuery("offset", "0")
		limit, _ := strconv.Atoi(limitStr)
		offset, _ := strconv.Atoi(offsetStr)

		messages, err := svc.GetMessages(uint(id), userID, limit, offset)
		if err != nil {
			c.JSON(404, gin.H{"error": "conversation not found"})
			return
		}
		c.JSON(200, messages)
	}
}
