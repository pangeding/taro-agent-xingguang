package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 配置跨域。
//
// 已移除 X-User-Id / X-User-ID：身份改为服务端会话 Cookie 后，
// 这两个头不再有任何作用，而把它们留在 AllowHeaders 里
// 会让人误以为「伪造该头即可切换身份」仍然可行。
//
// AllowCredentials 保持 true：前端以同源 rewrite 代理访问时为 same-origin，
// 但直连后端（NEXT_PUBLIC_API_URL 指向 8000）时需要带上凭证。
func CORS(origins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	})
}
