package config

import (
	"fmt"
	"log"
	"net/url"
	"strings"

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
