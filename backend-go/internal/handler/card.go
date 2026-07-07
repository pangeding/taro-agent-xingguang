package handler

import (
	"fmt"
	"net/http"

	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

func GETAllCards(svc *service.CardService) gin.HandlerFunc {
	return func(c *gin.Context) {
		cards := svc.GetAllCards()
		c.JSON(http.StatusOK, cards)
	}
}

func GetCard(svc *service.CardService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var id uint
		if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
			return
		}
		card, err := svc.GetCardByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "牌不存在"})
			return
		}
		c.JSON(http.StatusOK, card)
	}
}

func GetRandomCard(svc *service.CardService) gin.HandlerFunc {
	return func(c *gin.Context) {
		card, err := svc.GetRandomCard()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "暂无塔罗牌数据"})
			return
		}
		c.JSON(http.StatusOK, card)
	}
}
