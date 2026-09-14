# 星光塔罗AI助手

基于传统塔罗牌智慧与现代人工智能技术的个性化占卜体验。

## 项目概述

星光是一款 AI 塔罗占卜助手，结合传统塔罗牌智慧与现代人工智能技术，为用户提供个性化的塔罗牌解读和占卜体验。

## 最小MVP功能

- ✅ 单张塔罗牌抽牌
- ✅ AI智能解读（基于 DashScope / OpenAI 兼容接口）
- ✅ 匿名占卜会话
- ✅ 简洁美观的Web界面
- ✅ 基础占卜历史记录

## 技术栈

### 后端 (backend-go)
- Go 1.22+
- Gin (Web框架)
- GORM (ORM，MySQL 驱动为主 + SQLite 回退驱动)
- MySQL 8+ (默认数据库) / SQLite (回退)
- CloudWeGo Eino (Agent 编排)
- DashScope API / OpenAI 兼容接口 (AI 解读)

### 前端 (frontend)
- Next.js 14 (React框架)
- Tailwind CSS (样式)
- Axios (HTTP客户端)

### 开发工具
- Go Modules (Go包管理)
- pnpm (Node.js包管理)

## 快速开始

### 1. 环境准备

```bash
# 安装 Go 1.22+、GCC（CGO 依赖）、Node.js 16+
# 确保已安装 pnpm
```

### 2. 后端设置 (backend-go)

```bash
# 进入后端目录
cd backend-go

# 复制环境变量文件
cp .env.example .env

# 编辑 .env，配置数据库与 DashScope API 密钥
# MYSQL_HOST / MYSQL_PORT / MYSQL_DB / MYSQL_USER / MYSQL_PASSWORD
# DASHSCOPE_API_KEY="your_api_key_here"

# 准备 MySQL 数据库（一次性）
# CREATE DATABASE IF NOT EXISTS tarot CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

# 下载依赖
go mod download

# 初始化数据库并导入塔罗牌数据（自动建表）
go run scripts/import_cards/main.go

# （可选）一次性迁移旧 SQLite 历史数据
go run scripts/migrate_sqlite_to_mysql/main.go

# 启动后端服务器
go run cmd/server/main.go
```

后端将在 http://localhost:8000 运行，健康检查 `GET /health` 返回 `{"status":"healthy"}`。

也可在项目根目录使用 Makefile：

```bash
make install-backend
make init-db
make backend
```

### 3. 前端设置

```bash
# 进入前端目录
cd frontend

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev
```

前端将在 http://localhost:3000 运行

## API接口

### 核心接口

- `GET /health` - 健康检查
- `GET /api/v1/cards/` - 获取所有塔罗牌
- `GET /api/v1/cards/{id}` - 获取单张牌详情
- `GET /api/v1/cards/random/` - 随机抽取一张牌
- `POST /api/v1/readings/` - 创建新的占卜
- `POST /api/v1/readings/langgraph` - 创建新的占卜（Eino 编排）
- `GET /api/v1/readings/{reading_id}` - 获取占卜详情
- `WS /api/v1/readings/ws` - WebSocket 占卜

### 占卜请求示例

```json
{
  "question": "我最近的工作运势如何？",
  "spread_type": "single",
  "session_id": "optional_session_id"
}
```

## 项目结构

```
taro_agent/
├── backend-go/              # Go 后端服务
│   ├── cmd/server/          # 服务入口
│   ├── internal/
│   │   ├── agent/           # Eino Agent 编排、提示词、LLM
│   │   ├── config/          # 环境变量配置
│   │   ├── db/              # GORM 模型与数据库连接
│   │   ├── handler/         # HTTP / WebSocket handlers
│   │   ├── middleware/      # CORS、用户中间件
│   │   └── service/         # 业务逻辑
│   ├── scripts/            # import_cards / migrate_sqlite_to_mysql
│   ├── data/                # tarot_cards.json、tarot.db(SQLite 回退/迁移源)
│   └── .env.example         # 环境变量示例
├── frontend/               # 前端应用
│   ├── app/                # Next.js app目录
│   ├── components/         # React组件
│   ├── package.json        # 前端依赖
│   └── tailwind.config.js  # Tailwind配置
├── doc/                    # 技术文档
└── Makefile                # 常用命令
```

## 开发说明

### 数据库

默认使用 MySQL 8+，表结构由 GORM AutoMigrate 在启动/导入脚本中自动创建。可通过 `DB_DRIVER` 在 MySQL 与 SQLite 之间切换，代码无需改动：

```
# MySQL（默认）
DB_DRIVER=mysql
MYSQL_HOST=172.23.96.1
MYSQL_PORT=3306
MYSQL_DB=tarot
MYSQL_USER=root
MYSQL_PASSWORD=your_password

# SQLite 回退
DB_DRIVER=sqlite
DATABASE_URL=data/tarot.db
```

历史 SQLite 数据可用 `go run scripts/migrate_sqlite_to_mysql/main.go` 一次性迁移。

### AI解读

AI解读基于 DashScope（OpenAI 兼容接口），需要配置有效的 API 密钥。`.env` 主要配置项：

```
DB_DRIVER=mysql
MYSQL_HOST=172.23.96.1
MYSQL_PORT=3306
MYSQL_DB=tarot
MYSQL_USER=root
MYSQL_PASSWORD=your_password
DASHSCOPE_API_KEY=your_key_here
DASHSCOPE_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
DASHSCOPE_MODEL=qwen-plus
BACKEND_CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```

### 扩展计划

1. 添加更多塔罗牌数据（小阿尔卡纳）
2. 支持更多牌阵类型
3. 用户账户系统
4. 占卜分享功能
5. 移动端优化

## 注意事项

- 本产品仅供娱乐参考，请理性看待占卜结果
- 需要有效的 DashScope API 密钥才能使用AI解读功能
- 默认使用 MySQL 8+，可通过 `DB_DRIVER` 回退到 SQLite

## 许可证

MIT License
