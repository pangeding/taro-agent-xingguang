# 2026-04-25 技术文档：数据库路径修正

## 问题定位
1. 根目录 tarot.db 必须保留，但 backend 脚本和应用指向错误路径
2. 现有路径结构：
   - 数据库实际位置：`/home/pangding/ai/taro_agent/tarot.db`
   - 错误引用位置：`backend/app/db/tarot.db`

## 诊断步骤
1. 验证脚本引用路径：
   `grep -r "tarot.db" backend/scripts/`

2. 验证应用配置路径：
   `grep -r "db_path" backend/app/db/`

## 修正方案
### 方案A（推荐）：修改应用配置（最小侵入）
1. 更新 backend/app/config.py 中的 DB_PATH:
   ```python
   # 原始错误配置
   DB_PATH = "db/tarot.db"

   # 修正为根目录路径
   DB_PATH = "../tarot.db"
   ```

### 方案B：修改脚本路径（需调整工作目录）
1. 修改 scripts 执行命令：
   ```bash
   cd .. && python backend/scripts/import_cards.py
   ```

## 验证流程
1. 测试数据导入：
   `python backend/scripts/import_cards.py`

2. 验证数据库更新：
   `sqlite3 tarot.db 'SELECT COUNT(*) FROM cards;'`

## 回滚方案
若修正失败，恢复原路径配置：
```bash
mv tarot.db backend/app/db/
```