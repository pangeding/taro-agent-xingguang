package handler

import (
	"encoding/json"
	"net/http"

	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func WebSocketReading(svc *service.ReadingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
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

			result, err := svc.CreateReadingLangGraph(req.Question, req.SpreadType, sid, req.ModelName)
			if err != nil {
				conn.WriteJSON(map[string]string{"error": err.Error(), "status": "error"})
			} else {
				sessionID = result.SessionID
				conn.WriteJSON(result)
			}
		}
	}
}
