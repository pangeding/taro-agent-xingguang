# API 开发流程 SOP
# 用不了，太垃圾了
## 触发条件
接到新 API 需求时执行。

## 流程

### 1. 分析需求
- 明确 endpoint、method、request/response schema
- 确定归属模块（新建 or 已有）

### 2. 写技术文档
- 路径: `doc/yyyy-mm-dd-<brief>.md`
- 必须包含:
  - 路由变更（最终路径映射表）
  - request/response Pydantic model 定义
  - DB model 变更（如有）
  - agent 层调用（如有）

### 3. 编码

#### 3a. 新建模块
```
backend/app/api/endpoints/<module>.py
  └─ router = APIRouter()
  └─ @router.<method>("<path>")

backend/app/api/__init__.py
  └─ from .endpoints import <module>
  └─ router.include_router(<module>.router, prefix="/<module>", tags=["<tag>"])
```

#### 3b. 已有模块
```
在已有 endpoints/<module>.py 中追加路由
```

#### 3c. 编码规范
- endpoint 文件内只定义 `router = APIRouter()`，不设 prefix
- prefix 统一在 `api/__init__.py` 设置
- path 不以 `/` 结尾（避免 307 重定向）
- path 参数指定类型: `/{id: int}`
- 使用 `HTTPException(status_code=404)` 处理不存在资源

### 4. 校验
- `curl` 或 `curl -X GET http://localhost:8000/api/v1/<path>` 验证
- 确认 `/docs` Swagger UI 中路由和 tags 正确
- 确认 DB 操作无异常

## 文件结构速查

```
backend/app/
├── main.py                  # FastAPI app, include_router(api_router, prefix="/api/v1")
├── api/
│   ├── __init__.py          # 路由聚合入口
│   └── endpoints/           # 各模块 endpoint
│       ├── cards.py
│       └── readings.py
├── agent/                   # AI 解读逻辑
├── db/
│   ├── base.py             # DB 连接
│   └── models.py           # Peewee models
└── core/
    └── config.py            # settings (API_V1_STR, etc.)
```

## 路由层级

```
/api/v1
  /<module>/                 # POST/GET list
  /<module>/{id}             # GET detail
```
