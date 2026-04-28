# LangGraph 迁移技术文档
# Outdated and deprecated
## 一、现状分析

### 1.1 当前架构

```
FastAPI (main.py)
  └── API Router (api/__init__.py)
        ├── cards router (api/cards.py)
        └── readings router (api/readings.py)
              └── reading_service (service/reading_service.py)
                    └── interpret_card (agent/interpreter.py)
                          └── DashScope API (httpx)
```

### 1.2 现有 agent 模块

**文件**: `backend/app/agent/`
- `__init__.py`: 仅模块文档字符串
- `interpreter.py`: 核心解读逻辑 (159行)

**interpreter.py 功能拆解**:

| 函数 | 职责 | 行数 |
|------|------|------|
| `get_client()` | 创建/缓存 httpx.AsyncClient | 12 |
| `interpret_card()` | 入口函数，调度 prompt 构建 + API 调用 + 降级 | 24 |
| `build_prompt()` | 构建塔罗牌解读提示词 | 49 |
| `call_dashscope_api()` | 调用 DashScope OpenAI 兼容接口 | 27 |
| `get_basic_interpretation()` | API 失败时的降级解读 | 14 |
| `close_client()` | 关闭 HTTP 客户端 | 5 |

### 1.3 当前存在的问题

#### 1.3.1 模型耦合
- API Key、Base URL、Model 硬编码绑定到 DashScope
- `call_dashscope_api()` 函数名和实现强耦合单一供应商
- 无法在运行时切换模型
- 配置中 `DASHSCOPE_*` 前缀限制了多模型扩展

#### 1.3.2 三张牌阵逻辑缺陷
当前 `reading_service.py:_generate_interpretations()`:
- 对每张牌独立调用 `interpret_card()`
- 三次 API 调用之间无上下文关联
- 缺少"整体综合分析"环节
- 每张牌的解读不知道其他牌的结果

#### 1.3.3 无状态管理
- 每次 `interpret_card()` 是独立调用
- 不支持多轮对话/记忆
- 无法积累上下文

#### 1.3.4 无工具调用能力
- 当前是纯 prompt → LLM → text 的线性流程
- LLM 无法主动查询数据库（牌面含义、历史记录等）
- 无法进行多步推理

#### 1.3.5 错误处理单一
- 仅 try/except + 降级为模板文本
- 无重试机制
- 无超时控制（仅 httpx 全局 timeout）

### 1.4 现有依赖

```
pydantic-settings    # 配置管理
peewee               # ORM
httpx                # HTTP 客户端
fastapi              # Web 框架
python-dotenv        # 环境变量
```

## 二、迁移目标

### 2.1 核心目标
1. 使用 LangGraph 重构 agent 模块
2. 支持多模型切换（DashScope、OpenAI、其他 OpenAI 兼容接口）
3. 修复三张牌阵逻辑（整体综合分析）
4. 为未来工具调用、记忆功能预留架构

## 三、架构设计

### 3.1 整体架构

```
FastAPI
  └── reading_service
        └── TarotAgent (LangGraph)
              ├── 单牌阵: draw_card → interpret → output
              ├── 三牌阵: draw_cards → interpret_each → synthesize → output
              └── 降级路径: interpret → fallback → output

              └── LLM Router
                    ├── DashScope (当前默认)
                    ├── OpenAI
                    └── OpenAI Compatible
```

### 3.2 LangGraph 工作流设计

#### 3.2.1 单牌阵工作流

```
ENTRY → build_prompt → call_llm → check_response → EXIT
                                        ↓ (失败)
                                   fallback → EXIT
```

#### 3.2.2 三牌阵工作流

```
ENTRY → build_individual_prompts → call_llm_batch → check_responses
                                                        ↓ (部分失败)
                                                   fallback_partial
                                                        ↓
                                               synthesize_reading → EXIT
```

### 3.3 状态定义

```python
class CardInterpretation(TypedDict):
    card_id: int
    card_name: str
    is_reversed: bool
    position: int
    prompt: str
    interpretation: str
    status: Literal["success", "fallback"]

class ReadingState(TypedDict):
    # 输入
    question: str
    spread_type: Literal["single", "three"]
    session_id: str
    model_name: str  # 新增：支持运行时模型选择

    # 抽牌结果
    drawn_cards: list[dict]  # [{id, name, is_reversed}]

    # 每张牌的解读
    interpretations: list[CardInterpretation]

    # 三牌阵综合分析
    synthesis: str

    # 元信息
    status: Literal["success", "partial", "failed"]
    error: str | None
```

### 3.4 模型抽象层

```python
class LLMConfig(TypedDict):
    provider: Literal["dashscope", "openai", "compatible"]
    api_key: str
    base_url: str
    model: str
    temperature: float
    max_tokens: int

class LLMClient:
    async def chat_completion(self, messages: list[dict], config: LLMConfig) -> str: ...
```

### 3.5 文件结构

```
backend/app/agent/
├── __init__.py           # 模块入口，导出 TarotAgent
├── interpreter.py        # [保留] 旧实现，暂时兼容
├── state.py              # [新增] 工作流状态定义
├── nodes.py              # [新增] 工作流节点函数
├── graph.py              # [新增] LangGraph 工作流构建
├── llm_client.py         # [新增] 模型抽象层
├── prompts.py            # [新增] 提示词模板（从 interpreter.py 抽出）
└── fallback.py           # [新增] 降级逻辑（从 interpreter.py 抽出）
```

## 四、详细实现方案

### 4.1 配置扩展

**修改**: `backend/app/core/config.py`

```python
# 新增配置项
LLM_PROVIDER: str = "dashscope"           # 默认供应商
LLM_API_KEY: str = ""                     # 统一 API Key
LLM_BASE_URL: str = ""                    # 统一 Base URL
LLM_MODEL: str = ""                       # 统一模型名称
LLM_TEMPERATURE: float = 0.7
LLM_MAX_TOKENS: int = 1000

# 保留向后兼容
DASHSCOPE_API_KEY: str = ""
DASHSCOPE_BASE_URL: str = ""
DASHSCOPE_MODEL: str = ""
```

配置优先级：`LLM_*` > `DASHSCOPE_*`（兼容旧配置）

### 4.2 状态定义

**新建**: `backend/app/agent/state.py`

```python
from typing import TypedDict, Optional, Literal, Annotated
from langgraph.graph.message import add_messages

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

    drawn_cards: list[dict]

    interpretations: list[CardInterpretation]

    synthesis: Optional[str]

    status: Literal["success", "partial", "failed"]
    error: Optional[str]
```

### 4.3 模型抽象层

**新建**: `backend/app/agent/llm_client.py`

使用 OpenAI SDK 作为统一客户端（DashScope 兼容 OpenAI 接口）：

```python
from openai import AsyncOpenAI

class LLMClient:
    def __init__(self, config: LLMConfig):
        self.client = AsyncOpenAI(
            api_key=config.api_key,
            base_url=config.base_url,
        )
        self.model = config.model
        self.temperature = config.temperature
        self.max_tokens = config.max_tokens

    async def chat_completion(self, messages: list[dict], system_prompt: str = None) -> str:
        if system_prompt:
            messages = [{"role": "system", "content": system_prompt}] + messages

        response = await self.client.chat.completions.create(
            model=self.model,
            messages=messages,
            temperature=self.temperature,
            max_tokens=self.max_tokens,
        )
        return response.choices[0].message.content
```

### 4.4 提示词模板

**新建**: `backend/app/agent/prompts.py`

从 `interpreter.py:build_prompt()` 抽取：

```python
SYSTEM_PROMPT = "你是一位专业、富有洞察力的塔罗牌解读师..."

def build_card_prompt(card, is_reversed, question, position, spread_type) -> str:
    # 原有逻辑迁移

def build_synthesis_prompt(interpretations: list[CardInterpretation], question: str) -> str:
    # 三牌阵综合分析提示词（新增）
```

### 4.5 降级逻辑

**新建**: `backend/app/agent/fallback.py`

从 `interpreter.py:get_basic_interpretation()` 抽取：

```python
def get_basic_interpretation(card, is_reversed) -> str:
    # 原有逻辑迁移
```

### 4.6 工作流节点

**新建**: `backend/app/agent/nodes.py`

```python
async def build_prompt_node(state: ReadingState) -> ReadingState:
    """为每张牌构建提示词"""

async def call_llm_node(state: ReadingState) -> ReadingState:
    """调用 LLM 获取解读"""

async def fallback_node(state: ReadingState) -> ReadingState:
    """API 失败时的降级处理"""

async def synthesize_node(state: ReadingState) -> ReadingState:
    """三牌阵综合分析（新增）"""
```

### 4.7 工作流图

**新建**: `backend/app/agent/graph.py`

```python
from langgraph.graph import StateGraph, END
from .state import ReadingState

def create_reading_graph() -> CompiledStateGraph:
    builder = StateGraph(ReadingState)

    builder.add_node("build_prompts", build_prompt_node)
    builder.add_node("call_llm", call_llm_node)
    builder.add_node("fallback", fallback_node)
    builder.add_node("synthesize", synthesize_node)

    builder.set_entry_point("build_prompts")
    builder.add_edge("build_prompts", "call_llm")
    builder.add_conditional_edges("call_llm", route_after_llm)
    builder.add_edge("synthesize", END)

    return builder.compile()

def route_after_llm(state: ReadingState) -> str:
    if all(i["status"] == "success" for i in state["interpretations"]):
        return "synthesize" if state["spread_type"] == "three" else END
    else:
        return "fallback"
```

### 4.8 Agent 入口

**修改**: `backend/app/agent/__init__.py`

```python
from .graph import create_reading_graph

# 全局编译一次
reading_graph = create_reading_graph()
```

### 4.9 Service 层适配

**修改**: `backend/app/service/reading_service.py`

将 `_generate_interpretations()` 改为调用 LangGraph 工作流：

```python
from ..agent import reading_graph

async def _generate_interpretations(reading_id: int, model_name: str = None):
    reading = Reading.get(Reading.id == reading_id)
    reading_cards = list(ReadingCard.select().where(ReadingCard.reading == reading))

    initial_state = ReadingState(
        question=reading.question,
        spread_type=reading.spread_type,
        session_id=reading.session_id,
        model_name=model_name,
        drawn_cards=[
            {"id": rc.card.id, "name": rc.card.name, "is_reversed": rc.is_reversed}
            for rc in reading_cards
        ],
        interpretations=[],
        synthesis=None,
        status="success",
        error=None,
    )

    result = await reading_graph.ainvoke(initial_state)

    # 将结果写回数据库
    for i, interp in enumerate(result["interpretations"]):
        reading_cards[i].interpretation = interp["interpretation"]
        reading_cards[i].save()
```

### 4.10 API 层支持（可选）

**修改**: `backend/app/model/request.py`

新增 model_name 字段：

```python
class ReadingRequest(BaseModel):
    question: str
    spread_type: str = "single"
    session_id: Optional[str] = None
    model_name: Optional[str] = None  # 新增：运行时模型选择
```

## 五、实施步骤

### Phase 1: 基础设施
1. 添加依赖 `openai`, `langgraph` 到 pyproject.toml
2. 扩展 `config.py` 配置项
3. 创建 `llm_client.py` 模型抽象层
4. 创建 `prompts.py` 提示词模板
5. 创建 `fallback.py` 降级逻辑

### Phase 2: 工作流核心
6. 创建 `state.py` 状态定义
7. 创建 `nodes.py` 节点函数
8. 创建 `graph.py` 工作流图
9. 更新 `__init__.py` 导出

### Phase 3: 集成
10. 修改 `reading_service.py` 接入 LangGraph
11. 修改 `request.py` 添加 model_name 字段
12. 三牌阵综合分析节点实现

### Phase 4: 验证
13. 单牌阵正向测试
14. 单牌阵 API 失败降级测试
15. 三牌阵正向测试（验证综合分析）
16. 模型切换测试

## 六、依赖变更

### 新增
```
langgraph>=0.2.0
openai>=1.0.0
```

### 移除
```
httpx  # 被 openai SDK 替代（openai 内部依赖 httpx）
```

## 七、向后兼容

1. `interpreter.py` 暂时保留，不删除
2. `DASHSCOPE_*` 配置项保留，作为 `LLM_*` 的 fallback
3. API 接口不变，仅新增可选的 `model_name` 字段
4. 现有前端无需修改

## 八、未来扩展预留

| 功能 | 扩展点 |
|------|--------|
| 记忆功能 | State 中增加 `message_history`，使用 LangGraph 的 `add_messages` |
| 工具调用 | 在 nodes 中增加 tool_call 节点，LLM Client 增加 function calling 支持 |
| MCP | 增加 MCP Server 节点 |
| 流式输出 | LangGraph 支持 streaming，节点使用 `astream_events` |
| 多轮对话 | State 增加 `conversation_history`，graph 增加 human-in-the-loop |
