#!/usr/bin/env python3
"""塔罗占卜 CLI 客户端 - 统一入口"""
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))


def main():
    args = sys.argv[1:]
    mode = "local" if "--local" in args else "ws"

    if mode == "local":
        from scripts.cli_local import run
        run()
    else:
        ws_url = "ws://localhost:8000/api/v1/readings/ws"
        for i, arg in enumerate(args):
            if arg == "-u" and i + 1 < len(args):
                ws_url = args[i + 1]
        from scripts.cli_ws import run
        run(ws_url)


if __name__ == "__main__":
    main()
