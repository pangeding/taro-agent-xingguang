# 快速启动指南 (QuickStart)

## 前置要求
- Go 1.22+
- GCC (用于 `github.com/mattn/go-sqlite3` / CGO)

## 1. 环境变量配置
```bash
cp .env.example .env
# 编辑 .env，填入 DashScope API Key
```

## 2. 初始化数据库 (SQLite)
本服务使用 SQLite，数据库文件和表结构由脚本自动创建：
```bash
go run scripts/import_cards/main.go
```
执行后会生成 `data/tarot.db`。

## 3. 启动服务
默认监听 `:8000`：
```bash
go run cmd/server/main.go
```

## 4. 验证
```bash
curl http://localhost:8000/health
# 预期返回: {"status":"healthy"}

# 测试随机抽牌
curl http://localhost:8000/api/v1/cards/random/

# 测试占卜 (需配置 .env 中的 DASHSCOPE_API_KEY)
curl -X POST http://localhost:8000/api/v1/readings/langgraph \
  -H "Content-Type: application/json" \
  -d '{"question":"我最近运气怎么样？","spread_type":"single"}'
