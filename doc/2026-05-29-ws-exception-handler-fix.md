# 2026-05-29 WebSocket handler 异常处理修复

## Bug 复现

见 `2026-05-29-cli-ws-connection-closed-bug.md`

## 根因分析

`readings.py:websocket_reading` handler 中：

```python
try:
    while True:
        data = await websocket.receive_json()
        # ...
        result = await reading_service.create_reading_langgraph(...)  # 这里可能抛异常
        session_id = result.get("session_id")
        await websocket.send_json(result)
except WebSocketDisconnect:
    pass
```

只 catch 了 `WebSocketDisconnect`。当 `create_reading_langgraph` 内部抛出异常（如数据库异常、AI 调用异常等），异常穿透 try 块，FastAPI 直接终止 WebSocket 连接，不发送关闭帧，客户端报 `ConnectionClosedError: no close frame received or sent`。

## 修复方案

增加通用 `Exception` catch，将错误信息返回给客户端，而不是让连接崩溃：

```python
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

            try:
                result = await reading_service.create_reading_langgraph(
                    question=question,
                    spread_type=spread_type,
                    session_id=sid,
                    model_name=model_name,
                )
                session_id = result.get("session_id")
                await websocket.send_json(result)
            except Exception as e:
                await websocket.send_json({
                    "error": str(e),
                    "status": "error",
                })
    except WebSocketDisconnect:
        pass
```

## 文件变更

| 文件 | 操作 |
|------|------|
| `app/api/readings.py` | 内层 try-except 包裹 `create_reading_langgraph` 调用 |
