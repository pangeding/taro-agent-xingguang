package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend-go/internal/middleware"
	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

// toJSONString 把任意字符串编码成 JSON 字符串字面量（含引号与转义）。
//
// SSE 的 error 事件是手拼 JSON 的，若直接把 err.Error() 塞进 "..." 里，
// 一旦错误文案含引号/反斜杠/换行就会产生非法 JSON，前端 JSON.parse 失败后
// 只会看到一个语焉不详的 "请求失败"。这里统一走 encoding/json 转义。
func toJSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `"内部错误"`
	}
	return string(b)
}

func StreamChatMessage(svc *service.ConversationService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// 流前归属预检：SSE 一旦开始写，HTTP 状态就固定为 200，
		// 越权就只能退化成流内的 error 事件，任何只看状态码的客户端都会误判成功。
		actor := middleware.ActorFrom(c)
		if err := svc.AssertAccessible(uint(id), actor); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
			return
		}

		sse := NewSSEWriter(c.Writer)

		if err := svc.StreamChat(c.Request.Context(), uint(id), actor, req.Content, func(event string, data string) {
			sse.WriteEvent(event, data)
		}); err != nil {
			sse.WriteEvent("error", `{"error": `+toJSONString(err.Error())+`}`)
			return
		}
	}
}

func StreamTarotReading(svc *service.ConversationService, readingSvc *service.ReadingService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// 流前归属预检，理由同 StreamChatMessage。
		actor := middleware.ActorFrom(c)
		if err := svc.AssertAccessible(uint(id), actor); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
			return
		}

		sse := NewSSEWriter(c.Writer)

		if err := svc.StreamTarotReading(c.Request.Context(), uint(id), actor, req.Question, req.SpreadType, func(event string, data string) {
			sse.WriteEvent(event, data)
		}, readingSvc); err != nil {
			sse.WriteEvent("error", `{"error": `+toJSONString(err.Error())+`}`)
			return
		}
	}
}
