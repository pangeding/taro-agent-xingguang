# 2026-05-29 CLI 入口缺失修复

## 问题

`scripts/cli_local.py` 和 `scripts/cli_ws.py` 只有 `run()` 函数定义，没有 `if __name__ == "__main__"` 入口，导致直接运行 `python scripts/cli_local.py` 时 import 完就退出，没有任何交互。

## 修复

### 1. cli_local.py 末尾添加

```python
if __name__ == "__main__":
    run()
```

### 2. cli_ws.py 末尾添加

```python
if __name__ == "__main__":
    import sys
    ws_url = "ws://localhost:8000/api/v1/readings/ws"
    for i, arg in enumerate(sys.argv):
        if arg == "-u" and i + 1 < len(sys.argv):
            ws_url = sys.argv[i + 1]
    run(ws_url)
```

## 文件变更

| 文件 | 操作 |
|------|------|
| `scripts/cli_local.py` | 末尾追加 main guard |
| `scripts/cli_ws.py` | 末尾追加 main guard |
