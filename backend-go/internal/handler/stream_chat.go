package handler

import (
	"strconv"

	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

func StreamChatMessage(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}

		var req struct {
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "content is required"})
			return
		}

		sse := NewSSEWriter(c.Writer)

		if err := svc.StreamChat(c.Request.Context(), uint(id), userID, req.Content, func(event string, data string) {
			sse.WriteEvent(event, data)
		}); err != nil {
			sse.WriteEvent("error", `{"error": "`+err.Error()+`"}`)
			return
		}
	}
}

func StreamTarotReading(svc *service.ConversationService, readingSvc *service.ReadingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}

		var req struct {
			Question   string `json:"question" binding:"required"`
			SpreadType string `json:"spread_type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "question is required"})
			return
		}

		sse := NewSSEWriter(c.Writer)

		if err := svc.StreamTarotReading(c.Request.Context(), uint(id), userID, req.Question, req.SpreadType, func(event string, data string) {
			sse.WriteEvent(event, data)
		}, readingSvc); err != nil {
			sse.WriteEvent("error", `{"error": "`+err.Error()+`"}`)
			return
		}
	}
}
