# 2026-05-29 WebSocket CLI 实现

## 背景

当前后端以 RESTful API 为主（FastAPI），无任何 CLI 终端交互能力。需要提供三种 CLI 使用方式，覆盖不同场景。

## 现状分析

### 后端架构
- `backend/app/main.py` - FastAPI 应用入口
- `backend/app/api/readings.py` - RESTful 占卜接口
- `backend/app/service/reading_service.py` - 核心业务逻辑
- `backend/run.py` - uvicorn 启动入口
- `backend/scripts/init_db.py` / `import_cards.py` - 现有独立脚本模式

### 依赖
- 已有: fastapi, uvicorn, pydantic, peewee, langgraph, openai
- 需新增: websockets（仅用于 WebSocket 模式）

## 三种 CLI 模式

```
┌──────────────────────────────────────────────────────────────┐
│                     CLI 模式对比                              │
├─────────────┬──────────────┬────────────────┬────────────────┤
│             │ 模式A: 本地  │ 模式B: WS本地  │ 模式C: WS远程  │
├─────────────┼──────────────┼────────────────┼────────────────┤
│ 连接方式    │ 直接调用     │ ws://localhost │ ws://任意地址  │
│             │ service      │                │                │
├─────────────┼──────────────┼────────────────┼────────────────┤
│ 需要后端    │ 不需要       │ 需要           │ 需要           │
│ 运行        │              │                │                │
├─────────────┼──────────────┼────────────────┼────────────────┤
│ 适用场景    │ 本地开发调试 │ 本地完整测试   │ 分发给其他用户 │
├─────────────┼──────────────┼────────────────┼────────────────┤
│ 依赖        │ 无新增       │ websockets     │ websockets     │
├─────────────┼──────────────┼────────────────┼────────────────┤
│ 启动方式    │ cli --local  │ cli            │ cli -u ws://x  │
└─────────────┴──────────────┴────────────────┴────────────────┘
```

### 模式 A: 本地直调 CLI

最简单模式，直接调用 `reading_service`，不经过网络。

**启动**: `uv run python scripts/cli.py --local`

**实现**: `scripts/cli_local.py`
- 直接 `import` `reading_service.create_reading_langgraph`
- 同步调用（在 asyncio 中运行）
- 不需要启动后端服务

### 模式 B: WebSocket 本地 CLI

通过 WebSocket 连接 `ws://localhost:8000/api/v1/readings/ws`。

**启动**: `uv run python scripts/cli.py`

**实现**: `scripts/cli_ws.py`
- WebSocket 客户端连接本地后端
- 后端需先启动 `python run.py`

### 模式 C: WebSocket 远程 CLI

通过 WebSocket 连接任意地址的后端，可分发给其他用户。

**启动**: `uv run python scripts/cli.py -u ws://192.168.1.100:8000/api/v1/readings/ws`

**实现**: 复用 `scripts/cli_ws.py`，通过参数指定地址

### 统一入口

`scripts/cli.py` 作为统一入口，通过参数选择模式：

```python
#!/usr/bin/env python3
"""塔罗占卜 CLI 客户端"""
import asyncio
import json
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))


def main():
    args = sys.argv[1:]
    mode = "local" if "--local" in args else "ws"

    if mode == "local":
        from .cli_local import run
        run()
    else:
        # 解析 -u 参数
        ws_url = "ws://localhost:8000/api/v1/readings/ws"
        for i, arg in enumerate(args):
            if arg == "-u" and i + 1 < len(args):
                ws_url = args[i + 1]
        from .cli_ws import run
        run(ws_url)


if __name__ == "__main__":
    main()
```

## 实现步骤

### Step 1: 添加依赖

`backend/pyproject.toml`:
```toml
[project.scripts]
taro-cli = "scripts.cli:main"
```

### Step 2: 后端添加 WebSocket endpoint

在 `backend/app/api/readings.py` 中新增（与 HTTP 路由并行）：

```python
from fastapi import APIRouter, WebSocket, WebSocketDisconnect

@router.websocket("/ws")
async def websocket_reading(websocket: WebSocket):
    """WebSocket 占卜端"""
    await websocket.accept()
    session_id = None
    try:
        while True:
            data = await websocket.receive_json()
            question = data.get("question", "")
            spread_type = data.get("spread_type", "single")
            model_name = data.get("model_name")
            sid = data.get("session_id") or session_id

            result = await reading_service.create_reading_langgraph(
                question=question,
                spread_type=spread_type,
                session_id=sid,
                model_name=model_name,
            )
            session_id = result.get("session_id")
            await websocket.send_json(result)
    except WebSocketDisconnect:
        pass
```

### Step 3: 创建模式 A - 本地直调 CLI

新文件 `backend/scripts/cli_local.py`:

```python
#!/usr/bin/env python3
"""CLI - 本地直调模式"""
import asyncio
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from app.service.reading_service import create_reading_langgraph


def run():
    print("塔罗占卜（本地模式）\n")
    print("输入问题进行占卜（输入 quit 退出）\n")

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

        spread_type = "three"
        result = asyncio.run(
            create_reading_langgraph(
                question=question.strip(),
                spread_type=spread_type,
                session_id=session_id,
            )
        )
        session_id = result.get("session_id")
        _print_result(result)


def _print_result(data: dict):
    print(f"\n--- 占卜 #{data.get('reading_id')} ---")
    print(f"问题: {data.get('question')}\n")
    for card in data.get("cards", []):
        tag = " (逆位)" if card.get("is_reversed") else ""
        print(f"  位置[{card.get('position')}]: {card.get('name')}{tag}")
        interp = card.get("interpretation", "")
        if len(interp) > 300:
            interp = interp[:300] + "..."
        print(f"  解读: {interp}\n")
    print("---\n")
```

### Step 4: 创建模式 B/C - WebSocket CLI

新文件 `backend/scripts/cli_ws.py`:

```python
#!/usr/bin/env python3
"""CLI - WebSocket 模式"""
import asyncio
import json
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

try:
    import websockets
except ImportError:
    print("请安装 websockets: pip install websockets")
    sys.exit(1)


def run(ws_url: str):
    print(f"连接到塔罗服务: {ws_url}")
    try:
        asyncio.run(_interactive_session(ws_url))
    except (KeyboardInterrupt, EOFError):
        print("\n再见！")
    except ConnectionRefusedError:
        print("无法连接到服务，请确保后端正在运行: python run.py")
        sys.exit(1)


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

            payload = {
                "question": question.strip(),
                "spread_type": "three",
            }
            if session_id:
                payload["session_id"] = session_id

            await ws.send(json.dumps(payload))
            result = await ws.recv()
            data = json.loads(result)
            session_id = data.get("session_id")
            _print_result(data)


def _print_result(data: dict):
    print(f"\n--- 占卜 #{data.get('reading_id')} ---")
    print(f"问题: {data.get('question')}\n")
    for card in data.get("cards", []):
        tag = " (逆位)" if card.get("is_reversed") else ""
        print(f"  位置[{card.get('position')}]: {card.get('name')}{tag}")
        interp = card.get("interpretation", "")
        if len(interp) > 300:
            interp = interp[:300] + "..."
        print(f"  解读: {interp}\n")
    print("---\n")
```

## 使用方式

```bash
cd backend

# 模式 A: 本地直调（不需要启动后端）
uv run python scripts/cli.py --local

# 模式 B: WebSocket 本地连接（需要先启动后端）
uv run python run.py          # 终端1
uv run python scripts/cli.py  # 终端2

# 模式 C: WebSocket 远程连接
uv run python scripts/cli.py -u ws://192.168.1.100:8000/api/v1/readings/ws
```

## 文件变更清单

| 操作 | 文件 | 说明 |
|------|------|------|
| 修改 | `backend/pyproject.toml` | 添加 websockets 依赖 + entry point |
| 修改 | `backend/app/api/readings.py` | 新增 WebSocket 路由 |
| 新增 | `backend/scripts/cli.py` | CLI 统一入口 |
| 新增 | `backend/scripts/cli_local.py` | 模式A: 本地直调 |
| 新增 | `backend/scripts/cli_ws.py` | 模式B/C: WebSocket 客户端 |
