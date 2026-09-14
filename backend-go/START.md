# 快速启动指南 (QuickStart)

## 前置要求
- Go 1.22+
- MySQL 8+（默认数据库）
- GCC（仅回退到 SQLite 时需要，CGO 依赖 `mattn/go-sqlite3`）

## 1. 环境变量配置
```bash
cp .env.example .env
# 编辑 .env：填入 MYSQL_* 与 DASHSCOPE_API_KEY
```

数据库配置说明：
- `DB_DRIVER=mysql` 使用 MySQL；留空时若 `MYSQL_HOST` 非空也自动用 MySQL。
- 回退 SQLite：设置 `DB_DRIVER=sqlite`（使用 `DATABASE_URL` 指定的文件）。
- WSL2 访问 Windows 主机的 MySQL：`MYSQL_HOST` 填默认网关，用 `ip route show default | awk '{print $3}'` 复查（重启后可能变化）。

## 2. 准备 MySQL 数据库
```sql
CREATE DATABASE IF NOT EXISTS tarot CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```
确保 MySQL 监听 `0.0.0.0` 且防火墙放行 3306。

## 3. 初始化数据库并导入卡牌
建表由 GORM AutoMigrate 自动完成：
```bash
go run scripts/import_cards/main.go
```

## 4. 迁移 SQLite 历史数据（可选，一次性）
```bash
go run scripts/migrate_sqlite_to_mysql/main.go
# 可用 SQLITE_PATH 指定源文件，默认 data/tarot.db
```

## 5. 启动服务
默认监听 `:8000`：
```bash
go run cmd/server/main.go
```

## 6. 验证
```bash
curl http://localhost:8000/health
# 预期返回: {"status":"healthy"}

# 测试随机抽牌
curl http://localhost:8000/api/v1/cards/random/

# 测试占卜 (需配置 .env 中的 DASHSCOPE_API_KEY)
curl -X POST http://localhost:8000/api/v1/readings/langgraph \
  -H "Content-Type: application/json" \
  -d '{"question":"我最近运气怎么样？","spread_type":"single"}'
```
