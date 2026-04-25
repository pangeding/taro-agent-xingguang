# LangGraph重构技术文档

## 一、目标
将现有塔罗牌解读服务从函数式实现重构为LangGraph工作流，提升可维护性和可扩展性

## 二、架构设计
```
[用户请求] → FastAPI路由 → LangGraph工作流 → DashScope API
                        ↘ 基础解读备选
```

## 三、具体实施步骤

### 1. 添加依赖
```bash
# 修改 backend/requirements.txt
langgraph==0.1.0
```

### 2. 创建工作流状态
**文件**: `backend/app/agent/graph_state.py`
```python
from typing import TypedDict, Optional, Literal

class CardState(TypedDict):
    card_id: str
    question: str
    is_reversed: bool
    position: Optional[int]
    spread_type: str
    prompt: str
    api_response: Optional[str]
    status: Literal["success", "api_failed", "fallback_used"]
    fallback_reason: Optional[str]
```

### 3. 构建工作流节点
**文件**: `backend/app/agent/langgraph_agent.py`
```python
from langgraph.graph import StateGraph, END
from .graph_state import CardState
from .interpreter import call_dashscope_api, get_basic_interpretation

# 节点函数
async def build_prompt(state: CardState) -> CardState:
    # [此处实现build_prompt逻辑，参考原interpreter.py第58-106行]
    state['prompt'] = prompt
    return state

async def call_api(state: CardState) -> CardState:
    try:
        response = await call_dashscope_api(state['prompt'])
        state.update({
            'api_response': response,
            'status': 'success'
        })
    except Exception as e:
        state.update({
            'status': 'api_failed',
            'fallback_reason': str(e)
        })
    return state

async def handle_fallback(state: CardState) -> CardState:
    # 获取原始card对象（需从state.card_id查询数据库）
    card = ... # 伪代码：根据state.card_id获取TarotCard对象
    state['api_response'] = get_basic_interpretation(card, state['is_reversed'])
    state['status'] = 'fallback_used'
    return state

# 构建工作流
def create_interpretation_graph():
    builder = StateGraph(CardState)

    # 添加节点
    builder.add_node('build_prompt', build_prompt)
    builder.add_node('call_api', call_api)
    builder.add_node('handle_fallback', handle_fallback)

    # 设置入口点
    builder.set_entry_point('build_prompt')

    # 添加边
    builder.add_edge('build_prompt', 'call_api')
    builder.add_conditional_edges(
        'call_api',
        lambda state: state['status'],
        {
            'success': END,
            'api_failed': 'handle_fallback'
        }
    )
    builder.add_edge('handle_fallback', END)

    return builder.compile()
```

### 4. 更新API路由
**文件**: `backend/app/api/endpoints/tarot.py`
```python
# 替换原有interpret_card调用
from ..agent.langgraph_agent import create_interpretation_graph

interpretation_graph = create_interpretation_graph()

@router.post('/interpret')
async def interpret_card_route(...):
    initial_state = {
        'card_id': card_id,
        'question': question,
        'is_reversed': is_reversed,
        ...
    }
    final_state = await interpretation_graph.ainvoke(initial_state)
    return {'interpretation': final_state['api_response']}
```

### 5. 验证步骤
1. 确保`langgraph`已安装：`pip install -r backend/requirements.txt`
2. 测试单卡解读流程：正向路径和API失败备选路径
3. 验证工作流状态转换是否符合预期