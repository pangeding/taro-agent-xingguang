# LangGraph 迁移 - 实施指南

## 原则

**增量更新**：只做追加，不改既有逻辑。

## 任务

- `pyproject.toml` 新增: `langgraph>=0.2.0`, `openai>=1.0.0`
- 新增 agent 目录下文件: `config.py`, `state.py`, `prompts.py`, `fallback.py`, `llm_client.py`, `nodes.py`, `graph.py`
- 追加内容到: `core/config.py`, `agent/__init__.py`, `service/reading_service.py`, `model/request.py`
- `agent/interpreter.py` 不动

## Step 1: core/config.py 追加

读取现有 `core/config.py`，在末尾追加:
```python
LLM_PROVIDER: str = "dashscope"
LLM_API_KEY: str = ""
LLM_BASE_URL: str = ""
LLM_MODEL: str = ""
LLM_TEMPERATURE: float = 0.7
LLM_MAX_TOKENS: int = 1000
```

## Step 2: agent/config.py

```python
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
    api_key = settings.LLM_API_KEY or settings.DASHSCOPE_API_KEY
    base_url = settings.LLM_BASE_URL or settings.DASHSCOPE_BASE_URL
    model = model_name or settings.LLM_MODEL or settings.DASHSCOPE_MODEL
    provider = settings.LLM_PROVIDER
    return LLMConfig(provider, api_key, base_url, model)
```

## Step 3: agent/llm_client.py

```python
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
            model=self._model, messages=messages,
            temperature=self._temperature, max_tokens=self._max_tokens,
        )
        return resp.choices[0].message.content
```

## Step 4: agent/prompts.py

从 `agent/interpreter.py` 读取 `build_prompt()` 函数和 `SYSTEM_PROMPT`，复制为 `build_card_prompt()` 和 `SYSTEM_PROMPT`，并新增 `build_synthesis_prompt()`:

```python
SYSTEM_PROMPT = "你是一位专业、富有洞察力的塔罗牌解读师，擅长结合牌面含义和用户具体问题提供深入、个性化的解读。你的解读充满智慧、共情和启发性。"

def build_card_prompt(card, is_reversed, question, position=None, spread_type="single"):
    """从 interpreter.py:build_prompt 复制，函数名改为 build_card_prompt"""
    # TODO: 将 interpreter.py 中 build_prompt 的内容复制到这里，函数名改为 build_card_prompt

def build_synthesis_prompt(interpretations, question):
    cards_text = "\n".join([
        f"- 位置{i+1}（{'过去' if i==0 else '现在' if i==1 else '未来'}）: "
        f"{interp['card_name']}（{'逆位' if interp['is_reversed'] else '正位'}）\n"
        f"  解读: {interp['interpretation']}"
        for i, interp in enumerate(interpretations)
    ])
    return f"""
    你是一位专业的塔罗牌解读师。以下是三张牌的独立解读：

    ## 用户问题
    {question}

    ## 各牌解读
    {cards_text}

    ## 综合解读要求
    1. 将三张牌串联为一个完整的时间线叙事
    2. 分析牌与牌之间的能量流转和关联
    3. 给出整体性的建议
    4. 字数在300-500字左右
    """.strip()
```

## Step 5: agent/fallback.py

从 `agent/interpreter.py` 读取 `get_basic_interpretation()`，复制到 `fallback.py`:

```python
def get_basic_interpretation(card, is_reversed):
    """从 interpreter.py 原样复制"""
    # TODO: 从 interpreter.py:get_basic_interpretation 复制到这里
```

## Step 6: agent/state.py

```python
from typing import TypedDict, Optional, Literal

class CardInterpretation(TypedDict):
    card_id: int
    card_name: str
    is_reversed: bool
    position: int
    interpretation: str
    status: Literal["success", "fallback"]

class ReadingState(TypedDict):
    question: str
    spread_type: Literal["single", "three"]
    session_id: str
    model_name: Optional[str]
    cards_info: list                          # [{id, name, is_reversed, position, card_obj}]
    interpretations: list[CardInterpretation]
    synthesis: Optional[str]
    status: Literal["success", "partial", "failed"]
    error: Optional[str]
```

## Step 7: agent/nodes.py

```python
from .state import ReadingState, CardInterpretation
from .config import get_llm_config
from .llm_client import LLMClient
from .prompts import SYSTEM_PROMPT, build_card_prompt, build_synthesis_prompt
from .fallback import get_basic_interpretation

async def interpret_cards(state: ReadingState) -> ReadingState:
    config = get_llm_config(state.get("model_name"))
    client = LLMClient(config)
    interpretations = []
    for ci in state["cards_info"]:
        card = ci["card_obj"]
        prompt = build_card_prompt(card, ci["is_reversed"], state["question"],
                                   ci["position"], state["spread_type"])
        try:
            text = await client.chat_completion(
                messages=[{"role": "user", "content": prompt}], system_prompt=SYSTEM_PROMPT)
            interpretations.append({
                "card_id": ci["id"], "card_name": ci["name"],
                "is_reversed": ci["is_reversed"], "position": ci["position"],
                "interpretation": text.strip(), "status": "success"})
        except Exception:
            interpretations.append({
                "card_id": ci["id"], "card_name": ci["name"],
                "is_reversed": ci["is_reversed"], "position": ci["position"],
                "interpretation": get_basic_interpretation(card, ci["is_reversed"]),
                "status": "fallback"})
    state["interpretations"] = interpretations
    return state

async def synthesize(state: ReadingState) -> ReadingState:
    config = get_llm_config(state.get("model_name"))
    client = LLMClient(config)
    prompt = build_synthesis_prompt(state["interpretations"], state["question"])
    try:
        state["synthesis"] = (await client.chat_completion(
            messages=[{"role": "user", "content": prompt}], system_prompt=SYSTEM_PROMPT)).strip()
    except Exception as e:
        state["synthesis"] = "（综合分析暂时不可用）"
        state["status"] = "partial"
        state["error"] = str(e)
    return state
```

## Step 8: agent/graph.py

```python
from langgraph.graph import StateGraph, END
from .state import ReadingState
from .nodes import interpret_cards, synthesize

def create_reading_graph():
    builder = StateGraph(ReadingState)
    builder.add_node("interpret", interpret_cards)
    builder.add_node("synthesize", synthesize)
    builder.set_entry_point("interpret")
    builder.add_conditional_edges("interpret",
        lambda s: "synthesize" if s["spread_type"] == "three" else END)
    builder.add_edge("synthesize", END)
    return builder.compile()
```

## Step 9: agent/__init__.py

读取现有 `__init__.py`，末尾追加:
```python
from .graph import create_reading_graph
reading_graph = create_reading_graph()
```

## Step 10: service/reading_service.py 追加

读取现有 `reading_service.py`，追加:
```python
from ..agent import reading_graph
from ..agent.state import ReadingState

async def _generate_interpretations_langgraph(reading_id: int, model_name: str = None):
    reading = Reading.get(Reading.id == reading_id)
    reading_cards = list(ReadingCard.select().where(ReadingCard.reading == reading))
    initial_state = ReadingState(
        question=reading.question, spread_type=reading.spread_type,
        session_id=reading.session_id, model_name=model_name,
        cards_info=[{"id": rc.card.id, "name": rc.card.name,
                     "is_reversed": rc.is_reversed, "position": rc.position,
                     "card_obj": rc.card} for rc in reading_cards],
        interpretations=[], synthesis=None, status="success", error=None)
    result = await reading_graph.ainvoke(initial_state)
    for i, interp in enumerate(result["interpretations"]):
        reading_cards[i].interpretation = interp["interpretation"]
        reading_cards[i].save()
    if result.get("synthesis") and reading.spread_type == "three":
        last_card = reading_cards[-1]
        last_card.interpretation += f"\n\n---\n**综合解读**: {result['synthesis']}"
        last_card.save()
```

## Step 11: model/request.py 追加

读取现有 `request.py`，追加:
```python
class ReadingRequestV2(BaseModel):
    question: str
    spread_type: str = "single"
    session_id: Optional[str] = None
    model_name: Optional[str] = None
```
