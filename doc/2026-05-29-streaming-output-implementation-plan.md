# 塔罗占卜后端流式输出改造方案

## 1. 背景与现状

### 1.1 当前架构

```
CLI (cli_local.py)                    CLI (cli_ws.py)
    |                                      |
    v                                      v
reading_service.create_reading_langgraph()  WebSocket连接
    |                                      |
    v                                      v
LangGraph ainvoke() (一次性返回)         reading_service.create_reading_langgraph()
    |                                      |
    v                                      v
完整结果一次性返回                       完整结果一次性返回
```

### 1.2 问题分析

**cli_local.py**: 调用 `asyncio.run(create_reading_langgraph())`，该方法内部调用 `reading_graph.ainvoke()`，`ainvoke` 是 LangGraph 的**阻塞式调用**，等待整张图执行完毕后一次性返回全部结果。无任何流式能力。

**cli_ws.py**: 通过 WebSocket 连接，但服务端收到请求后同样调用 `create_reading_langgraph()`，等全部完成后才 `websocket.send_json(result)`。WebSocket 本身**完全支持流式输出**，但当前服务端实现是"全量完成后一次性发送"，没有利用 WebSocket 的逐帧推送能力。

### 1.3 核心瓶颈

流式输出的关键不在 CLI 端，而在**后端服务层和 LangGraph 执行方式**。当前瓶颈在：

1. **LLM 层**: `LLMClient.chat_completion()` 使用 `chat.completions.create()` 非流式 API，等待 LLM 生成全部文本后才返回
2. **LangGraph 层**: 使用 `ainvoke()` 而非 `astream()` / `astream_events()`，无法感知中间状态
3. **Service 层**: `create_reading_langgraph()` 是整体函数，无 yield / 异步生成器能力
4. **API 层**: HTTP 端点使用 `response_model=ReadingResponse` 固定结构；WebSocket 端点未做分段推送

---

## 2. 基础知识

### 2.1 什么是流式输出（Streaming）

流式输出指服务端在数据处理过程中，**边生成边推送**，而不是等所有数据处理完毕后才一次性返回。对于塔罗占卜场景：

- **非流式**: 用户提问 → 等待 10-30 秒 LLM 生成所有解读 → 一次性返回全部结果
- **流式**: 用户提问 → 抽牌结果立即返回 → 每张牌的解读生成后立即返回 → 综合解读生成后立即返回

### 2.2 WebSocket 是否支持流式输出

**完全支持**。WebSocket 是全双工通信协议，服务端可以随时主动向客户端推送消息。流式输出的实现方式是：服务端在处理过程中，分多次 `send()` 数据，每次发送一个"进度片段"。

WebSocket 流式模式示例：
```python
# 服务端
await websocket.send_json({"type": "cards_drawn", "data": {...}})
await websocket.send_json({"type": "card_interpretation", "data": {"position": 0, "text": "..."} })
await websocket.send_json({"type": "card_interpretation", "data": {"position": 1, "text": "..."} })
await websocket.send_json({"type": "synthesis", "data": {"text": "..." }})
await websocket.send_json({"type": "complete", "data": {...}})

# 客户端
while True:
    msg = await ws.recv()
    data = json.loads(msg)
    if data["type"] == "complete":
        break
    # 实时处理每个片段
```

### 2.3 SSE (Server-Sent Events)

SSE 是基于 HTTP 的单向流式推送技术。与 WebSocket 的对比：

| 特性 | WebSocket | SSE |
|------|-----------|-----|
| 通信方向 | 双向 | 服务端→客户端（单向） |
| 协议 | ws:// 或 wss:// | 标准 http/https |
| 流式能力 | 支持（分帧发送） | 支持（EventSource） |
| 适用场景 | 实时双向交互 | 只需要服务端推送 |
| FastAPI 支持 | `@app.websocket()` | `StreamingResponse` |

对于塔罗占卜，WebSocket 已在使用，继续扩展即可。HTTP 端点也可以额外增加 SSE 支持。

### 2.4 LangGraph 流式执行

LangGraph 提供多种流式执行方式：

- **`ainvoke()`**: 阻塞式，等整张图执行完返回（当前使用方式）
- **`astream()`**: 每个 node 执行完后 yield 一次状态快照
- **`astream_events()`**: 更细粒度，可以感知 node 内部的事件（如 LLM token 级别的流式）

```python
# astream 示例：每个 node 完成后推送一次
async for event in reading_graph.astream(initial_state):
    print(event)  # 每次是一个 dict，包含最新 state

# astream_events 示例：可以感知 LLM 的每个 token
async for event in reading_graph.astream_events(initial_state, version="v2"):
    if event["event"] == "on_chat_model_stream":
        print(event["data"]["chunk"].content)  # 每个 token
```

### 2.5 OpenAI 兼容接口的流式调用

大多数兼容 OpenAI API 的 LLM 服务商都支持流式输出：

```python
# 非流式（当前使用方式）
resp = await client.chat.completions.create(
    model="gpt-4", messages=messages, stream=False)
text = resp.choices[0].message.content  # 等待全部生成完

# 流式
stream = await client.chat.completions.create(
    model="gpt-4", messages=messages, stream=True)
async for chunk in stream:
    delta = chunk.choices[0].delta
    if delta.content:
        print(delta.content, end="")  # 逐 token 输出
```

---

## 3. 改造方案

### 3.1 方案总览

提供三种方案，按改动量从小到大排列：

| 方案 | 改动范围 | 流式粒度 | 推荐场景 |
|------|----------|----------|----------|
| A. 仅 WebSocket 分段推送 | 后端 API + CLI | 节点级（抽牌→解读1→解读2→综合） | 最快上线，用户感知明显改善 |
| B. WebSocket + LLM 级流式 | 后端 API + LLMClient + CLI | Token 级（LLM 逐字输出） | 最佳用户体验 |
| C. HTTP SSE + WebSocket 双通道 | 全链路 | 节点级/Token 级 | 同时支持 HTTP 客户端和 WS 客户端 |

**推荐按 A → B → C 的顺序逐步实施。**

---

### 3.2 方案 A：WebSocket 分段推送（节点级流式）

#### 3.2.1 改动清单

**新增文件：**
- `backend/app/service/reading_service_streaming.py` — 流式版 service

**修改文件：**
- `backend/app/api/readings.py` — WebSocket 端点改为流式推送
- `backend/scripts/cli_ws.py` — 客户端解析多段消息

#### 3.2.2 新增流式 Service

```python
# backend/app/service/reading_service_streaming.py
"""流式版占卜服务 - 分段推送进度"""
import uuid
import asyncio
from typing import AsyncGenerator
from ..db.models import Reading, ReadingCard, TarotCard
from ..agent.graph import reading_graph
from ..agent.state import ReadingState


def _draw_cards(count: int) -> list[dict]:
    """（复用原有逻辑）随机抽牌"""
    card_ids = [card.id for card in TarotCard.select(TarotCard.id)]
    selected_ids = random.sample(card_ids, min(count, len(card_ids)))
    cards = []
    for card_id in selected_ids:
        card = TarotCard.get(TarotCard.id == card_id)
        is_reversed = random.choice([True, False])
        cards.append({
            "id": card.id, "name": card.name, "is_reversed": is_reversed,
        })
    return cards


async def create_reading_langgraph_streaming(
    question: str,
    spread_type: str,
    session_id: str = None,
    model_name: str = None,
) -> AsyncGenerator[dict, None]:
    """流式版占卜 - 通过 AsyncGenerator 分段 yield 结果"""

    session_id = session_id or str(uuid.uuid4())

    # Step 1: 创建数据库记录
    reading = Reading.create(
        session_id=session_id, question=question, spread_type=spread_type,
    )

    # Step 2: 抽牌并立即推送
    cards = _draw_cards(1 if spread_type == "single" else 3)
    for i, card_data in enumerate(cards):
        ReadingCard.create(
            reading=reading,
            card=TarotCard.get(TarotCard.id == card_data["id"]),
            position=i, is_reversed=card_data["is_reversed"], interpretation="",
        )

    yield {
        "type": "cards_drawn",
        "session_id": session_id,
        "reading_id": reading.id,
        "cards": [
            {"name": c["name"], "position": i, "is_reversed": c["is_reversed"]}
            for i, c in enumerate(cards)
        ],
    }

    # Step 3: 使用 astream 逐步执行 LangGraph
    initial_state = ReadingState(
        question=question, spread_type=spread_type,
        session_id=session_id, model_name=model_name,
        cards_info=[
            {"id": c["id"], "name": c["name"],
             "is_reversed": c["is_reversed"], "position": i,
             "card_obj": TarotCard.get(TarotCard.id == c["id"])}
            for i, c in enumerate(cards)
        ],
        interpretations=[], synthesis=None, status="success", error=None,
    )

    async for event in reading_graph.astream(initial_state):
        # astream 在每个 node 完成后 yield 一次 state
        if "interpret" in event:
            interpretations = event["interpret"]["interpretations"]
            for idx, interp in enumerate(interpretations):
                yield {
                    "type": "card_interpretation",
                    "position": interp["position"],
                    "card_name": interp["card_name"],
                    "interpretation": interp["interpretation"],
                    "status": interp["status"],
                }
                # 更新数据库
                rc = ReadingCard.get(
                    (ReadingCard.reading == reading) &
                    (ReadingCard.position == interp["position"])
                )
                rc.interpretation = interp["interpretation"]
                rc.save()

        if "synthesize" in event:
            synthesis = event["synthesize"].get("synthesis", "")
            yield {
                "type": "synthesis",
                "interpretation": synthesis,
            }
            if synthesis and spread_type == "three":
                last_card = ReadingCard.select().where(
                    (ReadingCard.reading == reading)
                ).order_by(ReadingCard.position.desc()).first()
                last_card.interpretation += f"\n\n---\n**综合解读**: {synthesis}"
                last_card.save()

    # Step 4: 推送完成信号
    yield {
        "type": "complete",
        "reading_id": reading.id,
        "session_id": session_id,
    }
```

#### 3.2.3 修改 WebSocket API

```python
# backend/app/api/readings.py 中修改 websocket 端点

@router.websocket("/ws")
async def websocket_reading(websocket: WebSocket):
    await websocket.accept()
    session_id = None
    try:
        while True:
            data = await websocket.receive_json()
            question = data.get("question", "")
            spread_type = data.get("spread_type", "single")
            model_name = data.get("model_name")
            sid = data.get("session_id") or session_id

            try:
                # 改为流式调用
                async for chunk in reading_service_streaming.create_reading_langgraph_streaming(
                    question=question, spread_type=spread_type,
                    session_id=sid, model_name=model_name,
                ):
                    await websocket.send_json(chunk)
                    if chunk["type"] == "complete":
                        session_id = chunk["session_id"]
            except Exception as e:
                await websocket.send_json({"error": str(e), "status": "error"})
    except WebSocketDisconnect:
        pass
```

#### 3.2.4 修改 CLI WebSocket 客户端

```python
# backend/scripts/cli_ws.py 中修改 _interactive_session

async def _interactive_session(ws_url: str):
    async with websockets.connect(ws_url) as ws:
        print("已连接！输入问题进行占卜（输入 quit 退出）\n")
        session_id = None
        while True:
            try:
                question = input("你想知道什么？> ")
            except (KeyboardInterrupt, EOFError):
                break
            if question.lower() in ("quit", "exit", "q"):
                break
            if not question.strip():
                continue

            payload = {"question": question.strip(), "spread_type": "single"}
            if session_id:
                payload["session_id"] = session_id

            await ws.send(json.dumps(payload))

            # 流式接收多段消息
            while True:
                result = await ws.recv()
                data = json.loads(result)
                msg_type = data.get("type")

                if msg_type == "cards_drawn":
                    session_id = data.get("session_id")
                    print(f"\n--- 占卜 #{data.get('reading_id')} ---")
                    print(f"问题: {data.get('cards', [])}")
                elif msg_type == "card_interpretation":
                    print(f"  [{data['position']}] {data['card_name']}: {data['interpretation']}\n")
                elif msg_type == "synthesis":
                    print(f"  综合解读: {data['interpretation']}\n")
                elif msg_type == "complete":
                    print("--- 结束 ---\n")
                    break
                elif msg_type == "error":
                    print(f"  错误: {data.get('error')}\n")
                    break
```

#### 3.2.5 优点与局限

- **优点**: 改动量小，不改动 LLM 层，用户能立即看到抽牌结果和逐张解读
- **局限**: 每张牌内部的 LLM 生成过程仍然是阻塞的，用户看到一张牌的解读时，仍需等待该牌解读完全生成

---

### 3.3 方案 B：LLM 级流式（Token 级）

#### 3.3.1 改动清单

**修改文件：**
- `backend/app/agent/llm_client.py` — 增加 `stream_chat_completion` 方法
- `backend/app/agent/nodes.py` — 解读节点改为流式聚合
- `backend/app/agent/state.py` — 可能需要增加 token callback
- `backend/app/service/reading_service_streaming.py` — 配合方案 A 增加 token 级推送

#### 3.3.2 LLMClient 增加流式方法

```python
# backend/app/agent/llm_client.py

class LLMClient:
    # ... 原有代码不变 ...

    async def stream_chat_completion(
        self, messages: list[dict], system_prompt: str = ""
    ) -> AsyncGenerator[str, None]:
        """流式对话 - 逐 token yield"""
        if system_prompt:
            messages = [{"role": "system", "content": system_prompt}] + messages
        stream = await self._client.chat.completions.create(
            model=self._model,
            messages=messages,
            temperature=self._temperature,
            max_tokens=self._max_tokens,
            stream=True,  # 关键参数
        )
        full_text = ""
        async for chunk in stream:
            delta = chunk.choices[0].delta
            if delta and delta.content:
                full_text += delta.content
                yield delta.content
```

#### 3.3.3 Node 改为流式

```python
# backend/app/agent/nodes.py

async def interpret_cards_streaming(state: ReadingState) -> AsyncGenerator[dict, None]:
    """流式版解读节点 - 每张牌的每个 token 都 yield"""
    config = get_llm_config(state.get("model_name"))
    client = LLMClient(config)
    interpretations = []

    for ci in state["cards_info"]:
        card = ci["card_obj"]
        prompt = build_card_prompt(card, ci["is_reversed"], state["question"],
                                   ci["position"], state["spread_type"])

        # 推送开始信号
        yield {
            "type": "card_start",
            "position": ci["position"],
            "card_name": ci["name"],
        }

        card_text = ""
        try:
            async for token in client.stream_chat_completion(
                messages=[{"role": "user", "content": prompt}],
                system_prompt=SYSTEM_PROMPT,
            ):
                card_text += token
                # 推送 token 级事件
                yield {
                    "type": "card_token",
                    "position": ci["position"],
                    "token": token,
                }

            interpretations.append({
                "card_id": ci["id"], "card_name": ci["name"],
                "is_reversed": ci["is_reversed"], "position": ci["position"],
                "interpretation": card_text.strip(), "status": "success",
            })
        except Exception:
            interpretations.append({
                "card_id": ci["id"], "card_name": ci["name"],
                "is_reversed": ci["is_reversed"], "position": ci["position"],
                "interpretation": get_basic_interpretation(card, ci["is_reversed"]),
                "status": "fallback",
            })

        yield {
            "type": "card_end",
            "position": ci["position"],
            "interpretation": card_text.strip(),
        }

    state["interpretations"] = interpretations
```

#### 3.3.4 使用 LangGraph astream_events

更优雅的方式是使用 LangGraph 的 `astream_events`，它能自动捕获 LLM 的流式事件：

```python
async for event in reading_graph.astream_events(initial_state, version="v2"):
    event_type = event["event"]

    if event_type == "on_chat_model_stream":
        # LLM token 级别的事件
        chunk = event["data"]["chunk"]
        if chunk.content:
            yield {
                "type": "llm_token",
                "token": chunk.content,
                "node": event.get("metadata", {}).get("langgraph_node", ""),
            }
    elif event_type == "on_chain_end":
        # node 完成事件
        ...
```

#### 3.3.5 优点与局限

- **优点**: 用户可以看到 LLM 逐字生成的过程，体验最佳
- **局限**: 改动量较大，需要重构 node 逻辑；对 LLM API 的流式支持有依赖（部分 API 可能不支持或限制流式）

---

### 3.4 方案 C：HTTP SSE + WebSocket 双通道

#### 3.4.1 思路

在方案 A/B 的基础上，为 HTTP API 也增加流式能力，使用 Server-Sent Events：

```python
# backend/app/api/readings.py

from fastapi.responses import StreamingResponse

@router.post("/langgraph/stream")
async def create_reading_langgraph_stream(request: ReadingRequestV2):
    """SSE 流式占卜端点"""
    async def event_stream():
        async for chunk in reading_service_streaming.create_reading_langgraph_streaming(
            question=request.question, spread_type=request.spread_type,
            session_id=request.session_id, model_name=request.model_name,
        ):
            yield f"data: {json.dumps(chunk, ensure_ascii=False)}\n\n"

    return StreamingResponse(event_stream(), media_type="text/event-stream")
```

这样 HTTP 客户端（如浏览器 EventSource、curl）也能获得流式体验。

#### 3.4.2 优点与局限

- **优点**: 同时支持 WebSocket 和 HTTP 两种流式通道，兼容性最好
- **局限**: 维护两套流式通道，复杂度增加

---

## 4. 推荐实施顺序

```
阶段 1: 方案 A（WebSocket 分段推送）
  ├── 新增 reading_service_streaming.py
  ├── 修改 readings.py 的 WS 端点
  └── 修改 cli_ws.py

阶段 2: 方案 B（LLM 级流式）
  ├── LLMClient 增加 stream_chat_completion
  ├── nodes.py 改为流式或使用 astream_events
  └── 整合到 WS 端点

阶段 3: 方案 C（HTTP SSE，可选）
  ├── 新增 /langgraph/stream SSE 端点
  └── cli_local.py 改为 SSE 客户端（或保留非流式）
```

---

## 5. 关键技术决策

### 5.1 AsyncGenerator 模式

Python 的 `AsyncGenerator` 是实现后端流式的核心模式：

```python
async def streaming_function() -> AsyncGenerator[dict, None]:
    yield {"step": 1, "data": "..."}
    await asyncio.sleep(1)  # 模拟耗时
    yield {"step": 2, "data": "..."}
```

调用方：
```python
async for chunk in streaming_function():
    print(chunk)  # 每次 yield 都会到这里
```

### 5.2 数据库写入时机

流式模式下，数据库写入有两种策略：

- **边生成边写入**: 每张牌解读完成后立即 `save()`，数据安全性高
- **全部完成后写入**: 性能略好，但中断时数据丢失

推荐方案 A 的"边生成边写入"策略。

### 5.3 错误处理

流式模式下错误需要在每个 yield 点检查：

```python
try:
    async for chunk in stream:
        yield chunk
except Exception as e:
    yield {"type": "error", "message": str(e)}
```

---

## 6. 术语表

| 术语 | 解释 |
|------|------|
| Streaming / 流式输出 | 边生成边推送，不等全部完成 |
| WebSocket | 双向实时通信协议 |
| SSE (Server-Sent Events) | 基于 HTTP 的单向推送 |
| AsyncGenerator | Python 异步生成器，支持 async for |
| ainvoke | LangGraph 阻塞式调用 |
| astream | LangGraph 流式调用，每 node 完成后 yield |
| astream_events | LangGraph 事件级流式，可感知 LLM token |
| Token | LLM 生成的最小文本单元（约 1 个英文单词或 1 个中文字） |
| Node | LangGraph 图中的处理节点 |


