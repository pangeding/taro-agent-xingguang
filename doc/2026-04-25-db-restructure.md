# 2026-04-25 技术文档：数据库路径重构至 backend

## 问题重定位
1. 用户明确要求：所有后端相关文件应统一在 backend 目录
2. 当前问题：
   - 根目录存在 taro.db（应移入 backend）
   - 配置文件路径混乱（.env 使用绝对路径）

## 重构方案
1. 创建数据库目录结构
   `mkdir -p backend/app/db`

2. 移动数据库文件
   `mv taro.db backend/app/db/`

3. 修正配置文件路径（统一使用相对路径）
   ```diff
   # backend/.env.example
   - DATABASE_URL="sqlite:///taro.db"
   + DATABASE_URL="sqlite:///app/db/taro.db"

   # backend/.env
   - DATABASE_URL="sqlite:////home/pangding/ai/taro_agent/taro.db"
   + DATABASE_URL="sqlite:///app/db/taro.db"
   ```

## 强制验证步骤
1. 检查数据库文件位置：
   `ls backend/app/db/taro.db`

2. 测试数据导入流程：
   ```bash
   cd backend
   python scripts/import_cards.py
   sqlite3 app/db/taro.db 'SELECT COUNT(*) FROM cards;'
   ```

## 回滚方案
```bash
mv backend/app/db/taro.db .
sed -i 's|app/db/taro.db|taro.db|g' backend/.env*
```