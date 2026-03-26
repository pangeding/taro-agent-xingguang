from pydantic_settings import BaseSettings
from typing import Optional


class Settings(BaseSettings):
    """应用配置"""

    # API设置
    API_V1_STR: str = "/api/v1"
    PROJECT_NAME: str = "星光塔罗AI助手"
    VERSION: str = "0.1.0"

    # 数据库设置 (使用SQLite简化)
    DATABASE_URL: str = "sqlite:///./taro.db"

    # DeepSeek API设置
    DEEPSEEK_API_KEY: str = ""
    DEEPSEEK_API_BASE: str = "https://api.deepseek.com"
    DEEPSEEK_MODEL: str = "deepseek-chat"

    # CORS设置
    BACKEND_CORS_ORIGINS: list[str] = ["http://localhost:3000", "http://127.0.0.1:3000"]

    class Config:
        env_file = ".env"
        case_sensitive = True


settings = Settings()