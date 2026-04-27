# 2026-04-27 pyproject.toml 移动至 backend 目录的兼容问题修复

## 问题清单

### P1 Makefile init-db/import-cards 路径
- 文件: Makefile
- 行: 43-52
- 问题: `python scripts/init_db.py` 和 `python scripts/import_cards.py` 脚本已移至 `backend/scripts/`
- 修复: 改为 `cd backend && python scripts/init_db.py` 和 `cd backend && python scripts/import_cards.py`

### P2 import_cards.py sys.path 错误
- 文件: backend/scripts/import_cards.py
- 行: 9
- 问题: `sys.path.append(os.path.join(os.path.dirname(__file__), "..", "backend"))` 假设脚本在根目录
- 修复: 改为 `sys.path.append(os.path.join(os.path.dirname(__file__), ".."))`

## 执行顺序
1. 修复 Makefile
2. 修复 import_cards.py sys.path
3. 验证: 根目录执行 `make init-db` `make import-cards` `make backend`

注: .env 保持在 backend/ 目录下，原有相对路径模式不变。Makefile 修复后所有后端命令均 `cd backend && ...` 执行，.env 加载和数据库路径自然不受影响。
