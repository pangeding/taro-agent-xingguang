# 星光塔罗AI助手

基于传统塔罗牌智慧与现代人工智能技术的个性化占卜体验。

## 项目概述

星光是一款 AI 塔罗占卜助手，结合传统塔罗牌智慧与现代人工智能技术，为用户提供个性化的塔罗牌解读和占卜体验。

## 最小MVP功能

- ✅ 单张塔罗牌抽牌
- ✅ AI智能解读（基于 DashScope / OpenAI 兼容接口）
- ✅ 用户账号系统（登录 / 改密 / 数据隔离）
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

首次启动会自动创建开发用引导账号 **admin / 123456**（仅当 `APP_ENV=dev`，且 users 表为空）。
生产环境请见下方「用户系统」。

已有历史数据需先执行一次数据迁移（把旧数据归入 admin 名下）：

```bash
# 先预览将要执行的 SQL，确认无误后再正式执行
DRY_RUN=1 go run scripts/migrate_user_system/main.go
go run scripts/migrate_user_system/main.go
```

⚠️ 迁移脚本必须在部署带认证的新版后端**之前**跑完，否则新代码按 `owner_id`
过滤而列还没回填，登录后会看到 0 条历史。

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

## 用户系统

### 认证方式

服务端会话表 + `HttpOnly` Cookie（`tarot_session`），**不使用 JWT**。
选它的关键原因是 `WS /api/v1/readings/ws` 存在——浏览器 `WebSocket` 构造函数
无法自定义请求头，Cookie 会被自动携带，而 Bearer 方案只能退化成 `?token=` 查询参数。

- 口令用 bcrypt（cost 12）存储，会话 token 只存 sha256，明文仅存在于 Cookie。
- 连续 5 次登录失败锁定 15 分钟。
- 改密后除当前会话外的所有会话失效。

### 账号管理（不开放自助注册）

```bash
go run scripts/admin_user/main.go list
go run scripts/admin_user/main.go create --username alice --password 'Str0ng-Passw0rd' --role user
go run scripts/admin_user/main.go reset-password --username alice --password 'New-Passw0rd'
go run scripts/admin_user/main.go rename --from admin --to myuser
go run scripts/admin_user/main.go disable --username alice
```

### 数据隔离

`conversations` / `readings` / `feedbacks` 均有 `owner_id`，全部查询在服务层按
`auth.Actor.VisibleTo()` 过滤。管理员**默认也只能看到自己的数据**，跨用户查看需显式
加 `?scope=all`。

### 生产部署

`APP_ENV=prod` 会启用三重保护，任一不过即**拒绝启动**：

1. **L1 引导账号校验**：`BOOTSTRAP_ADMIN_USERNAME` / `BOOTSTRAP_ADMIN_PASSWORD` 必填；
   用户名不得为 `admin`/`root`/`administrator`/`test`/`guest`/`user` 等保留字；
   口令长度 ≥ 12 且不得命中弱口令表。
2. **L2 存量弱口令扫描**：启动时用 bcrypt 逐个比对已知弱口令，命中即拒绝启动
   （防止把 dev 库直接搬到生产）。
3. **L3 强制改密门禁**：`must_change_password=true` 的账号，除
   `/auth/me`、`/auth/logout`、`/auth/password` 外的接口一律 403 `password_change_required`。

从 dev 切到 prod 的操作步骤：

```bash
# 1. 改掉默认用户名（admin 是保留字，prod 会拒绝启动）
go run scripts/admin_user/main.go rename --from admin --to <你的用户名>
# 2. 设置强口令
go run scripts/admin_user/main.go reset-password --username <你的用户名> --password '<强口令>'
# 3. .env 中设置 APP_ENV=prod 与 BOOTSTRAP_ADMIN_USERNAME / BOOTSTRAP_ADMIN_PASSWORD
```

生产环境要求：`BIND_ADDR` 默认绑 `127.0.0.1:8000`，由 nginx 做 TLS 终止并反代；
`COOKIE_SECURE` 默认 `true`。建议另建只授 `tarot` 库 DML 权限的 MySQL 账号，
不要用 `root` 连库。

## API接口

除 `POST /api/v1/auth/login` 外，`/api/v1/*` 全部需要登录，未登录返回 401。

### 核心接口

- `GET /health` - 健康检查
- `POST /api/v1/auth/login` - 登录（唯一无需认证的接口）
- `POST /api/v1/auth/logout` - 登出
- `GET /api/v1/auth/me` - 当前用户
- `POST /api/v1/auth/password` - 修改密码
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
│   │   ├── auth/            # Actor 归属过滤、口令哈希、会话 token
│   │   ├── bootstrap/       # 启动期账号引导与生产保护
│   │   ├── config/          # 环境变量配置
│   │   ├── db/              # GORM 模型与数据库连接
│   │   ├── handler/         # HTTP / WebSocket handlers
│   │   ├── middleware/      # CORS、认证中间件
│   │   └── service/         # 业务逻辑
│   ├── scripts/            # import_cards / migrate_sqlite_to_mysql
│   │                       # migrate_user_system / admin_user
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
3. ~~用户账户系统~~ ✅ 已完成
4. 占卜分享功能
5. 移动端优化

## 注意事项

- 本产品仅供娱乐参考，请理性看待占卜结果
- 需要有效的 DashScope API 密钥才能使用AI解读功能
- 默认使用 MySQL 8+，可通过 `DB_DRIVER` 回退到 SQLite

## 许可证

MIT License
