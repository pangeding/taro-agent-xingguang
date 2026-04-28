# 技术文档：将 API 分离为 API 层和 Service 层

## 日期
2026-04-28

## 背景

当前项目中，API endpoint 文件（`backend/app/api/endpoints/`）混合了路由定义和业务逻辑。需要将业务逻辑抽离到 Service 层，使 API 层只负责路由和请求/响应处理。

### 当前问题分析

1. `readings.py` 中包含大量业务逻辑：
   - 抽牌逻辑 `draw_cards()`
   - AI 解读生成逻辑 `generate_interpretations()`
   - 数据库操作散布在 endpoint 函数中

2. `cards.py` 中直接操作数据库模型：
   - 直接调用 `TarotCard.select()` / `TarotCard.get()`
   - 数据转换逻辑写在 endpoint 中

## 目标架构

```
backend/app/
├── api/                    # API 层（仅负责路由、参数校验、响应格式化）
│   ├── __init__.py
│   ├── cards.py            # 塔罗牌路由（从 endpoints/ 移上来）
│   └── readings.py         # 占卜路由（从 endpoints/ 移上来）
├── services/               # 新增：Service 层（业务逻辑）
│   ├── __init__.py
│   ├── card_service.py     # 塔罗牌相关业务逻辑
│   └── reading_service.py  # 占卜相关业务逻辑
├── model/                  # 新增：请求/响应模型
│   ├── __init__.py
│   ├── request.py          # 请求模型（Pydantic）
│   └── response.py         # 响应模型（Pydantic）
├── agent/                  # 不动
│   └── interpreter.py
├── core/
│   └── config.py
└── db/
    ├── base.py
    └── models.py           # 数据库模型（peewee）保持不变
```

## 变更详情

### 1. 新建 Service 层

#### 1.1 `backend/app/services/__init__.py`
空文件，标记为包。

#### 1.2 `backend/app/services/card_service.py`
职责：封装所有塔罗牌相关的业务逻辑

方法：
- `get_all_cards() -> list[dict]` - 获取所有牌，返回列表
- `get_card_by_id(card_id: int) -> dict | None` - 根据 ID 获取牌详情
- `get_random_card() -> dict` - 随机抽取一张牌（含正逆位判定）

实现：内部调用 `db.models.TarotCard` 模型，返回字典格式数据。

#### 1.3 `backend/app/services/reading_service.py`
职责：封装所有占卜相关的业务逻辑

方法：
- `create_reading(question: str, spread_type: str, session_id: str | None) -> dict` - 创建占卜，含抽牌和解读
- `get_reading_by_id(reading_id: int) -> dict | None` - 获取占卜详情
- `_draw_cards(count: int) -> list[dict]` - 随机抽牌（内部方法）
- `_generate_interpretations(reading_id: int)` - 生成 AI 解读（内部方法）

实现：内部调用 `db.models.Reading`、`ReadingCard`、`TarotCard` 模型，以及 `agent.interpreter.interpret_card`。

### 2. 已有 Model 层

#### 2.1 `backend/app/model/__init__.py`
空文件，标记为包。

#### 2.2 `backend/app/model/request.py`
存放所有 Pydantic 请求模型：

```python
class ReadingRequest(BaseModel):
    question: str
    spread_type: str = "single"
    session_id: str | None = None
```

#### 2.3 `backend/app/model/response.py`
存放所有 Pydantic 响应模型：

```python
class CardResponse(BaseModel):
    id: int
    name: str
    arcana_type: str
    suit: str | None
    number: int | None
    keywords: str
    image_url: str | None
    # ... 等

class ReadingCardResponse(BaseModel):
    card_id: int
    name: str
    position: int
    is_reversed: bool
    interpretation: str
    image_url: str | None

class ReadingResponse(BaseModel):
    reading_id: int
    session_id: str
    question: str
    spread_type: str
    created_at: datetime
    cards: list[ReadingCardResponse]
```

### 3. 重构 API 层

#### 3.1 删除 `backend/app/api/endpoints/` 目录
将文件直接移到 `api/` 下：
- `endpoints/cards.py` -> `api/cards.py`
- `endpoints/readings.py` -> `api/readings.py`

#### 3.2 `backend/app/api/cards.py`
变更后只保留：
- 路由定义 `@router.get("/")`, `@router.get("/{card_id}")`, `@router.get("/random/")`
- 参数接收和校验
- 调用 `card_service` 方法
- 返回响应（或抛出 HTTPException）

移除：
- 直接调用 `TarotCard` 模型
- 数据转换逻辑

#### 3.3 `backend/app/api/readings.py`
变更后只保留：
- 路由定义 `@router.post("/")`, `@router.get("/{reading_id}")`
- 调用 `reading_service` 方法
- 返回响应

移除：
- `ReadingRequest` / `ReadingResponse` 定义（移至 `model/response.py`）
- `draw_cards()` 函数
- `generate_interpretations()` 函数
- 直接调用 `Reading`、`ReadingCard`、`TarotCard` 模型

#### 3.4 更新 `backend/app/api/__init__.py`
更新导入路径（不再有 `endpoints` 子目录）。


## 执行步骤

1. 创建 `services/` 目录和 `__init__.py`
2. 创建 `model/` 目录和 `__init__.py`
3. 实现 `services/card_service.py`
4. 实现 `services/reading_service.py`
5. 创建 `model/request.py`
6. 创建 `model/response.py`
7. 删除 `api/endpoints/` 目录，将文件移至 `api/` 下
8. 重构 `api/cards.py`
9. 重构 `api/readings.py`
10. 更新 `api/__init__.py` 导入路径
11. 验证：运行应用，测试所有 endpoint
