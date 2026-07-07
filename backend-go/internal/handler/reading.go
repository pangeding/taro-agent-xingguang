package handler

import (
	"fmt"
	"net/http"

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
		result, err := svc.CreateReading(req.Question, req.SpreadType, req.SessionID)
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
		result, err := svc.CreateReadingLangGraph(req.Question, req.SpreadType, req.SessionID, req.ModelName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func GetReading(svc *service.ReadingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var id uint
		if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
			return
		}
		result, err := svc.GetReadingByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "占卜记录不存在"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
