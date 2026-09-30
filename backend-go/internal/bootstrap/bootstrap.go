// Package bootstrap 负责启动期的账号引导与生产环境保护。
//
// 独立于 service 包：这样迁移脚本与 admin_user CLI 都能复用它，
// 而不必连带引入 agent / eino 那一大串依赖。
//
// 三层保护见 doc/2026-09-28-用户系统与数据隔离技术文档.md §3.3：
//
//	L1 引导账号校验（users 表为空时）
//	L2 存量弱口令扫描（每次启动，仅 prod）
//	L3 强制改密门禁（按请求生效，在 middleware/auth.go）
//
// L1/L2 失败一律返回 error，由调用方 log.Fatalf —— fail-closed，绝不降级启动。
package bootstrap

import (
	"errors"
	"fmt"
	"log"

	"backend-go/internal/auth"
	"backend-go/internal/config"
	"backend-go/internal/db"

	"gorm.io/gorm"
)

// Users 执行启动期引导：确保有可用账号，并在 prod 下拒绝弱口令库。
//
// 返回新建的引导账号（若本次创建了），否则返回 (nil, nil)。
func Users(d *gorm.DB, cfg *config.Settings) (*db.User, error) {
	var count int64
	if err := d.Model(&db.User{}).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("统计 users 表失败: %w", err)
	}

	var created *db.User
	if count == 0 {
		u, err := createBootstrapAdmin(d, cfg)
		if err != nil {
			return nil, err
		}
		created = u
	}

	// L2：每次启动都扫，仅 prod 拦截。
	// 这是防止「把 dev 库直接搬到 prod」的保险：dev 建的 admin/123456 会在这里被拦下。
	if cfg.IsProd() {
		if err := scanWeakPasswords(d); err != nil {
			return nil, err
		}
	}

	return created, nil
}

// createBootstrapAdmin 创建首个管理员账号（L1）。
func createBootstrapAdmin(d *gorm.DB, cfg *config.Settings) (*db.User, error) {
	username := auth.NormalizeUsername(cfg.BootstrapAdminUsername)
	password := cfg.BootstrapAdminPassword

	if cfg.IsProd() {
		// prod 必须显式配置，且不得使用保留用户名与弱口令。
		if username == "" {
			return nil, errors.New("APP_ENV=prod 且 users 表为空：必须设置 BOOTSTRAP_ADMIN_USERNAME")
		}
		if password == "" {
			return nil, errors.New("APP_ENV=prod 且 users 表为空：必须设置 BOOTSTRAP_ADMIN_PASSWORD")
		}
		if auth.IsReservedUsername(username) {
			return nil, fmt.Errorf("APP_ENV=prod：引导管理员用户名 %q 是保留字，生产环境不允许使用（admin/root/administrator/test/guest/user 等）", username)
		}
		if err := auth.ValidateBootstrapCredential(username, password); err != nil {
			return nil, fmt.Errorf("APP_ENV=prod：BOOTSTRAP_ADMIN_PASSWORD 不合规：%w", err)
		}
	} else {
		// dev 有默认值，开箱即用。
		if username == "" {
			username = auth.DefaultDevUsername
		}
		if password == "" {
			password = auth.DefaultDevPassword
		}
	}

	if err := auth.ValidateUsername(username); err != nil {
		return nil, fmt.Errorf("引导用户名 %q 不合法: %w", username, err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("生成口令哈希失败: %w", err)
	}

	u := db.User{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  username,
		Role:         auth.RoleAdmin,
		Status:       1,
		// 引导账号不设 must_change_password：
		// prod 下这里的口令已经过强度校验；dev 下要的就是"开箱即用、不打扰"。
		// 该标志留给 admin_user reset-password 与 create --must-change 使用。
		MustChangePassword: false,
	}
	if err := d.Create(&u).Error; err != nil {
		return nil, fmt.Errorf("创建引导管理员失败: %w", err)
	}

	if cfg.IsProd() {
		log.Printf("[bootstrap] 已创建生产引导管理员 %q", u.Username)
	} else {
		log.Printf("[bootstrap] 已创建开发引导账号 %s / %s（APP_ENV=%s，生产环境将拒绝此类默认口令）",
			u.Username, password, cfg.AppEnv)
	}
	return &u, nil
}

// scanWeakPasswords（L2）逐个用户比对弱口令表，命中即返回 error。
//
// bcrypt 无法反查，只能逐条试；用户量在个位数，启动开销可忽略。
func scanWeakPasswords(d *gorm.DB) error {
	var users []db.User
	if err := d.Where("status = ?", 1).Find(&users).Error; err != nil {
		return fmt.Errorf("读取用户列表失败: %w", err)
	}

	var offenders []string
	for i := range users {
		if auth.MatchesKnownWeakPassword(users[i].PasswordHash) {
			offenders = append(offenders, users[i].Username)
		}
	}
	if len(offenders) > 0 {
		return fmt.Errorf(
			"APP_ENV=prod 检测到 %d 个使用默认/弱口令的账号（%v），拒绝在生产环境启动。"+
				"请先执行：go run scripts/admin_user/main.go reset-password --username <用户名> --password <强口令>",
			len(offenders), offenders)
	}
	return nil
}
