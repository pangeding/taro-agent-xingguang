package handler

import (
	"errors"
	"net/http"

	"backend-go/internal/auth"
	"backend-go/internal/config"
	"backend-go/internal/db"
	"backend-go/internal/middleware"
	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

// UserDTO 是对外暴露的用户信息。
//
// 用显式 DTO 而不是直接序列化 db.User：即使将来有人给模型加了敏感字段
// 而忘了打 json:"-"，也不会从这里泄露出去。
type UserDTO struct {
	ID                 uint   `json:"id"`
	Username           string `json:"username"`
	DisplayName        string `json:"display_name"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"`
}

func toUserDTO(u *db.User) UserDTO {
	return UserDTO{
		ID:                 u.ID,
		Username:           u.Username,
		DisplayName:        u.DisplayName,
		Role:               u.Role,
		MustChangePassword: u.MustChangePassword,
	}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 登录并下发会话 Cookie。
func Login(svc *service.UserService, cfg *config.Settings) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
			return
		}

		result, err := svc.Login(req.Username, req.Password, c.Request.UserAgent(), c.ClientIP())
		if err != nil {
			status := http.StatusUnauthorized
			switch {
			case errors.Is(err, auth.ErrAccountDisabled):
				status = http.StatusForbidden
			case errors.Is(err, auth.ErrAccountLocked):
				status = http.StatusTooManyRequests
			case errors.Is(err, auth.ErrInvalidCredential):
				status = http.StatusUnauthorized
			default:
				status = http.StatusInternalServerError
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		setSessionCookie(c, cfg, result.Token, int(cfg.SessionTTL().Seconds()))
		c.JSON(http.StatusOK, gin.H{
			"user":       toUserDTO(result.User),
			"expires_at": result.Expires,
		})
	}
}

// Logout 删除当前会话并让 Cookie 立即过期。
func Logout(svc *service.UserService, cfg *config.Settings) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(cfg.SessionCookieName)
		if err := svc.Logout(token); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "登出失败"})
			return
		}
		setSessionCookie(c, cfg, "", -1)
		c.Status(http.StatusNoContent)
	}
}

// Me 返回当前登录用户。
func Me() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middleware.UserFrom(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": toUserDTO(u)})
	}
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改口令。成功后除当前会话外的所有会话失效。
func ChangePassword(svc *service.UserService, cfg *config.Settings) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middleware.UserFrom(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}

		var req changePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "原密码和新密码不能为空"})
			return
		}

		currentToken, _ := c.Cookie(cfg.SessionCookieName)
		err := svc.ChangePassword(u.ID, currentToken, req.OldPassword, req.NewPassword)
		if err != nil {
			status := http.StatusBadRequest
			switch {
			case errors.Is(err, auth.ErrOldPasswordWrong):
				status = http.StatusUnauthorized
			case errors.Is(err, auth.ErrUnauthenticated):
				status = http.StatusUnauthorized
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// setSessionCookie 统一下发/清除会话 Cookie。
//
// 注意：gin 的 SetCookie 不会写 SameSite，必须先单独调 SetSameSite，
// 否则浏览器拿到的 Cookie 没有 SameSite 属性（老 handler/user.go 就有这个缺陷）。
func setSessionCookie(c *gin.Context, cfg *config.Settings, value string, maxAge int) {
	if maxAge < 0 {
		value = ""
	}
	c.SetSameSite(http.SameSiteLaxMode)
	// 参数顺序与 gin 签名一致：name, value, maxAge, path, domain, secure, httpOnly
	c.SetCookie(cfg.SessionCookieName, value, maxAge, "/", "", cfg.SecureCookie(), true)
}
