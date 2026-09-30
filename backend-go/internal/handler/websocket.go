package handler

import (
	"encoding/json"
	"net/http"
	"slices"

	"backend-go/internal/config"
	"backend-go/internal/middleware"
	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketReading 提供 CLI / 调试用的占卜通道（见 doc/2026-05-29-websocket-cli-implementation.md）。
//
// 本路由挂在已带 Auth 中间件的 v1 组下，因此连接建立前已完成会话校验。
//
// 加固点（原实现存在跨站 WebSocket 劫持风险）：
//   - CheckOrigin 由恒真的 `return true` 改为按 BACKEND_CORS_ORIGINS 白名单校验。
//     Cookie 会被浏览器自动带进 WebSocket 握手，若放开 Origin，
//     任意站点都能用受害者的 Cookie 建立连接并读取占卜结果。
//   - 归属取自服务端会话（actor），不再信任客户端传入的 session_id。
func WebSocketReading(svc *service.ReadingService, cfg *config.Settings) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 未认证直接拒绝，不升级协议。
		actor := middleware.ActorFrom(c)
		if actor.UserID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: originChecker(cfg),
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var sessionID string
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var req struct {
				Question   string  `json:"question"`
				SpreadType string  `json:"spread_type"`
				ModelName  *string `json:"model_name"`
				SessionID  *string `json:"session_id"`
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				conn.WriteJSON(map[string]string{"error": "invalid request", "status": "error"})
				continue
			}

			sid := req.SessionID
			if sid == nil || *sid == "" {
				if sessionID != "" {
					sid = &sessionID
				}
			}

			// 归属由服务端会话决定，与客户端传入的 session_id 无关。
			result, err := svc.CreateReadingLangGraph(req.Question, req.SpreadType, sid, req.ModelName, actor.UserID, nil)
			if err != nil {
				conn.WriteJSON(map[string]string{"error": err.Error(), "status": "error"})
			} else {
				sessionID = result.SessionID
				conn.WriteJSON(result)
			}
		}
	}
}

// originChecker 返回按白名单校验 Origin 的函数。
//
// Origin 为空时放行：非浏览器客户端（curl、CLI 工具）不会发 Origin 头，
// 而它们本来也不受 CSWSH 影响——CSWSH 依赖浏览器自动携带 Cookie。
func originChecker(cfg *config.Settings) func(*http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return slices.Contains(cfg.BackendCORSOrigins, origin)
	}
}
