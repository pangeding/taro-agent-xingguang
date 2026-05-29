# 2026-05-29 WebSocket CLI 连接关闭 bug

## 现象

通过 `cli_ws.py` 连接后端 WebSocket 发送问题后，后端关闭连接且客户端没有收到正常响应。

## 复现步骤

```bash
# 终端1: 启动后端
cd backend && uv run python run.py

# 终端2: 启动 CLI
cd backend && uv run python scripts/cli_ws.py -u ws://localhost:8000/api/v1/readings/ws

# 输入问题后崩溃
你想知道什么？> 我的工作运势如何
```

## 完整报错

```
Traceback (most recent call last):
  File "/home/pangding/ai/taro_agent/backend/scripts/cli_ws.py", line 75, in <module>
    run(ws_url)
  File "/home/pangding/ai/taro_agent/backend/scripts/cli_ws.py", line 20, in run
    asyncio.run(_interactive_session(ws_url))
  File "/home/pangding/.local/share/uv/python/cpython-3.12.13-linux-x86_64-gnu/lib/python3.12/asyncio/runners.py", line 195, in run
    return runner.run(main)
           ^^^^^^^^^^^^^^^^
  File "/home/pangding/.local/share/uv/python/cpython-3.12.13-linux-x86_64-gnu/lib/python3.12/asyncio/runners.py", line 118, in run
    return self._loop.run_until_complete(task)
           ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  File "/home/pangding/.local/share/uv/python/cpython-3.12.13-linux-x86_64-gnu/lib/python3.12/asyncio/base_events.py", line 691, in run_until_complete
    return future.result()
           ^^^^^^^^^^^^^^^
  File "/home/pangding/ai/taro_agent/backend/scripts/cli_ws.py", line 51, in _interactive_session
    result = await ws.recv()
             ^^^^^^^^^^^^^^^
  File "/home/pangding/ai/taro_agent/backend/.venv/lib/python3.12/site-packages/websockets/asyncio/connection.py", line 324, in recv
    raise self.protocol.close_exc from self.recv_exc
websockets.exceptions.ConnectionClosedError: no close frame received or sent
```

## 分析

`ConnectionClosedError: no close frame received or sent` 说明 **后端主动异常关闭了连接**，没有发送正常的 WebSocket 关闭帧。

问题出在后端 `websocket_reading` handler 中，`create_reading_langgraph` 内部出现异常（很可能是数据库操作异常），但 handler 没有 catch，导致整个 WebSocket 连接崩溃。
