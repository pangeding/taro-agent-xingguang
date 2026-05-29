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
