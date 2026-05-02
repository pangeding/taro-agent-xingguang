from dataclasses import dataclass
from ..core.config import settings


@dataclass
class LLMConfig:
    provider: str
    api_key: str
    base_url: str
    model: str
    temperature: float = 0.7
    max_tokens: int = 1000


def get_llm_config(model_name: str = None) -> LLMConfig:
    api_key = settings.DASHSCOPE_API_KEY
    base_url = settings.DASHSCOPE_BASE_URL
    model = model_name or settings.DASHSCOPE_MODEL
    return LLMConfig("dashscope", api_key, base_url, model)
