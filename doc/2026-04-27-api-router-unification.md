# API 路由统一重构技术文档

## 背景

当前项目路由结构存在以下问题需要统一规范：

1. 各功能模块路由分散，缺乏统一的路由注册机制
2. 路由前缀需要在父级统一配置，避免子模块重复定义
3. 需要为后续更多模块（如用户、会话等）预留扩展能力

## 当前路由结构分析

### 现有文件结构
```
backend/app/
├── main.py                    # FastAPI 应用入口
├── api/
│   ├── __init__.py            # 主路由聚合
│   └── endpoints/
│       ├── cards.py           # 塔罗牌相关 API
│       └── readings.py        # 占卜相关 API
```

### 当前路由层级
```
/api/v1
  /readings         # readings 模块
    /               # POST - 创建占卜
    /{reading_id}   # GET  - 获取占卜详情
  /cards            # cards 模块
    /               # GET  - 获取所有塔罗牌
    /{card_id}      # GET  - 获取单张塔罗牌
    /random/        # GET  - 随机抽取塔罗牌
```

## 重构方案

### 目标路由结构
保持当前路由层级不变，但明确以下规范：

1. `api/__init__.py` 作为唯一的路由聚合入口
2. 每个 endpoint 文件只定义自己的 `router = APIRouter()`，不设置 prefix
3. prefix 统一在 `api/__init__.py` 中注册时设置
4. tags 也统一在 `api/__init__.py` 中设置

### 文件变更

#### 1. `backend/app/api/__init__.py`
保持不变，当前实现已符合规范：
```python
from fastapi import APIRouter
from .endpoints import readings, cards

router = APIRouter()

router.include_router(readings.router, prefix="/readings", tags=["占卜"])
router.include_router(cards.router, prefix="/cards", tags=["塔罗牌"])
```

#### 2. `backend/app/api/endpoints/cards.py`
无需变更，当前实现已正确：
- 使用 `router = APIRouter()` 创建路由
- 不在子模块设置 prefix
- 路径定义为 `/`, `/{card_id}`, `/random/`

最终路由：`/api/v1/cards/`, `/api/v1/cards/{card_id}`, `/api/v1/cards/random/`

#### 3. `backend/app/api/endpoints/readings.py`
无需变更，当前实现已正确。

最终路由：`/api/v1/readings/`, `/api/v1/readings/{reading_id}`

### 最终路由映射表

| 模块 | 方法 | 路径 | 描述 |
|------|------|------|------|
| cards | GET | `/api/v1/cards/` | 获取所有塔罗牌 |
| cards | GET | `/api/v1/cards/{card_id}` | 获取单张塔罗牌详情 |
| cards | GET | `/api/v1/cards/random/` | 随机抽取塔罗牌 |
| readings | POST | `/api/v1/readings/` | 创建新的占卜 |
| readings | GET | `/api/v1/readings/{reading_id}` | 获取占卜详情 |

## 扩展指南

当需要添加新模块时（例如 `users`）：

1. 在 `endpoints/` 下创建 `users.py`
2. 在 `users.py` 中定义 `router = APIRouter()`
3. 在 `api/__init__.py` 中添加：
   ```python
   from .endpoints import users
   router.include_router(users.router, prefix="/users", tags=["用户"])
   ```

## 注意事项

1. `/random/` 路径末尾的斜杠会导致 FastAPI 自动重定向 `/random` 到 `/random/`
   - 建议统一为 `/random`（无尾斜杠），或保持现状接受重定向行为
2. 所有路径参数（如 `{card_id}`）应在路由定义时明确类型（如 `{card_id: int}`）
3. tags 使用中文，便于 Swagger UI 展示
