package service

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"backend-go/internal/auth"
	"backend-go/internal/config"
	"backend-go/internal/db"

	"gorm.io/gorm"
)

// 登录失败策略：连续 5 次失败锁定 15 分钟。
const (
	maxFailedAttempts = 5
	lockDuration      = 15 * time.Minute
)

// 哨兵错误统一来自 auth 包，这样 middleware 的 errors.Is 能正确匹配
// （两处各声明一份 var 会让比对永远不成立）。
var (
	ErrInvalidCredential = auth.ErrInvalidCredential
	ErrAccountDisabled   = auth.ErrAccountDisabled
	ErrAccountLocked     = auth.ErrAccountLocked
	ErrOldPasswordWrong  = auth.ErrOldPasswordWrong
	ErrUnauthenticated   = auth.ErrUnauthenticated
)

// LoginResult 是登录成功的返回值。
type LoginResult struct {
	User    *db.User
	Token   string
	Expires time.Time
}

type UserService struct {
	DB  *gorm.DB
	Cfg *config.Settings
}

func NewUserService(db *gorm.DB, cfg *config.Settings) *UserService {
	return &UserService{DB: db, Cfg: cfg}
}

// Login 校验口令并创建会话。
//
// 关键点：用户不存在时也执行一次等价的 bcrypt 比对（auth.VerifyDummy），
// 否则「用户不存在」会瞬间返回而「口令错误」要 ~213ms，
// 登录接口就成了用户名枚举预言机。
func (s *UserService) Login(username, password, userAgent, ip string) (*LoginResult, error) {
	uname := auth.NormalizeUsername(username)

	var u db.User
	err := s.DB.Where("username = ?", uname).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			auth.VerifyDummy(password)
			return nil, ErrInvalidCredential
		}
		return nil, err
	}

	if u.Status != 1 {
		return nil, ErrAccountDisabled
	}
	if u.LockedUntil != nil && u.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("%w（剩余 %d 分钟）",
			ErrAccountLocked, int(time.Until(*u.LockedUntil).Minutes())+1)
	}

	if err := auth.VerifyPassword(u.PasswordHash, password); err != nil {
		s.registerFailure(&u)
		return nil, ErrInvalidCredential
	}

	now := time.Now()
	s.DB.Model(&db.User{}).Where("id = ?", u.ID).Updates(map[string]any{
		"failed_attempts": 0,
		"locked_until":    nil,
		"last_login_at":   now,
	})
	u.FailedAttempts = 0
	u.LockedUntil = nil
	u.LastLoginAt = &now

	token, session, err := s.createSession(u.ID, userAgent, ip)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: &u, Token: token, Expires: session.ExpiresAt}, nil
}

// registerFailure 累加失败次数，达到阈值则锁定。
func (s *UserService) registerFailure(u *db.User) {
	attempts := u.FailedAttempts + 1
	updates := map[string]any{"failed_attempts": attempts}
	if attempts >= maxFailedAttempts {
		lock := time.Now().Add(lockDuration)
		updates["locked_until"] = lock
		updates["failed_attempts"] = 0 // 锁定后重置计数，锁定到期即可重新尝试
		log.Printf("[auth] 账号 %q 连续失败 %d 次，锁定至 %s", u.Username, attempts, lock.Format(time.RFC3339))
	}
	s.DB.Model(&db.User{}).Where("id = ?", u.ID).Updates(updates)
}

// createSession 生成 token 并落库（只存 sha256）。
func (s *UserService) createSession(userID uint, userAgent, ip string) (string, *db.Session, error) {
	token, err := auth.NewToken()
	if err != nil {
		return "", nil, fmt.Errorf("生成会话 token 失败: %w", err)
	}

	sess := db.Session{
		TokenHash: auth.HashToken(token),
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.Cfg.SessionTTL()),
		UserAgent: truncate(userAgent, 255),
		IP:        truncate(ip, 64),
	}
	if err := s.DB.Create(&sess).Error; err != nil {
		return "", nil, fmt.Errorf("写入会话失败: %w", err)
	}
	return token, &sess, nil
}

// ResolveSession 由明文 token 查出调用者。
//
// 每次请求都读一次 users：这样禁用账号、强制改密、改角色都能立即生效，
// 不需要额外的缓存失效机制（见技术文档 §3.1）。
func (s *UserService) ResolveSession(token string) (*db.User, *db.Session, error) {
	if token == "" {
		return nil, nil, ErrUnauthenticated
	}

	var sess db.Session
	err := s.DB.Where("token_hash = ?", auth.HashToken(token)).First(&sess).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrUnauthenticated
		}
		return nil, nil, err
	}
	if sess.ExpiresAt.Before(time.Now()) {
		s.DB.Delete(&db.Session{}, sess.ID)
		return nil, nil, ErrUnauthenticated
	}

	var u db.User
	if err := s.DB.First(&u, sess.UserID).Error; err != nil {
		return nil, nil, ErrUnauthenticated
	}
	if u.Status != 1 {
		s.DB.Where("user_id = ?", u.ID).Delete(&db.Session{})
		return nil, nil, ErrAccountDisabled
	}
	return &u, &sess, nil
}

// Logout 删除当前会话。
func (s *UserService) Logout(token string) error {
	if token == "" {
		return nil
	}
	return s.DB.Where("token_hash = ?", auth.HashToken(token)).Delete(&db.Session{}).Error
}

// ChangePassword 修改口令。
//
// 成功后删除该用户除当前会话外的全部会话 —— 这是「改密踢掉其他设备」的语义，
// 也是口令泄露后的止损手段。currentToken 为空时删除全部会话。
func (s *UserService) ChangePassword(userID uint, currentToken, oldPassword, newPassword string) error {
	var u db.User
	if err := s.DB.First(&u, userID).Error; err != nil {
		return ErrUnauthenticated
	}
	if err := auth.VerifyPassword(u.PasswordHash, oldPassword); err != nil {
		return ErrOldPasswordWrong
	}
	if err := auth.PasswordStrengthError(newPassword, false); err != nil {
		return err
	}
	if newPassword == oldPassword {
		return errors.New("新密码不能与原密码相同")
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("生成口令哈希失败: %w", err)
	}
	if err := s.DB.Model(&db.User{}).Where("id = ?", u.ID).Updates(map[string]any{
		"password_hash":        hash,
		"must_change_password": false,
	}).Error; err != nil {
		return err
	}

	q := s.DB.Where("user_id = ?", u.ID)
	if currentToken != "" {
		q = q.Where("token_hash <> ?", auth.HashToken(currentToken))
	}
	return q.Delete(&db.Session{}).Error
}

// CleanupExpiredSessions 清理过期会话，启动时调用一次。
func (s *UserService) CleanupExpiredSessions() {
	res := s.DB.Where("expires_at < ?", time.Now()).Delete(&db.Session{})
	if res.Error == nil && res.RowsAffected > 0 {
		log.Printf("[auth] 清理过期会话 %d 条", res.RowsAffected)
	}
}

// —————————————————— 管理员操作（供 admin_user CLI 与后续管理界面复用） ——————————————————

// CreateUser 创建账号。
func (s *UserService) CreateUser(username, password, role, displayName string, mustChange bool) (*db.User, error) {
	uname := auth.NormalizeUsername(username)
	if err := auth.ValidateUsername(uname); err != nil {
		return nil, err
	}
	if role != auth.RoleAdmin {
		role = auth.RoleUser
	}
	if err := auth.PasswordStrengthError(password, false); err != nil {
		return nil, err
	}

	var exists int64
	s.DB.Model(&db.User{}).Where("username = ?", uname).Count(&exists)
	if exists > 0 {
		return nil, fmt.Errorf("用户名 %q 已存在", uname)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	if displayName == "" {
		displayName = uname
	}
	u := db.User{
		Username:           uname,
		PasswordHash:       hash,
		DisplayName:        displayName,
		Role:               role,
		Status:             1,
		MustChangePassword: mustChange,
	}
	if err := s.DB.Create(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// ListUsers 列出全部账号。
func (s *UserService) ListUsers() ([]db.User, error) {
	var users []db.User
	err := s.DB.Order("id ASC").Find(&users).Error
	return users, err
}

// ResetPassword 管理员重置口令，并踢掉该用户全部会话、置 must_change_password。
func (s *UserService) ResetPassword(username, newPassword string, mustChange bool) (*db.User, error) {
	uname := auth.NormalizeUsername(username)
	var u db.User
	if err := s.DB.Where("username = ?", uname).First(&u).Error; err != nil {
		return nil, fmt.Errorf("用户 %q 不存在", uname)
	}
	if err := auth.PasswordStrengthError(newPassword, false); err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return nil, err
	}
	if err := s.DB.Model(&db.User{}).Where("id = ?", u.ID).Updates(map[string]any{
		"password_hash":        hash,
		"must_change_password": mustChange,
		"failed_attempts":      0,
		"locked_until":         nil,
	}).Error; err != nil {
		return nil, err
	}
	// 口令已重置，旧会话必须全部失效。
	if err := s.DB.Where("user_id = ?", u.ID).Delete(&db.Session{}).Error; err != nil {
		return nil, err
	}
	u.MustChangePassword = mustChange
	return &u, nil
}

// RenameUser 改名。用于把 dev 的 admin 迁移成生产用户名。
func (s *UserService) RenameUser(from, to string) (*db.User, error) {
	oldName := auth.NormalizeUsername(from)
	newName := auth.NormalizeUsername(to)
	if err := auth.ValidateUsername(newName); err != nil {
		return nil, err
	}

	var u db.User
	if err := s.DB.Where("username = ?", oldName).First(&u).Error; err != nil {
		return nil, fmt.Errorf("用户 %q 不存在", oldName)
	}
	var exists int64
	s.DB.Model(&db.User{}).Where("username = ?", newName).Count(&exists)
	if exists > 0 {
		return nil, fmt.Errorf("目标用户名 %q 已被占用", newName)
	}
	if err := s.DB.Model(&db.User{}).Where("id = ?", u.ID).Update("username", newName).Error; err != nil {
		return nil, err
	}
	u.Username = newName
	return &u, nil
}

// SetStatus 启用/禁用账号。禁用时踢掉全部会话。
//
// 保护：不允许禁用最后一个启用状态的管理员，否则会把自己锁在门外。
func (s *UserService) SetStatus(username string, status int8) (*db.User, error) {
	uname := auth.NormalizeUsername(username)
	var u db.User
	if err := s.DB.Where("username = ?", uname).First(&u).Error; err != nil {
		return nil, fmt.Errorf("用户 %q 不存在", uname)
	}

	if status == 0 && u.Role == auth.RoleAdmin {
		var remaining int64
		s.DB.Model(&db.User{}).
			Where("role = ? AND status = 1 AND id <> ?", auth.RoleAdmin, u.ID).
			Count(&remaining)
		if remaining == 0 {
			return nil, errors.New("不能禁用最后一个启用中的管理员账号")
		}
	}

	if err := s.DB.Model(&db.User{}).Where("id = ?", u.ID).Update("status", status).Error; err != nil {
		return nil, err
	}
	if status == 0 {
		if err := s.DB.Where("user_id = ?", u.ID).Delete(&db.Session{}).Error; err != nil {
			return nil, err
		}
	}
	u.Status = status
	return &u, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}
