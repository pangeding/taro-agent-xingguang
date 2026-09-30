package handler

import (
	"fmt"
	"net/http"

	"backend-go/internal/middleware"
	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

type ReadingRequest struct {
	Question   string  `json:"question" binding:"required"`
	SpreadType string  `json:"spread_type"`
	SessionID  *string `json:"session_id"`
	ModelName  *string `json:"model_name"`
}

func CreateReading(svc *service.ReadingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ReadingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.SpreadType == "" {
			req.SpreadType = "single"
		}
		actor := middleware.ActorFrom(c)
		result, err := svc.CreateReading(req.Question, req.SpreadType, req.SessionID, actor.UserID, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func CreateReadingLangGraph(svc *service.ReadingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ReadingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.SpreadType == "" {
			req.SpreadType = "single"
		}
		actor := middleware.ActorFrom(c)
		result, err := svc.CreateReadingLangGraph(req.Question, req.SpreadType, req.SessionID, req.ModelName, actor.UserID, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetReading 读取占卜详情。
//
// 该接口此前完全不校验归属（IDOR），前端占卜历史点开就走这条路径。
// 现在非本人记录一律 404：不用 403，否则「ID 存在但不属于你」会暴露存在性。
func GetReading(svc *service.ReadingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var id uint
		if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
			return
		}
		result, err := svc.GetReadingByID(id, middleware.ActorFrom(c))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "占卜记录不存在"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
