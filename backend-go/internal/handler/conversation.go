package handler

import (
	"strconv"
	"time"

	"backend-go/internal/middleware"
	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

func CreateConversation(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// channel 可选；缺省或非法值都会落到 chat
		var req struct {
			Channel string `json:"channel"`
		}
		_ = c.ShouldBindJSON(&req)

		conv, err := svc.CreateConversation(middleware.ActorFrom(c), req.Channel)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, conv)
	}
}

// ListConversations 支持 ?channel=chat|reading|all，缺省只返回 chat。
func ListConversations(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		conversations, err := svc.ListConversations(middleware.ActorFrom(c), c.Query("channel"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		type ConvSummary struct {
			ID        uint   `json:"id"`
			Title     string `json:"title"`
			Channel   string `json:"channel"`
			UpdatedAt string `json:"updated_at"`
		}

		summaries := make([]ConvSummary, len(conversations))
		for i, conv := range conversations {
			summaries[i] = ConvSummary{
				ID:        conv.ID,
				Title:     conv.Title,
				Channel:   conv.Channel,
				UpdatedAt: conv.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}

		c.JSON(200, summaries)
	}
}

// ConversationDetail 是会话详情的响应体。
// 消息上附带 cards（由 messages.reading_id 反查），前端刷新后据此还原牌面。
type ConversationDetail struct {
	ID        uint                    `json:"id"`
	OwnerID   uint                    `json:"owner_id"`
	Title     string                  `json:"title"`
	Channel   string                  `json:"channel"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
	Messages  []service.MessageDetail `json:"messages"`
}

func GetConversation(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}

		conv, err := svc.GetConversation(uint(id), middleware.ActorFrom(c))
		if err != nil {
			c.JSON(404, gin.H{"error": "conversation not found"})
			return
		}

		c.JSON(200, ConversationDetail{
			ID:        conv.ID,
			OwnerID:   conv.OwnerID,
			Title:     conv.Title,
			Channel:   conv.Channel,
			CreatedAt: conv.CreatedAt,
			UpdatedAt: conv.UpdatedAt,
			Messages:  svc.DecorateMessages(conv.Messages),
		})
	}
}

func DeleteConversation(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}

		if err := svc.DeleteConversation(uint(id), middleware.ActorFrom(c)); err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"success": true})
	}
}

func UpdateConversationTitle(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		if err := svc.UpdateTitle(uint(id), middleware.ActorFrom(c), req.Title); err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"success": true})
	}
}

func GetMessages(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		messages, err := svc.GetMessages(uint(id), middleware.ActorFrom(c), limit, offset)
		if err != nil {
			c.JSON(404, gin.H{"error": "conversation not found"})
			return
		}
		c.JSON(200, svc.DecorateMessages(messages))
	}
}
