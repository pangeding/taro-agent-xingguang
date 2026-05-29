from pydantic_settings import BaseSettings
from typing import Optional
from dotenv import load_dotenv

import os

load_dotenv(dotenv_path=os.path.join(os.path.dirname(os.path.dirname(__file__)), '..', '.env'))

class Settings(BaseSettings):
    """应用配置"""

    # API设置
    API_V1_STR: str = "/api/v1"
    PROJECT_NAME: str = "星光塔罗AI助手"
    VERSION: str = "0.1.0"

    # 数据库设置 (使用SQLite简化)
    DATABASE_URL: str = os.getenv("DATABASE_URL")

    # DeepSeek API设置
    DASHSCOPE_API_KEY: str = os.getenv("DASHSCOPE_API_KEY")
    DASHSCOPE_BASE_URL: str = os.getenv("DASHSCOPE_BASE_URL")
    DASHSCOPE_MODEL: str = os.getenv("DASHSCOPE_MODEL")

    # CORS设置
    BACKEND_CORS_ORIGINS: list[str] = os.getenv("BACKEND_CORS_ORIGINS")

    class Config:
        env_file = ".env"
        case_sensitive = True


settings = Settings()