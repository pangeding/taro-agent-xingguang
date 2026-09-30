package middleware

import (
	"errors"
	"net/http"
	"strings"

	"backend-go/internal/auth"
	"backend-go/internal/config"
	"backend-go/internal/db"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserKey  = "auth_user"
	ctxActorKey = "auth_actor"
)

// SessionResolver 是 middleware 依赖的最小接口。
//
// 刻意不直接依赖 service.UserService：避免 middleware 与 service 耦合成环，
// 也让中间件可以在测试里注入假实现。
type SessionResolver interface {
	ResolveSession(token string) (*db.User, *db.Session, error)
}

// mustChangePasswordAllowlist 是强制改密状态下仍可访问的路径后缀。
// 必须包含这三条，否则会造成死锁：用户改不了密码，而所有业务接口都拒绝他。
var mustChangePasswordAllowlist = []string{
	"/auth/password",
	"/auth/logout",
	"/auth/me",
}

// Auth 解析会话 Cookie 并把调用者写入 context。
//
//	未登录 / 会话过期 / 会话无效 -> 401
//	账号被禁用                    -> 403 账号已禁用
//	prod 下需强制改密             -> 403 password_change_required（白名单除外）
func Auth(cfg *config.Settings, resolver SessionResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cfg.SessionCookieName)
		if err != nil || token == "" {
			unauthorized(c)
			return
		}

		user, _, err := resolver.ResolveSession(token)
		if err != nil {
			if errors.Is(err, auth.ErrAccountDisabled) {
				c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
				c.Abort()
				return
			}
			unauthorized(c)
			return
		}

		// L3 强制改密门禁（仅 prod）。
		// 不用「启动时拒绝启动」是因为那会死锁：库里有弱口令就起不来，也就改不了密码。
		if cfg.IsProd() && user.MustChangePassword && !isAllowedDuringPasswordChange(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "password_change_required"})
			c.Abort()
			return
		}

		c.Set(ctxUserKey, user)
		c.Set(ctxActorKey, buildActor(c, user))
		c.Next()
	}
}

// buildActor 组装调用者。
//
// scope=all 仅在「管理员 + 显式传参」时生效；非管理员传了也静默忽略，
// 不返回错误——否则该参数会变成角色探测器。
func buildActor(c *gin.Context, user *db.User) auth.Actor {
	return auth.Actor{
		UserID: user.ID,
		Role:   user.Role,
		All:    c.Query("scope") == "all" && user.Role == auth.RoleAdmin,
	}
}

// isAllowedDuringPasswordChange 判断当前请求是否属于改密白名单。
func isAllowedDuringPasswordChange(c *gin.Context) bool {
	p := c.Request.URL.Path
	for _, suffix := range mustChangePasswordAllowlist {
		if strings.HasSuffix(p, suffix) {
			return true
		}
	}
	return false
}

// ActorFrom 取出当前调用者。必须在 Auth 之后调用。
func ActorFrom(c *gin.Context) auth.Actor {
	if v, ok := c.Get(ctxActorKey); ok {
		if a, ok := v.(auth.Actor); ok {
			return a
		}
	}
	return auth.Actor{}
}

// UserFrom 取出当前用户。必须在 Auth 之后调用。
func UserFrom(c *gin.Context) *db.User {
	if v, ok := c.Get(ctxUserKey); ok {
		if u, ok := v.(*db.User); ok {
			return u
		}
	}
	return nil
}

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
	c.Abort()
}
