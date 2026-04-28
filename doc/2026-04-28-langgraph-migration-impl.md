# LangGraph 迁移 - 实施指南

## 原则

**增量更新，旧有不变**：只新增文件，不修改已有文件的逻辑；旧接口保持原样，新能力通过额外入口提供。

## 目标
新增 LangGraph 工作流，支持多模型切换，修复三牌阵逻辑。旧 `interpreter.py` 不动。

## 依赖

`pyproject.toml` 新增:
```
langgraph>=0.2.0
openai>=1.0.0
```
保留 `httpx`（旧 interpreter.py 仍使用，不删除）。

## 文件结构（仅标记新增，不改动已有文件逻辑）

```
backend/app/agent/
├── __init__.py          # 新增: 导出 reading_graph（保留原有导出）
├── interpreter.py       # 【不动】旧接口保持原样
├── config.py            # [新增] LLM 配置
├── state.py             # [新增] 工作流状态
├── prompts.py           # [新增] 提示词模板
├── fallback.py          # [新增] 降级逻辑
├── llm_client.py        # [新增] 模型抽象层
├── nodes.py             # [新增] 工作流节点
└── graph.py             # [新增] 工作流图

backend/app/service/
└── reading_service.py   # 新增: _generate_interpretations_langgraph()，旧方法不动

backend/app/model/
└── request.py           # 新增: ReadingRequestV2 继承旧 Request，追加可选 model_name 字段
```

## 兼容性保障

| 旧接口 | 新接口 | 兼容方式 |
|--------|--------|----------|
| `interpreter.interpret_card()` | `graph.ainvoke(state)` | `interpreter.py` 不动，任何调用方不受影响 |
| `reading_service._generate_interpretations()` | `_generate_interpretations_langgraph()` | 新增独立方法，旧方法不动 |
| `ReadingRequest(question, spread_type)` | `ReadingRequestV2(question, spread_type, model_name=None)` | 新模型独立定义，旧模型不变 |

## Step 1: 扩展全局配置

修改 `backend/app/core/config.py`，**仅新增字段，不改已有字段**:

```python
# LLM 统一配置（新增）
LLM_PROVIDER: str = "dashscope"
LLM_API_KEY: str = ""
LLM_BASE_URL: str = ""
LLM_MODEL: str = ""
LLM_TEMPERATURE: float = 0.7
LLM_MAX_TOKENS: int = 1000
```

规则: `LLM_*` 优先，为空则 fallback 到 `DASHSCOPE_*`，保证旧配置仍可用。

## Step 2: 新建 agent/config.py

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

## Step 3: 新建 agent/llm_client.py

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
            model=self._model,
            messages=messages,
            temperature=self._temperature,
            max_tokens=self._max_tokens,
        )
        return resp.choices[0].message.content
```

## Step 4: 新建 agent/prompts.py

- `build_card_prompt()` 逻辑从 `interpreter.py:build_prompt()` 原样复制
- 新增 `build_synthesis_prompt()` 用于三牌阵综合分析

```python
SYSTEM_PROMPT = "你是一位专业、富有洞察力的塔罗牌解读师，擅长结合牌面含义和用户具体问题提供深入、个性化的解读。你的解读充满智慧、共情和启发性。"

def build_card_prompt(card, is_reversed, question, position=None, spread_type="single"):
    """原样迁移自 interpreter.py:build_prompt()"""
    position_desc = ""
    if spread_type == "three" and position is not None:
        positions = ["过去", "现在", "未来"]
        if position < len(positions):
            position_desc = f"这张牌在牌阵中代表{positions[position]}。"

    reversal_desc = "逆位" if is_reversed else "正位"
    meaning = card.meaning_reversed if is_reversed else card.meaning_upright

    return f"""
    你是一位专业的塔罗牌解读师，请为以下抽牌结果提供深入、富有洞察力的解读：

    ## 用户问题
    {question}

    ## 抽牌结果
    - 牌名：{card.name}
    - 牌型：{card.arcana_type}（{"大阿尔卡纳" if card.arcana_type == "major" else "小阿尔卡纳"}）
    - 方位：{reversal_desc}
    {f"- 花色：{card.suit}" if card.suit else ""}
    {f"- 数字：{card.number}" if card.number else ""}
    - 关键词：{card.keywords}
    {position_desc}

    ## 牌面基础含义
    {meaning}

    ## 解读要求
    1. 结合用户问题分析这张牌的含义
    2. 解释这张牌在当前问题中的象征意义
    3. 提供具体的建议或启示
    4. 语言亲切、富有共情力
    5. 避免过于笼统的描述，要具体化
    6. 字数在200-300字左右

    请开始你的解读：
    """.strip()

def build_synthesis_prompt(interpretations, question):
    """三牌阵综合分析提示词（新增）"""
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

## Step 5: 新建 agent/fallback.py

从 `interpreter.py:get_basic_interpretation()` 原样迁移:

```python
def get_basic_interpretation(card, is_reversed):
    """原样迁移自 interpreter.py"""
    meaning = card.meaning_reversed if is_reversed else card.meaning_upright
    reversal_desc = "逆位" if is_reversed else "正位"
    return f"""
    【{card.name} - {reversal_desc}】

    基础含义：{meaning}

    关键词：{card.keywords}

    （注：AI深度解读暂时不可用，这是牌面的基础含义。）
    """.strip()
```

## Step 6: 新建 agent/state.py

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

## Step 7: 新建 agent/nodes.py

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
                messages=[{"role": "user", "content": prompt}],
                system_prompt=SYSTEM_PROMPT,
            )
            interpretations.append({
                "card_id": ci["id"], "card_name": ci["name"],
                "is_reversed": ci["is_reversed"], "position": ci["position"],
                "interpretation": text.strip(), "status": "success",
            })
        except Exception:
            text = get_basic_interpretation(card, ci["is_reversed"])
            interpretations.append({
                "card_id": ci["id"], "card_name": ci["name"],
                "is_reversed": ci["is_reversed"], "position": ci["position"],
                "interpretation": text, "status": "fallback",
            })

    state["interpretations"] = interpretations
    return state

async def synthesize(state: ReadingState) -> ReadingState:
    config = get_llm_config(state.get("model_name"))
    client = LLMClient(config)
    prompt = build_synthesis_prompt(state["interpretations"], state["question"])
    try:
        state["synthesis"] = (await client.chat_completion(
            messages=[{"role": "user", "content": prompt}],
            system_prompt=SYSTEM_PROMPT,
        )).strip()
    except Exception as e:
        state["synthesis"] = "（综合分析暂时不可用）"
        state["status"] = "partial"
        state["error"] = str(e)
    return state
```

## Step 8: 新建 agent/graph.py

```python
from langgraph.graph import StateGraph, END
from .state import ReadingState
from .nodes import interpret_cards, synthesize

def create_reading_graph():
    builder = StateGraph(ReadingState)
    builder.add_node("interpret", interpret_cards)
    builder.add_node("synthesize", synthesize)
    builder.set_entry_point("interpret")
    builder.add_conditional_edges(
        "interpret",
        lambda s: "synthesize" if s["spread_type"] == "three" else END,
    )
    builder.add_edge("synthesize", END)
    return builder.compile()
```

## Step 9: 修改 agent/__init__.py

在现有导出基础上新增：

```python
"""
AI塔罗牌解读模块
基于DeepSeek API的智能解读引擎
"""

# 保持旧接口导出
from .interpreter import interpret_card  # noqa

# 新增 LangGraph 工作流
from .graph import create_reading_graph
reading_graph = create_reading_graph()
```

## Step 10: 新增 service/reading_service.py 中的 LangGraph 路径

新增 `_generate_interpretations_langgraph()` 方法，旧 `_generate_interpretations()` 不动：

```python
from ..agent import reading_graph

async def _generate_interpretations_langgraph(reading_id: int, model_name: str = None):
    reading = Reading.get(Reading.id == reading_id)
    reading_cards = list(ReadingCard.select().where(ReadingCard.reading == reading))

    initial_state = ReadingState(
        question=reading.question,
        spread_type=reading.spread_type,
        session_id=reading.session_id,
        model_name=model_name,
        cards_info=[{
            "id": rc.card.id, "name": rc.card.name,
            "is_reversed": rc.is_reversed, "position": rc.position,
            "card_obj": rc.card,
        } for rc in reading_cards],
        interpretations=[],
        synthesis=None,
        status="success",
        error=None,
    )

    result = await reading_graph.ainvoke(initial_state)

    for i, interp in enumerate(result["interpretations"]):
        reading_cards[i].interpretation = interp["interpretation"]
        reading_cards[i].save()

    # 三牌阵：追加综合分析
    if result.get("synthesis") and reading.spread_type == "three":
        last_card = reading_cards[-1]
        last_card.interpretation += f"\n\n---\n**综合解读**: {result['synthesis']}"
        last_card.save()
```

## Step 11: 新增 model/request.py 中的新请求模型

新增 `ReadingRequestV2`，继承或扩展旧模型，不影响已有调用：

```python
class ReadingRequestV2(BaseModel):
    question: str
    spread_type: str = "single"
    session_id: Optional[str] = None
    model_name: Optional[str] = None  # 新增，可选
```

## 验证清单

- [ ] `uv build` 通过
- [ ] 旧调用 `from agent.interpreter import interpret_card` 仍可用
- [ ] 旧调用 `await interpret_card(...)` 返回与之前相同的文本
- [ ] 单牌阵：返回 AI 解读
- [ ] 单牌阵 API 失败：返回降级文本
- [ ] 三牌阵：3 张牌各有解读 + 末尾追加综合分析
- [ ] 不传 `model_name` 时使用默认配置（行为不变）
