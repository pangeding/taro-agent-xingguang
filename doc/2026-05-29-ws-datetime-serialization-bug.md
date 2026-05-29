# 2026-05-29 WebSocket datetime JSON 序列化错误

## 现象

通过 WebSocket 发送占卜请求后，后端报 `TypeError: Object of type datetime is not JSON serializable`。

## 完整报错

```
ERROR:    Exception in ASGI application
Traceback (most recent call last):
  ...
  File "/home/pangding/ai/taro_agent/backend/app/api/readings.py", line 61, in websocket_reading
    await websocket.send_json(result)
  File ".../starlette/websockets.py", line 174, in send_json
    text = json.dumps(data, separators=(",", ":"), ensure_ascii=False)
  File ".../json/__init__.py", line 238, in dumps
    **kw).encode(obj)
  File ".../json/encoder.py", line 258, in iterencode
    return _iterencode(o, 0)
  File ".../json/encoder.py", line 180, in default
    raise TypeError(f'Object of type {o.__class__.__name__} '
TypeError: Object of type datetime is not JSON serializable
```

## 根因分析

`reading_service.create_reading_langgraph()` 返回的 dict 中 `created_at` 字段是 `datetime` 对象（peewee DateTimeField），`websocket.send_json()` 内部调用 `json.dumps` 无法处理 datetime 类型。

HTTP 路由没有问题，因为 `ReadingResponse` Pydantic model 会自动转换 datetime 为 ISO 字符串。但 WebSocket 端直接传 dict，绕过了 Pydantic 序列化。

## 修复方案

在 `reading_service.py` 中，将 `created_at` 转为字符串：

```python
"created_at": reading.created_at.isoformat() if reading.created_at else None,
```

## 文件变更

| 文件 | 操作 |
|------|------|
| `app/service/reading_service.py` | `create_reading_langgraph` 返回中 `created_at` 转 isoformat |
