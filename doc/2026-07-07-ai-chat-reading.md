# 塔罗占卜AI对话功能 - 技术设计文档

## 1. 功能概述

实现一个类似ChatGPT/DeepSeek的对话式塔罗占卜界面，用户可以像与塔罗师对话一样进行多轮交互，获得塔罗牌解读和神秘学指导。

### 1.1 核心功能
- 多轮对话：支持连续提问和上下文记忆
- 塔罗抽牌：对话中可触发抽牌
- AI解读：基于LLM的个性化塔罗解读
- 对话历史：保存和加载历史对话
- 流式输出：实时显示AI回复（SSE）
- 神秘学知识库：整合塔罗牌含义、星座、牌阵等知识
- 简单用户系统：使用Cookie/LocalStorage存储user_id隔离用户数据

### 1.2 用户体验
- 类似DeepSeek/ChatGPT的聊天界面
- 左侧对话列表，右侧对话区域
- 支持新建对话、切换对话、删除对话
- 消息卡片支持Markdown渲染
- 抽牌动画效果
- 对话标题AI自动生成（第一轮对话后）

## 2. 技术架构

### 2.1 前端 (Next.js)
- 使用 `use client` 组件实现交互
- fetch + ReadableStream 进行SSE实现
- Tailwind CSS 样式
- Markdown渲染库 (react-markdown)
- Cookie/LocalStorage 管理user_id

### 2.2 后端 (Go)
- Gin框架提供HTTP API
- GORM操作SQLite数据库
- Server-Sent Events (SSE) 实现流式响应
- 复用现有的agent系统 (LLM调用、塔罗知识)
- 对话历史和消息持久化
- 基于user_id的数据隔离

### 2.3 用户系统（轻量级）
- 用户首次访问时自动生成UUID作为user_id
- user_id存储在Cookie和LocalStorage
- 所有API通过user_id隔离数据
- 无需密码登录，但支持"导出"用户数据

## 3. 数据库设计

### 3.1 新增模型

#### Conversation (对话会话)
```go
type Conversation struct {
    ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
    UserID    string    `gorm:"size:100;index" json:"user_id"`  // Cookie中的user_id
    Title     string    `gorm:"size:200;default:新对话" json:"title"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"autoCreateTime;autoUpdateTime" json:"updated_at"`
    Messages  []Message `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}
```

#### Message (对话消息)
```go
type Message struct {
    ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
    ConversationID uint      `gorm:"index" json:"conversation_id"`
    Role           string    `gorm:"size:20" json:"role"`        // user, assistant, system
    Content        string    `gorm:"type:text" json:"content"`
    Type           string    `gorm:"size:20;default:text" json:"type"` // text, cards, reading
    ReadingID      *uint     `json:"reading_id"`                   // 关联的占卜记录
    CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}
```

### 3.2 表关系
- Conversation 1:N Message
- Message 1:1 Reading (可选关联)

## 4. 后端API设计

### 4.1 用户 API

#### 获取/初始化用户ID
```
GET /api/v1/user/init
Response: { "user_id": "uuid-string" }
// 前端需要在响应后设置Cookie
```

### 4.2 对话管理 API

所有对话API请求头需要包含: `X-User-Id: <user_id>` 或从Cookie读取

#### 创建对话
```
POST /api/v1/conversations/
Body: { }  // 不需要title，AI会在第一轮对话后自动生成
Response: { "id": uint, "title": "新对话", "created_at": time }
```

#### 获取对话列表
```
GET /api/v1/conversations/
Response: [{ "id": uint, "title": string, "updated_at": time }]
```

#### 获取对话详情（含消息）
```
GET /api/v1/conversations/:id
Response: { "id": uint, "title": string, "messages": [...] }
```

#### 删除对话
```
DELETE /api/v1/conversations/:id
Response: { "success": true }
```

### 4.3 对话消息 API

#### 发送消息（流式输出）
```
POST /api/v1/conversations/:id/messages
Accept: text/event-stream
Body: { "content": "string" }
Response: SSE stream of { "delta": "string", "done": bool }
```

#### 获取对话消息
```
GET /api/v1/conversations/:id/messages?limit=50&offset=0
Response: [{ "id": uint, "role": string, "content": string, "type": string, "created_at": time }]
```

### 4.4 塔罗对话专用 API

#### 塔罗问答（在对话中触发抽牌）
```
POST /api/v1/conversations/:id/tarot
Accept: text/event-stream
Body: { 
    "question": "string",
    "spread_type": "single"  // single, three
}
Response: SSE stream with card draws, interpretations, synthesis
```

SSE事件格式：
```
event: card_drawn
data: {"name": "The Fool", "is_reversed": false, "position": 0}

event: interpretation
data: {"delta": "这张牌代表...", "card_index": 0}

event: done
data: {"reading_id": 123, "cards": [...]}
```

## 5. 后端实现细节

### 5.1 目录结构
```
backend-go/internal/
├── handler/
│   ├── conversation.go          # 对话相关handler
│   └── user.go                  # 用户初始化handler
├── service/
│   └── conversation_service.go  # 对话业务逻辑
├── middleware/
│   └── user_auth.go             # user_id中间件（从Cookie/Header提取）
└── db/
    └── models.go                # 新增Conversation, Message模型

backend-go/internal/agent/
├── chat_agent.go                # 新增：对话式AI agent
└── tarot_chat.go                # 新增：塔罗对话专用逻辑
```

### 5.2 ConversationService 方法
```go
type ConversationService struct {
    DB           *gorm.DB
    LLM          *agent.ChatClient
    ReadingSvc   *ReadingService
}

func (s *ConversationService) CreateConversation(userID string) (*db.Conversation, error)
func (s *ConversationService) ListConversations(userID string) ([]db.Conversation, error)
func (s *ConversationService) GetConversation(id uint, userID string) (*db.Conversation, error)
func (s *ConversationService) DeleteConversation(id uint, userID string) error
func (s *ConversationService) AddMessage(conversationID uint, role, content, messageType string) (*db.Message, error)
func (s *ConversationService) GetMessages(conversationID uint, limit, offset int) ([]db.Message, error)
func (s *ConversationService) GenerateTitle(conversationID uint) error  // AI生成标题
```

### 5.3 流式输出实现

使用SSE (Server-Sent Events)：

```go
func (s *ConversationService) StreamChat(ctx context.Context, convID uint, userID, content string, writer io.Writer) error {
    // 1. 保存用户消息
    // 2. 获取历史消息作为上下文
    // 3. 调用LLM流式接口
    // 4. 将每个token通过SSE发送
    // 5. 保存完整回复
    // 6. 如果是第一轮对话（2条消息），触发标题生成（异步）
}

func (s *ConversationService) StreamTarotReading(ctx context.Context, convID uint, userID, question, spreadType string, writer io.Writer) error {
    // 1. 获取历史对话上下文
    // 2. 抽牌
    // 3. 发送抽牌事件
    // 4. 流式生成解读
    // 5. 保存占卜记录和消息
    // 6. 如果是第一轮对话，触发标题生成
}

func (s *ConversationService) GenerateTitleAsync(conversationID uint) {
    // 异步调用LLM生成标题
    // 获取前两条消息的内容
    // 要求LLM生成一个简短标题（10-20字）
    // 更新conversation的title字段
}
```

### 5.4 Agent层扩展

#### ChatAgent
```go
type ChatAgent struct {
    LLM *agent.ChatClient
}

func (a *ChatAgent) CreateChatCompletion(ctx context.Context, messages []Message, stream bool) (string, error)
func (a *ChatAgent) CreateChatStream(ctx context.Context, messages []Message) (<-chan string, error)
func (a *ChatAgent) GenerateTitle(ctx context.Context, conversationHistory string) (string, error)
```

#### Tarot聊天System Prompt
使用塔罗专家人设，整合塔罗知识进行对话。

## 6. 前端实现细节

### 6.1 页面结构
```
frontend/app/chat/
├── page.jsx                 # 聊天主页面

frontend/components/chat/
├── Sidebar.jsx              # 对话列表侧边栏
├── ChatInput.jsx            # 聊天输入框
├── MessageList.jsx          # 消息列表
├── MessageBubble.jsx        # 单条消息
├── TarotCardDisplay.jsx     # 牌面展示组件
└── TypingIndicator.jsx      # AI正在输入指示器
```

### 6.2 状态管理

使用React useState + Context：

```jsx
const ChatContext = createContext()

function ChatProvider({ children }) {
    const [userId, setUserId] = useState(null)
    const [conversations, setConversations] = useState([])
    const [currentConversation, setCurrentConversation] = useState(null)
    const [messages, setMessages] = useState([])
    const [isLoading, setIsLoading] = useState(false)
    
    // 初始化用户
    const initUser = async () => { /* 设置user_id */ }
    
    // 方法
    const createConversation = async () => {}
    const loadConversation = async (id) => {}
    const sendMessage = async (content) => {}
    const sendTarotReading = async (question, spreadType) => {}
    const deleteConversation = async (id) => {}
    
    return (...)
}
```

### 6.3 SSE集成 (ReadableStream)

```jsx
async function streamResponse(url, body, headers, onChunk, onDone) {
    const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...headers },
        body: JSON.stringify(body)
    })
    
    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    
    while (true) {
        const { done, value } = await reader.read()
        if (done) break
        
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop()
        
        for (const line of lines) {
            if (line.startsWith('data: ')) {
                const data = JSON.parse(line.slice(6))
                if (data.done) {
                    onDone(data)
                } else {
                    onChunk(data.delta)
                }
            }
        }
    }
}
```

## 7. 实现步骤（按顺序执行）

### Phase 1: 后端基础 (Backend Foundation)
1. 新增数据库模型 Conversation, Message (models.go)
2. 创建user_id中间件
3. 创建 ConversationService (conversation_service.go)
4. 创建 ConversationHandler (conversation.go)
5. 创建 UserHandler (user.go)
6. 在 main.go 注册新路由

### Phase 2: 流式支持 (Streaming Support)
1. 扩展 LLM client 支持流式输出 (openai-go SDK已支持)
2. 实现 SSE handler
3. 实现 StreamChat 方法
4. 实现 StreamTarotReading 方法
5. 实现 GenerateTitleAsync 方法

### Phase 3: 前端基础 (Frontend Foundation)
1. 创建 /chat 页面
2. 创建 ChatSidebar 组件
3. 创建 ChatInput 组件
4. 创建 MessageList 和 MessageBubble 组件
5. 实现user_id管理

### Phase 4: 流式UI集成 (Streaming UI)
1. 实现 SSE 客户端集成 (ReadableStream)
2. 实现打字机效果
3. 实现 TarotCardDisplay 组件
4. 完善状态管理和错误处理

### Phase 5: 优化完善 (Polish)
1. Markdown 渲染支持
2. 响应式适配
3. 加载动画和过渡效果

## 8. 关键设计决策

| 项目 | 决策 |
|------|------|
| 用户系统 | 轻量级：Cookie/LocalStorage存储UUID，无需登录 |
| 对话标题 | AI自动生成：第一轮对话(2条消息)后异步生成 |
| 流式方案 | SSE (Server-Sent Events) |
| 前后端通信 | REST API + SSE流式响应 |
| 数据隔离 | 基于user_id |
