package config

import (
	"log"

	"github.com/caarlos0/env/v11"
)

type Settings struct {
	APIV1Str           string   `env:"API_V1_STR" envDefault:"/api/v1"`
	ProjectName        string   `env:"PROJECT_NAME" envDefault:"星光塔罗AI助手"`
	Version            string   `env:"VERSION" envDefault:"0.1.0"`
	DatabaseURL        string   `env:"DATABASE_URL" envDefault:"data/taro.db"`
	DashScopeAPIKey    string   `env:"DASHSCOPE_API_KEY"`
	DashScopeBaseURL   string   `env:"DASHSCOPE_BASE_URL" envDefault:"https://dashscope.aliyuncs.com/compatible-mode/v1"`
	DashScopeModel     string   `env:"DASHSCOPE_MODEL"`
	BackendCORSOrigins []string `env:"BACKEND_CORS_ORIGINS" envDefault:"http://localhost:3000,http://127.0.0.1:3000"`
}

var Config *Settings

func Load() *Settings {
	cfg, err := env.ParseAs[Settings]()
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
	Config = &cfg
	return Config
}
