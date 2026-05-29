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
13. 多语言

### 2. agent
1. 转 langgraph
2. 添加工具
3. 添加MCP
4. 3张牌逻辑不对
5. 3张牌只有一次请求，整理牌库
6. 流式输出 


### 2. 前端

1. 流式输出SSE fetch + ReadableStream
2. 刷新后原有的占卜结果就消失了
3. ai正在深度提醒的提示应该消失了
4. 3张牌逻辑不对
5. 历史记录是mock的


### 3. 架构

1. SQLite 转 MySQL
2. 【done】重构项目，将 .venv pyprojecy.toml data/ scripts/ 移动到backend目录下面
3. 【done】将 backend中的requirements.txt 删去
4. 【done】将pyproject.toml直接从根目录移动到backend的兼容问题
5. 【done】将 makefile 的 scripts 的命令修改
6. 【done】将 scripts 里的 import_cards.py 脚本修改

### 4. 运维
1. 【done】scripts 脚本运行失败
2. 

### 5. 测试
1. 测试用例

### 6. 文档
1. 前端功能是如何实现的
2. 整体架构

### 7. go重构
1. 重构后端
2. 手搓
3. 使用 eino 库

### 8. 完整python方案

### 9. go 网关 和go my-coding-

### 10. cli ws 
1. 流式输出
2. 添加注释 q 和 quit退出 这种
3. 
