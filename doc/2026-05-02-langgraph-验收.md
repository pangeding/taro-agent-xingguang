# LangGraph 验收技术文档

## 目标

1. 配置 langgraph-cli 本地开发服务器，使用 Studio UI 测试 graph
2. 将 reading_service 中的 langgraph 逻辑整合到 API 路由中

## 任务 1: langgraph-cli 本地 Studio 测试

### 1.1 添加依赖

在 `backend/pyproject.toml` 的 dependencies 中添加:
- `langgraph-cli[inmem]>=0.2.0`

人工执行 `uv sync` 安装

### 1.2 创建 langgraph.json 配置文件

在 `backend/` 根目录创建 `langgraph.json`:
```json
{
  "dependencies": ["."],
  "graphs": {
    "reading_graph": "./app/agent/graph.py:reading_graph"
  }
}
```

### 1.3 启动 Studio（人工执行）

```bash
cd backend
langgraph dev
```

- 人工访问浏览器 `http://localhost:2024` 打开 Studio UI
- 人工在 UI 中输入 JSON state 测试 graph 执行

### 1.4 验收标准（人工验证）

- 人工在 Studio UI 中确认 graph 加载成功
- 人工输入合法 state 确认返回正确解读
- 人工确认单牌/三牌阵执行路径正确

## 任务 2: reading_service 整合到 API

### 2.1 现状分析

- 当前 `reading_service.create_reading` 使用旧 `_generate_interpretations`（非 langgraph）
- `_generate_interpretations_langgraph` 已存在但未被调用
- 需要将 API 切换为使用 langgraph 版本

### 2.2 修改 `model/request.py`

- `ReadingRequest` 增加 `model_name: str | None = None` 字段

### 2.3 修改 `api/readings.py`

- `create_reading` 端点增加 `model_name` 可选参数
- 传递给 `reading_service.create_reading`

### 2.4 验收标准（人工验证）

- 人工通过 `/docs` Swagger 页面测试 POST `/api/v1/readings/`，确认使用 langgraph 生成解读
- 人工确认请求体可选传 `model_name`
- 人工确认返回结果结构不变

## 执行顺序

1. 任务 1.1 → 1.2 → 1.3（人工）→ 1.4（人工验证）
2. 任务 2.2 → 2.3 → 2.4（人工验证）
