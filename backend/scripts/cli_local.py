#!/usr/bin/env python3
"""CLI - 本地直调模式（不需要启动后端服务）"""
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
