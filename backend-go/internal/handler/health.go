package handler

import (
	"net/http"

	"backend-go/internal/config"

	"github.com/gin-gonic/gin"
)

func Root(cfg *config.Settings) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "欢迎使用星光塔罗AI助手",
			"version": cfg.Version,
			"docs":    "/docs",
		})
	}
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
