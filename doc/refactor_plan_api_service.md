# API/Service 分离实施计划

## 一、现状分析

### 1. 当前结构
- API层文件位于 `backend/app/api/endpoints/`
  - `cards.py`：塔罗牌相关接口
  - `readings.py`：占卜相关接口
- 业务逻辑与API处理混合，缺乏分层设计

### 2. 问题点
1. **代码职责不清**
   - 数据库操作、业务逻辑与HTTP处理混合
   - 例如：`readings.py`中的`draw_cards`和`generate_interpretations`属于业务逻辑
2. **可测试性差**
   - 无法独立测试业务逻辑（依赖FastAPI请求上下文）
3. **复用性低**
   - 相同逻辑在多处重复（如随机抽牌逻辑）

## 二、分离方案

### 1. 目录结构调整
```
backend/app/
├── api/
│   └── endpoints/
│       ├── cards.py
│       └── readings.py
└── service/
    ├── __init__.py
    ├── card_service.py
    └── reading_service.py
```

### 2. 职责划分

#### (1) Service层职责
| 模块 | 新增方法 | 功能说明 |
|------|----------|----------|
| `card_service.py` | `get_all_cards()` | 获取所有塔罗牌数据 |
| | `get_card_by_id(card_id: int)` | 按ID获取单张牌 |
| | `get_random_card()` | 随机抽取一张牌（含正逆位） |
| `reading_service.py` | `create_reading(question, spread_type, session_id)` | 创建占卜记录核心逻辑 |
| | `draw_cards(count: int)` | 抽取指定数量的牌 |
| | `generate_interpretations(reading_id: int)` | 生成AI解读结果 |

#### (2) API层职责
- 仅保留HTTP接口定义
- 参数校验与转换
- 错误处理（HTTPException）
- 调用Service层方法
- 响应格式化

### 3. 具体实施步骤

#### 第一阶段：重构cards模块
1. 创建`backend/app/service/card_service.py`
   - 将`cards.py`中的业务逻辑迁移至此
   - 保留原始文件的数据库模型导入
2. 修改API层（`cards.py`）：
   ```python
   from ..service.card_service import get_all_cards, get_card_by_id, get_random_card

   @router.get("/", response_model=List[dict])
   async def get_all_cards_api():
       return await get_all_cards()
   ```
3. 验证：运行单元测试确保功能一致

#### 第二阶段：重构readings模块
1. 创建`backend/app/service/reading_service.py`
   - 迁移`draw_cards`和`generate_interpretations`
   - 提取`create_reading`核心流程
2. 重构API层（`readings.py`）：
   ```python
   from ..service.reading_service import ReadingService

   @router.post("/", response_model=ReadingResponse)
   async def create_reading_api(request: ReadingRequest):
       service = ReadingService()
       return await service.create_reading(
           question=request.question,
           spread_type=request.spread_type,
           session_id=request.session_id
       )
   ```
3. 特别注意：
   - `interpret_card`保持原路径导入（属于AI模块）
   - 事务处理需在Service层实现

#### 第三阶段：验证与测试
1. 编写Service层单元测试（覆盖核心逻辑）
2. 验证API端点功能与之前一致
3. 性能对比测试（重点关注DB查询优化）

## 三、注意事项
1. **依赖注入**：建议使用FastAPI的Depends实现Service注入
2. **事务管理**：复杂操作（如占卜创建）需添加事务装饰器
3. **错误处理**：Service层抛出领域异常，API层转换为HTTP错误

## 四、实施时间表
- Day 1：完成cards模块分离
- Day 2：完成readings模块分离
- Day 3：编写测试用例并验证