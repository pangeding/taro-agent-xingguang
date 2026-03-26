### 1. 项目介绍

星光是一款 AI 塔罗占卜助手，结合传统塔罗牌智慧与现代人工智能技术，为用户提供个性化的塔罗牌解读和占卜体验。

### 2. 项目架构

```
project/
├── backend/          # 后端服务
│   ├── agent/        # AI 代理模块
│   ├── api/          # API 接口层
│   └── db/           # 数据库相关
├── frontend/         # 前端应用
├── data/            # 数据资源（塔罗牌知识库）
└── docs/            # 文档
```

**核心模块说明：**

- **backend/agent**: AI 代理核心，负责塔罗牌解读、对话交互、智能推荐
- **backend/api**: RESTful API 接口，提供用户管理、占卜记录、牌阵查询等服务
- **backend/db**: 数据库模型、迁移脚本、数据访问层
- **frontend**: 用户界面，支持网页端和移动端
- **data**: 78 张阿尔卡纳塔罗牌的详细知识库，包括牌义、象征、元素等

### 3. 技术栈

**AI 技术栈：**
- 大语言模型（LLM）集成
- 自然语言处理（NLP）
- 情感分析
- 个性化推荐算法

**后端技术栈：**
- 编程语言：Python 3.9+
- Web 框架：FastAPI / Flask
- 数据库：PostgreSQL / MySQL
- 缓存：Redis
- ORM：SQLAlchemy
- 认证：JWT

**前端技术栈：**
- 框架：React / Vue.js / Taro（跨平台）
- 状态管理：Redux / Pinia
- UI 组件：Ant Design / Element Plus
- 构建工具：Vite / Webpack
- 跨平台：Taro（支持小程序、H5、App）

**部署与运维：**
- 容器化：Docker
- 编排：Kubernetes
- CI/CD：GitHub Actions / GitLab CI
- 监控：Prometheus + Grafana

### 4. Agent 功能

**核心功能模块：**

1. **智能抽牌系统**
   - 支持多种经典牌阵（单张、三张、凯尔特十字等）
   - 随机抽牌算法
   - 正逆位判断

2. **深度解读引擎**
   - 基于牌面含义的基础解读
   - 结合用户问题的上下文分析
   - 多牌组合意义关联
   - 正逆位差异化解释

3. **个性化对话**
   - 自然语言问答
   - 追问与澄清
   - 情感共鸣与建议
   - 记忆用户历史占卜

4. **用户画像与推荐**
   - 记录用户占卜偏好
   - 分析用户关注领域
   - 推荐合适的牌阵
   - 定期运势提醒

5. **知识图谱**
   - 78 张塔罗牌详细信息
   - 牌与牌之间的关联
   - 元素、星座、数字对应关系
   - 神话原型与象征

### 5. 数据库设计

**核心表结构：**

```sql
-- 用户表
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE,
    password_hash VARCHAR(255),
    avatar_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_username (username),
    INDEX idx_email (email)
);

-- 塔罗牌表
CREATE TABLE tarot_cards (
    id INT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    arcana_type ENUM('major', 'minor') NOT NULL,
    suit ENUM('wands', 'cups', 'swords', 'pentacles') ,
    number INT,
    meaning_upright TEXT,      -- 正位含义
    meaning_reversed TEXT,     -- 逆位含义
    keywords VARCHAR(255),     -- 关键词
    element VARCHAR(20),       -- 元素
    zodiac_sign VARCHAR(20),   -- 星座
    image_url VARCHAR(255),
    description TEXT
);

-- 占卜记录表
CREATE TABLE readings (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT,
    question TEXT,             -- 用户问题
    spread_type VARCHAR(50),   -- 牌阵类型
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at)
);

-- 抽牌结果表
CREATE TABLE reading_cards (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    reading_id BIGINT,
    card_id INT,
    position INT,              -- 在牌阵中的位置
    is_reversed BOOLEAN,       -- 是否逆位
    interpretation TEXT,       -- AI 解读
    FOREIGN KEY (reading_id) REFERENCES readings(id),
    FOREIGN KEY (card_id) REFERENCES tarot_cards(id)
);

-- 用户反馈表
CREATE TABLE feedbacks (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    reading_id BIGINT,
    rating INT,                -- 评分 1-5
    comment TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (reading_id) REFERENCES readings(id)
);
```

### 6. 开发计划

**Phase 1: 基础建设（1-2 周）**
- [ ] 搭建项目骨架
- [ ] 完成数据库设计与初始化
- [ ] 实现基础 API 接口
- [ ] 导入 78 张塔罗牌数据

**Phase 2: 核心功能（2-3 周）**
- [ ] 实现 AI 解读引擎
- [ ] 开发抽牌系统
- [ ] 完成用户认证系统
- [ ] 基础前端页面

**Phase 3: 优化迭代（2-3 周）**
- [ ] 多种牌阵支持
- [ ] 对话系统优化
- [ ] 用户体验提升
- [ ] 性能优化

**Phase 4: 扩展功能（持续）**
- [ ] 社交分享功能
- [ ] 每日运势推送
- [ ] 付费咨询系统
- [ ] 多语言支持

### 7. 快速开始

**环境要求：**
- Python 3.9+
- Node.js 16+
- MySQL 8.0+ / PostgreSQL 13+
- Redis 6.0+

**安装步骤：**

```bash
# 克隆项目
git clone <repository-url>
cd taro_agent

# 后端安装
cd backend
pip install -r requirements.txt
cp .env.example .env
# 配置环境变量
python manage.py migrate
python manage.py runserver

# 前端安装
cd frontend
npm install
npm run dev
```

### 8. 注意事项

1. **数据安全**：用户隐私数据需加密存储
2. **合规性**：明确标注娱乐性质，避免迷信宣传
3. **性能**：高频抽牌场景需考虑缓存策略
4. **可扩展**：预留多 AI 模型切换能力
