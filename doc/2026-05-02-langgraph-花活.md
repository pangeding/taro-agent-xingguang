LangGraph 缺失的重要特性

  1. Checkpoint / Memory（持久化） - 没有配置 checkpointer，无法实现：
    - 对话状态持久化（跨请求保存状态）
    - 断点续传/时间旅行（get_state_history）
    - 人工审核（human-in-the-loop）
  2. Streaming（流式输出） - 没有使用 astream() / astream_events()，无法实现：
    - 实时向用户推送解读进度
    - SSE 流式响应
  3. Retry / 错误恢复机制 - 没有使用 LangGraph 内置的重试策略（retry_policy），目前只是裸 try/except
  4. Subgraph（子图） - 没有利用子图来做模块化编排，如果未来需要更复杂的多 agent 协作会受限
  5. Tool Calling - LangGraph 原生支持 @tool 和工具路由，当前完全没有使用

  LangChain 缺失的重要特性

  1. ChatModel 抽象 - 自己用 openai.AsyncOpenAI 手写客户端，而不是用 ChatOpenAI / ChatDashScope，丢失了：
    - 内置的 retry/降级
    - token 统计（get_usage_metadata())
    - 结构化输出（with_structured_output）
  2. Prompt Templates - 用字符串拼接而非 ChatPromptTemplate，缺少：
    - 模板变量校验
    - 消息历史管理
  3. Callbacks - 没有使用 LangChain 的 callback 系统，无法实现：
    - 链路追踪（LangSmith）
    - 性能监控
    - 日志审计

  核心问题

  最大的问题是 LLM 客户端手写。使用 ChatOpenAI + with_structured_output 可以直接输出 Pydantic 模型，比现在手动拼接 prompt + 解析字符串要可靠得多。