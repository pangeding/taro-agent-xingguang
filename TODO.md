### 1. 后端

1. 添加支付功能 stripe
2. 【done】api service层分离
3. 多 llm 模型兼容
4. 【not needed】用 openai 代替 httpx 框架
5. 流式输出 SSE
6. 分离硬编码的工具 tool
7. tool 单牌阵和多牌阵
8. session 功能
9. 记忆功能 memory
10. 3张牌逻辑不对
11. 历史记录功能
12. 【done】各个功能怎么用一个 router？有在 init 里注册
13. 【done】多语言 已经转向go开发了
14. 没有用户隔离
15. 数据库隔离也不对
16. 超级管理员 能看到所有的

### 2. agent
1. 转 langgraph
2. 添加工具
3. 添加MCP
4. 3张牌逻辑不对
5. 3张牌只有一次请求，整理牌库
6. 流式输出 agent api cli ws
7. agent 占卜师功能 talk


### 2. 前端

1. 【partially done】流式输出SSE fetch + ReadableStream
2. 三张牌的占卜还是不行
2. 【done】刷新后原有的占卜结果就消失了
3. 【done】ai正在深度提醒的提示应该消失了
4. 3张牌逻辑不对
5. 【done】历史记录是mock的
6. agent 占卜师功能 talk
7. 单张解读出现了2遍，多张牌阵 似乎是有必要的
8. 对话界面的牌似乎没有了，很不对
9. 占卜信息怎么混到对话里面去了，很不对的


### 3. 架构

1. SQLite 转 MySQL

### 4. 运维
1. 【done】scripts 脚本运行失败
2. 

### 5. 测试
1. 测试用例

### 6. 文档
1. 前端功能是如何实现的
2. 整体架构
