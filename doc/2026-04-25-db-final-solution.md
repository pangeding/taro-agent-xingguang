# 2026-04-25 技术文档：数据库迁移完整解决方案

## 问题总结
1. 路径混乱导致的数据库访问失败
   - 数据文件路径错误：`../data/tarot_cards.json` → 需调整为 `backend/data`
   - 模块导入失败：`No module named 'app'`
   - 配置路径未同步更新

## 标准化路径结构
```
backend/
├── app/
│   ├── __init__.py
│   └── db/
│       ├── __init__.py
│       └── models.py
├── scripts/
│   ├── init_db.py
│   └── import_cards.py
├── data/
│   └── tarot_cards.json
└── app/db/
    └── taro.db
```

## 完整实施步骤

### 1. 目录结构调整
```bash
mv data backend/
mkdir -p backend/app/db
mv taro.db backend/app/db/
```

### 2. 修复模块导入问题
创建必须的 `__init__.py` 文件：
```bash
touch backend/app/__init__.py
touch backend/app/db/__init__.py
```

### 3. 配置文件修正
```diff
# backend/.env
- DATABASE_URL="sqlite:////home/.../taro.db"
+ DATABASE_URL="sqlite:///app/db/taro.db"
```

### 4. 脚本执行规范
```bash
# 必须在 backend 目录执行，设置 PYTHONPATH
cd backend
PYTHONPATH=. ./.venv/bin/python scripts/init_db.py
PYTHONPATH=. ./.venv/bin/python scripts/import_cards.py
```

## 关键验证点
1. 数据库初始化验证：
   `sqlite3 app/db/taro.db '.tables'` → 应显示 cards 表

2. 数据导入验证：
   `sqlite3 app/db/taro.db 'SELECT COUNT(*) FROM cards;'` → 非零结果

## 已知问题解决清单
| 问题现象 | 解决方案 |
|----------|----------|
| ModuleNotFoundError | 添加 `__init__.py` + 设置 `PYTHONPATH=.` |
| 数据文件路径错误 | 调整脚本中的 `../data` → `data` |
| 数据库路径错误 | 统一使用 `sqlite:///app/db/taro.db` |

## 预防措施
1. 所有路径使用相对于 `backend/` 的路径
2. 脚本头部添加路径修复：
   ```python
   import sys
   from pathlib import Path
   sys.path.append(str(Path(__file__).parent.parent))
   ```