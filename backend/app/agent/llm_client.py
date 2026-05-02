from openai import AsyncOpenAI
from .config import LLMConfig


class LLMClient:
    def __init__(self, config: LLMConfig):
        self._client = AsyncOpenAI(api_key=config.api_key, base_url=config.base_url)
        self._model = config.model
        self._temperature = config.temperature
        self._max_tokens = config.max_tokens

    async def chat_completion(self, messages: list[dict], system_prompt: str = "") -> str:
        if system_prompt:
            messages = [{"role": "system", "content": system_prompt}] + messages
        resp = await self._client.chat.completions.create(
            model=self._model,
            messages=messages,
            temperature=self._temperature,
            max_tokens=self._max_tokens,
        )
        return resp.choices[0].message.content
