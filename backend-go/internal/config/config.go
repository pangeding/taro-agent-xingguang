package config

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Settings struct {
	APIV1Str           string   `env:"API_V1_STR" envDefault:"/api/v1"`
	ProjectName        string   `env:"PROJECT_NAME" envDefault:"星光塔罗AI助手"`
	Version            string   `env:"VERSION" envDefault:"0.1.0"`
	DatabaseURL        string   `env:"DATABASE_URL" envDefault:"data/tarot.db"`
	DBDriver           string   `env:"DB_DRIVER"`
	MySQLHost          string   `env:"MYSQL_HOST"`
	MySQLPort          string   `env:"MYSQL_PORT" envDefault:"3306"`
	MySQLDB            string   `env:"MYSQL_DB"`
	MySQLUser          string   `env:"MYSQL_USER"`
	MySQLPassword      string   `env:"MYSQL_PASSWORD"`
	DashScopeAPIKey    string   `env:"DASHSCOPE_API_KEY"`
	DashScopeBaseURL   string   `env:"DASHSCOPE_BASE_URL" envDefault:"https://dashscope.aliyuncs.com/compatible-mode/v1"`
	DashScopeModel     string   `env:"DASHSCOPE_MODEL"`
	BackendCORSOrigins []string `env:"BACKEND_CORS_ORIGINS" envDefault:"http://localhost:3000,http://127.0.0.1:3000"`

	// —— 用户系统（见 doc/2026-09-28-用户系统与数据隔离技术文档.md §3.3）——
	// AppEnv 决定生产保护是否生效：dev | prod。
	AppEnv string `env:"APP_ENV" envDefault:"dev"`
	// BootstrapAdmin* 仅在 users 表为空时用于创建首个账号。
	// dev 有默认值（admin/123456，开箱即用）；prod 必须显式配置且通过强度与保留字校验。
	BootstrapAdminUsername string `env:"BOOTSTRAP_ADMIN_USERNAME"`
	BootstrapAdminPassword string `env:"BOOTSTRAP_ADMIN_PASSWORD"`
	SessionCookieName      string `env:"SESSION_COOKIE_NAME" envDefault:"tarot_session"`
	SessionTTLHours        int    `env:"SESSION_TTL_HOURS" envDefault:"720"`
	// CookieSecure 三态：留空则跟随 AppEnv（prod=true）。本地以 http 起 prod 调试时可显式设 false。
	CookieSecure string `env:"COOKIE_SECURE"`
	// BindAddr 留空则 prod 绑 127.0.0.1:8000（由 nginx 反代），否则绑 :8000。
	BindAddr string `env:"BIND_ADDR"`
}

// IsProd 返回是否处于生产环境保护模式。
func (s *Settings) IsProd() bool { return strings.EqualFold(strings.TrimSpace(s.AppEnv), "prod") }

// SecureCookie 返回 Cookie 是否带 Secure 标志。留空时跟随 IsProd。
func (s *Settings) SecureCookie() bool {
	switch strings.ToLower(strings.TrimSpace(s.CookieSecure)) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	}
	return s.IsProd()
}

// ListenAddr 返回 HTTP 监听地址。prod 默认只绑回环，不直接对外。
func (s *Settings) ListenAddr() string {
	if a := strings.TrimSpace(s.BindAddr); a != "" {
		return a
	}
	if s.IsProd() {
		return "127.0.0.1:8000"
	}
	return ":8000"
}

// SessionTTL 返回会话有效期。
func (s *Settings) SessionTTL() time.Duration {
	if s.SessionTTLHours <= 0 {
		return 720 * time.Hour
	}
	return time.Duration(s.SessionTTLHours) * time.Hour
}

var Config *Settings

func Load() *Settings {
	// Overload：以 .env 为准，覆盖 shell 中可能存在的过期同名变量
	_ = godotenv.Overload()
	cfg, err := env.ParseAs[Settings]()
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
	cfg.BackendCORSOrigins = normalizeOrigins(cfg.BackendCORSOrigins)
	Config = &cfg
	return Config
}

// normalizeOrigins 兼容逗号分隔与 JSON 数组（["a","b"]）两种写法。
func normalizeOrigins(in []string) []string {
	var out []string
	for _, o := range in {
		o = strings.TrimSpace(o)
		o = strings.Trim(o, "[]")
		o = strings.TrimSpace(o)
		o = strings.Trim(o, "\"'")
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

// MySQLDSN 由 MYSQL_* 字段构造 go-sql-driver DSN。
func (s *Settings) MySQLDSN() string {
	if s.MySQLDB == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		s.MySQLUser, s.MySQLPassword, s.MySQLHost, s.MySQLPort, s.MySQLDB)
}

// EffectiveDB 返回最终使用的驱动名（mysql/sqlite）与 DSN。
// 优先级：DATABASE_URL 的 mysql:// 前缀 > DB_DRIVER/MYSQL_HOST > sqlite。
func (s *Settings) EffectiveDB() (string, string) {
	if strings.HasPrefix(s.DatabaseURL, "mysql://") {
		return "mysql", mysqlURLToDSN(s.DatabaseURL)
	}
	if s.DBDriver == "mysql" || (s.DBDriver == "" && s.MySQLHost != "") {
		return "mysql", s.MySQLDSN()
	}
	return "sqlite", s.DatabaseURL
}

// mysqlURLToDSN 将 mysql://user:pass@host:port/db?params 转换为 go-sql-driver DSN。
// 注意：userinfo 中的特殊字符需 URL 编码（如 @ -> %40）。
func mysqlURLToDSN(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	user := u.User.Username()
	pass, _ := u.User.Password()
	database := strings.TrimPrefix(u.Path, "/")
	q := u.RawQuery
	if !strings.Contains(q, "charset") {
		q = appendParam(q, "charset=utf8mb4")
	}
	if !strings.Contains(q, "parseTime") {
		q = appendParam(q, "parseTime=True")
	}
	if !strings.Contains(q, "loc") {
		q = appendParam(q, "loc=Local")
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", user, pass, u.Host, database, q)
}

func appendParam(query, param string) string {
	if query == "" {
		return param
	}
	return query + "&" + param
}
