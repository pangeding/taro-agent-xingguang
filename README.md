# 星光塔罗AI助手

基于传统塔罗牌智慧与现代人工智能技术的个性化占卜体验。

## 项目概述

星光是一款 AI 塔罗占卜助手，结合传统塔罗牌智慧与现代人工智能技术，为用户提供个性化的塔罗牌解读和占卜体验。

## 最小MVP功能

- ✅ 单张塔罗牌抽牌
- ✅ AI智能解读（基于DeepSeek API）
- ✅ 匿名占卜会话
- ✅ 简洁美观的Web界面
- ✅ 基础占卜历史记录

## 技术栈

### 后端
- Python 3.10+
- FastAPI (Web框架)
- SQLite (数据库，简化版)
- Peewee (ORM)
- DeepSeek API (AI解读)

### 前端
- Next.js 14 (React框架)
- Tailwind CSS (样式)
- Axios (HTTP客户端)

### 开发工具
- uv (Python包管理)
- pnpm (Node.js包管理)

## 快速开始

### 1. 环境准备

```bash
# 安装 Python 3.10+ 和 Node.js 16+
# 确保已安装 uv 和 pnpm
```

### 2. 后端设置

```bash
# 进入后端目录
cd backend

# 复制环境变量文件
cp .env.example .env

# 编辑 .env 文件，配置 DeepSeek API 密钥
# DEEPSEEK_API_KEY="your_api_key_here"

# 使用 uv 安装依赖
uv sync

# 初始化数据库
python ../scripts/init_db.py

# 导入塔罗牌数据
python ../scripts/import_cards.py

# 启动后端服务器
python run.py
```

后端将在 http://localhost:8000 运行，API文档在 http://localhost:8000/docs

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

- `GET /api/v1/cards/` - 获取所有塔罗牌
- `GET /api/v1/cards/random/` - 随机抽取一张牌
- `POST /api/v1/readings/` - 创建新的占卜
- `GET /api/v1/readings/{reading_id}` - 获取占卜详情

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
├── backend/                 # 后端服务
│   ├── app/
│   │   ├── api/            # API路由
│   │   ├── core/           # 核心配置
│   │   ├── db/             # 数据库模型
│   │   ├── agent/          # AI代理模块
│   │   └── main.py         # FastAPI应用入口
│   ├── requirements.txt    # Python依赖
│   ├── .env.example        # 环境变量示例
│   └── run.py             # 启动脚本
├── frontend/               # 前端应用
│   ├── app/                # Next.js app目录
│   ├── components/         # React组件
│   ├── package.json        # 前端依赖
│   └── tailwind.config.js  # Tailwind配置
├── data/                   # 塔罗牌数据
│   └── tarot_cards.json    # 22张大阿尔卡纳牌数据
├── scripts/                # 工具脚本
│   ├── init_db.py         # 数据库初始化
│   └── import_cards.py    # 数据导入脚本
└── BEGIN.md               # 项目规划文档
```

## 开发说明

### 数据库

项目使用 SQLite 简化部署，数据文件位于 `backend/taro.db`。

### AI解读

AI解读基于 DeepSeek API，需要配置有效的 API 密钥。解读过程是异步的，创建占卜后会在后台生成解读结果。

### 扩展计划

1. 添加更多塔罗牌数据（小阿尔卡纳）
2. 支持更多牌阵类型
3. 用户账户系统
4. 占卜分享功能
5. 移动端优化

## 注意事项

- 本产品仅供娱乐参考，请理性看待占卜结果
- 需要有效的 DeepSeek API 密钥才能使用AI解读功能
- 默认使用 SQLite，生产环境建议更换为 MySQL/PostgreSQL

## 许可证

MIT License