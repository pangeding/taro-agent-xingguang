# 塔罗 AI 助手 Go 版本迁移技术文档

## 1. 项目概述

将 `backend/`（Python/FastAPI/LangGraph/SQLite）完整迁移为 `backend-go/`（Go/Gin/Eino/SQLite），保持 API 兼容、功能一致。

---

## 2. 技术栈映射

| 层级 | Python (backend) | Go (backend-go) |
|------|------------------|-----------------|
| Web 框架 | FastAPI | Gin |
| 数据库 ORM | Peewee + SQLite | GORM + mattn/go-sqlite3 |
| AI 框架 | LangGraph | CloudWeGo Eino (compose + adk) |
| LLM 客户端 | openai.AsyncOpenAI | github.com/openai/openai-go + DashScope (OpenAI 兼容) |
| 配置管理 | pydantic-settings + dotenv | github.com/caarlos0/env/v11 |
| 数据验证 | Pydantic | 自定义 struct + 校验 |

---

## 3. 项目目录结构

```
backend-go/
├── go.mod
├── go.sum
├── .env.example
├── data/
│   └── tarot_cards.json          (从 backend/data/ 复制)
├── cmd/
│   └── server/
│       └── main.go               (入口，初始化并启动服务)
├── internal/
│   ├── config/
│   │   └── config.go             (环境变量配置)
│   ├── db/
│   │   ├── database.go           (数据库连接、初始化)
│   │   └── models.go             (GORM 模型定义)
│   ├── handler/
│   │   ├── card.go               (塔罗牌 HTTP handlers)
│   │   ├── reading.go            (占卜 HTTP handlers + WebSocket)
│   │   └── health.go             (健康检查)
│   ├── service/
│   │   ├── card_service.go       (塔罗牌业务逻辑)
│   │   └── reading_service.go    (占卜业务逻辑 + 抽牌 + Eino 编排)
│   ├── agent/
│   │   ├── graph.go              (Eino compose graph: interpret -> synthesize)
│   │   ├── llm.go                (LLM ChatModel 配置)
│   │   ├── prompt.go             (提示词模板)
│   │   └── fallback.go           (API 失败时的基础解读)
│   └── middleware/
│       └── cors.go               (CORS 中间件)
└── scripts/
    └── import_cards/
        └── main.go               (从 JSON 导入塔罗牌到数据库)
```

---

## 4. 数据库模型 (GORM)

### 4.1 TarotCard

```go
type TarotCard struct {
    ID             uint   `gorm:"primaryKey;autoIncrement"`
    Name           string `gorm:"size:50;uniqueIndex"`
    ArcanaType     string `gorm:"size:10"`      // "major" / "minor"
    Suit           *string `gorm:"size:20"`     // "wands","cups","swords","pentacles" 或 NULL
    Number         *int
    MeaningUpright string `gorm:"type:text"`
    MeaningReversed string `gorm:"type:text"`
    Keywords       string `gorm:"size:255"`
    Element        *string `gorm:"size:20"`
    ZodiacSign     *string `gorm:"size:20"`
    ImageURL       *string `gorm:"size:255"`
    Description    string `gorm:"type:text"`
}
TableName: "tarot_cards"
```

### 4.2 Reading

```go
type Reading struct {
    ID         uint       `gorm:"primaryKey;autoIncrement"`
    SessionID  string     `gorm:"size:100;index"`
    Question   string     `gorm:"type:text"`
    SpreadType string     `gorm:"size:50;default:single"`
    CreatedAt  time.Time  `gorm:"index;autoCreateTime"`
    Cards      []ReadingCard `gorm:"foreignKey:ReadingID"`
}
TableName: "readings"
```

### 4.3 ReadingCard

```go
type ReadingCard struct {
    ID             uint   `gorm:"primaryKey;autoIncrement"`
    ReadingID      uint   `gorm:"index"`
    CardID         uint
    Position       int    `gorm:"default:0"`
    IsReversed     bool   `gorm:"default:false"`
    Interpretation string `gorm:"type:text"`
    Card           TarotCard `gorm:"foreignKey:ID;references:CardID"`
}
TableName: "reading_cards"
```

### 4.4 Feedback

```go
type Feedback struct {
    ID        uint      `gorm:"primaryKey;autoIncrement"`
    ReadingID uint      `gorm:"index"`
    Rating    int
    Comment   *string
    CreatedAt time.Time `gorm:"autoCreateTime"`
}
TableName: "feedbacks"
```

---

## 5. 配置 (internal/config/config.go)

```go
type Settings struct {
    APIV1Str           string   `env:"API_V1_STR" envDefault:"/api/v1"`
    ProjectName        string   `env:"PROJECT_NAME" envDefault:"星光塔罗AI助手"`
    Version            string   `env:"VERSION" envDefault:"0.1.0"`
    DatabaseURL        string   `env:"DATABASE_URL" envDefault:"data/taro.db"`
    DashScopeAPIKey    string   `env:"DASHSCOPE_API_KEY"`
    DashScopeBaseURL   string   `env:"DASHSCOPE_BASE_URL" envDefault:"https://dashscope.aliyuncs.com/compatible-mode/v1"`
    DashScopeModel     string   `env:"DASHSCOPE_MODEL"`
    BackendCORSOrigins []string `env:"BACKEND_CORS_ORIGINS" envDefault:"http://localhost:3000,http://127.0.0.1:3000"`
}
```

.env.example:
```
DATABASE_URL=data/taro.db
DASHSCOPE_API_KEY=your_key_here
DASHSCOPE_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
DASHSCOPE_MODEL=qwen-plus
BACKEND_CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```

注意：DashScope 的 OpenAI 兼容 endpoint 使用 `/compatible-mode/v1` 路径。

---

## 6. Eino Agent 编排 (internal/agent/)

### 6.1 状态定义

LangGraph 的 ReadingState 映射为 Go struct：

```go
type CardInterpretation struct {
    CardID        uint
    CardName      string
    IsReversed    bool
    Position      int
    Interpretation string
    Status        string // "success" | "fallback"
}

type ReadingState struct {
    Question       string
    SpreadType     string // "single" | "three"
    SessionID      string
    ModelName      string
    CardsInfo      []CardInfo          // [{ID, Name, IsReversed, Position, Card}]
    Interpretations []CardInterpretation
    Synthesis      string
    Status         string // "success" | "partial" | "failed"
    Error          string
}

type CardInfo struct {
    ID       uint
    Name     string
    IsReversed bool
    Position int
    Card     *db.TaroCard
}
```

### 6.2 LLM 封装 (internal/agent/llm.go)

使用 `github.com/cloudwego/eino-ext/components/model/openai` 创建 ChatModel：

```go
func NewChatModel(modelName, apiKey, baseURL string) (*openai.ChatModel, error) {
    if modelName == "" {
        modelName = config.Settings.DashScopeModel
    }
    return openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:    modelName,
        APIKey:   apiKey,
        BaseURL:  baseURL,
        Temperature: float32(0.7),
        MaxTokens: 1000,
    })
}
```

### 6.3 Graph 构建 (internal/agent/graph.go)

使用 `compose.NewGraph` 构建与 LangGraph 等价的工作流：

```go
func CreateReadingGraph(chatModel *openai.ChatModel) (compose.Runnable, error) {
    graph := compose.NewGraph[*ReadingState, *ReadingState]()

    // interpret 节点：对每张牌调用 LLM 生成解读
    graph.AddLambdaNode("interpret", compose.InvokableLambdaWithOption(interpretCardsFunc(chatModel)))

    // synthesize 节点：对三张牌进行综合解读（仅 spread_type="three" 时执行）
    graph.AddLambdaNode("synthesize", compose.InvokableLambdaWithOption(synthesizeFunc(chatModel)))

    // 边
    graph.AddEdge(compose.START, "interpret")
    graph.AddBranch("interpret", compose.NewGraphBranch(func(ctx context.Context, state *ReadingState) (string, error) {
        if state.SpreadType == "three" {
            return "synthesize", nil
        }
        return compose.END, nil
    }, map[string]bool{compose.END: true, "synthesize": true}))
    graph.AddEdge("synthesize", compose.END)

    return graph.Compile(ctx)
}
```

### 6.4 interpret 节点实现

```go
func interpretCardsFunc(chatModel *openai.ChatModel) compose.InvokableLambdaWithOptionHandler[*ReadingState, *ReadingState] {
    return func(ctx context.Context, state *ReadingState, opts ...any) (*ReadingState, error) {
        for _, ci := range state.CardsInfo {
            prompt := BuildCardPrompt(ci.Card, ci.IsReversed, state.Question, ci.Position, state.SpreadType)
            resp, err := chatModel.Generate(ctx, []*schema.Message{
                schema.SystemMessage(SYSTEM_PROMPT),
                schema.UserMessage(prompt),
            })
            if err != nil {
                state.Interpretations = append(state.Interpretations, CardInterpretation{
                    CardID: ci.ID, CardName: ci.Name, IsReversed: ci.IsReversed,
                    Position: ci.Position, Interpretation: GetBasicInterpretation(ci.Card, ci.IsReversed),
                    Status: "fallback",
                })
            } else {
                state.Interpretations = append(state.Interpretations, CardInterpretation{
                    CardID: ci.ID, CardName: ci.Name, IsReversed: ci.IsReversed,
                    Position: ci.Position, Interpretation: strings.TrimSpace(resp.Message.Content),
                    Status: "success",
                })
            }
        }
        return state, nil
    }
}
```

### 6.5 synthesize 节点实现

```go
func synthesizeFunc(chatModel *openai.ChatModel) compose.InvokableLambdaWithOptionHandler[*ReadingState, *ReadingState] {
    return func(ctx context.Context, state *ReadingState, opts ...any) (*ReadingState, error) {
        prompt := BuildSynthesisPrompt(state.Interpretations, state.Question)
        resp, err := chatModel.Generate(ctx, []*schema.Message{
            schema.SystemMessage(SYSTEM_PROMPT),
            schema.UserMessage(prompt),
        })
        if err != nil {
            state.Synthesis = "（综合分析暂时不可用）"
            state.Status = "partial"
            state.Error = err.Error()
        } else {
            state.Synthesis = strings.TrimSpace(resp.Message.Content)
        }
        return state, nil
    }
}
```

### 6.6 提示词 (internal/agent/prompt.go)

```go
const SYSTEM_PROMPT = "你是一位专业、富有洞察力的塔罗牌解读师，擅长结合牌面含义和用户具体问题提供深入、个性化的解读。你的解读充满智慧、共情和启发性。"

func BuildCardPrompt(card *db.TaroCard, isReversed bool, question string, position int, spreadType string) string { ... }
func BuildSynthesisPrompt(interps []CardInterpretation, question string) string { ... }
```

逻辑等同 Python 版 `prompts.py`。

### 6.7 Fallback (internal/agent/fallback.go)

```go
func GetBasicInterpretation(card *db.TaroCard, isReversed bool) string { ... }
```

逻辑等同 Python 版 `fallback.py`。

---

## 7. 业务逻辑 (internal/service/)

### 7.1 Card Service (internal/service/card_service.go)

```go
type CardService struct {
    DB *gorm.DB
}

func (s *CardService) GetAllCards() []CardItem { ... }       // SELECT * FROM tarot_cards
func (s *CardService) GetCardByID(id uint) (*CardDetail, error) { ... }
func (s *CardService) GetRandomCard() (*RandomCardItem, error) { ... }  // ORDER BY RANDOM() LIMIT 1
```

### 7.2 Reading Service (internal/service/reading_service.go)

```go
type ReadingService struct {
    DB      *gorm.DB
    Graph   compose.Runnable  // 编译后的 Eino graph
}

func (s *ReadingService) DrawCards(count int) ([]DrawnCard, error) { ... }

func (s *ReadingService) CreateReading(question, spreadType string, sessionID *string) (*ReadingResult, error) {
    // 1. 创建 Reading 记录
    // 2. 抽牌并创建 ReadingCard 记录
    // 3. 调用 Eino graph 生成解读
    // 4. 保存解读结果到 ReadingCard
    // 5. 返回结果
}

func (s *ReadingService) CreateReadingLangGraph(question, spreadType string, sessionID, modelName *string) (*ReadingResult, error) {
    // 等同 CreateReading，但使用 Eino graph 编排 LLM 调用
    // 构建 ReadingState -> graph.Invoke -> 保存结果
}

func (s *ReadingService) GetReadingByID(id uint) (*ReadingResult, error) { ... }
```

### 7.3 CreateReadingLangGraph 核心流程

```go
func (s *ReadingService) CreateReadingLangGraph(question, spreadType string, sessionID, modelName *string) (*ReadingResult, error) {
    // 1. 生成 session_id
    sid := uuid.New().String()
    if sessionID != nil { sid = *sessionID }

    // 2. 创建 Reading
    reading := db.Reading{SessionID: sid, Question: question, SpreadType: spreadType}
    s.DB.Create(&reading)

    // 3. 抽牌
    drawnCards, _ := s.DrawCards(cardCount(spreadType))

    // 4. 创建 ReadingCard
    for i, dc := range drawnCards {
        s.DB.Create(&db.ReadingCard{
            ReadingID: reading.ID, CardID: dc.ID, Position: i, IsReversed: dc.IsReversed,
        })
    }

    // 5. 构建 Eino state
    state := &agent.ReadingState{
        Question: question, SpreadType: spreadType, SessionID: sid, ModelName: orDefault(modelName, ""),
        CardsInfo: cardsInfo, // 关联完整 TarotCard 对象
    }

    // 6. 调用 graph
    result, err := s.Graph.Invoke(context.Background(), state)

    // 7. 保存解读
    for i, interp := range result.Interpretations {
        s.DB.Model(&db.ReadingCard{}).Where("id = ?", cardIDs[i]).Update("interpretation", interp.Interpretation)
    }
    if result.Synthesis != "" && spreadType == "three" {
        // 追加到最后一条 ReadingCard
    }

    // 8. 查询并返回完整结果
    return s.GetReadingByID(reading.ID)
}
```

---

## 8. HTTP Handlers (internal/handler/)

### 8.1 请求/响应结构

```go
type ReadingRequest struct {
    Question   string  `json:"question" binding:"required"`
    SpreadType string  `json:"spread_type"`
    SessionID  *string `json:"session_id"`
    ModelName  *string `json:"model_name"`
}
// JSON 默认值: spread_type="single"

type CardItem struct {
    ID        uint    `json:"id"`
    Name      string  `json:"name"`
    ArcanaType string `json:"arcana_type"`
    Suit      *string `json:"suit"`
    Number    *int    `json:"number"`
    Keywords  string  `json:"keywords"`
    ImageURL  *string `json:"image_url"`
}

type ReadingCardResponse struct {
    CardID       uint    `json:"card_id"`
    Name         string  `json:"name"`
    Position     int     `json:"position"`
    IsReversed   bool    `json:"is_reversed"`
    Interpretation string `json:"interpretation"`
    ImageURL     *string `json:"image_url"`
}

type ReadingResponse struct {
    ReadingID  uint                    `json:"reading_id"`
    SessionID  string                  `json:"session_id"`
    Question   string                  `json:"question"`
    SpreadType string                  `json:"spread_type"`
    CreatedAt  time.Time               `json:"created_at"`
    Cards      []ReadingCardResponse   `json:"cards"`
}
```

### 8.2 Card Handler (internal/handler/card.go)

| Method | Path | Handler | 说明 |
|--------|------|---------|------|
| GET | `/api/v1/cards/` | `GETAllCards` | 所有塔罗牌列表 |
| GET | `/api/v1/cards/:id` | `GetCard` | 单张牌详情 |
| GET | `/api/v1/cards/random/` | `GetRandomCard` | 随机抽牌 |

- `GETAllCards`: 调用 CardService.GetAllCards()，返回 JSON 数组
- `GetCard`: 调用 CardService.GetCardByID(id)，404 时返回 `{"error":"牌不存在"}`
- `GetRandomCard`: 调用 CardService.GetRandomCard()，404 时返回 `{"error":"暂无塔罗牌数据"}`

### 8.3 Reading Handler (internal/handler/reading.go)

| Method | Path | Handler | 说明 |
|--------|------|---------|------|
| POST | `/api/v1/readings/` | `CreateReading` | 创建占卜（传统方式） |
| POST | `/api/v1/readings/langgraph` | `CreateReadingLangGraph` | 创建占卜（Eino graph） |
| GET | `/api/v1/readings/:id` | `GetReading` | 获取占卜详情 |
| WS | `/api/v1/readings/ws` | `WebSocketReading` | WebSocket 占卜 |

`CreateReading` 和 `CreateReadingLangGraph` 调用对应的 service 方法，返回 `ReadingResponse` JSON。

`GetReading`: 404 时返回 `{"error":"占卜记录不存在"}`。

### 8.4 WebSocket Handler

```go
func WebSocketReading(c *gin.Context) {
    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil { return }
    defer conn.Close()

    var sessionID string
    for {
        _, msg, err := conn.ReadMessage()
        if err != nil { break }

        var req struct {
            Question   string  `json:"question"`
            SpreadType string  `json:"spread_type"`
            ModelName  *string `json:"model_name"`
            SessionID  *string `json:"session_id"`
        }
        json.Unmarshal(msg, &req)

        sid := req.SessionID
        if sid == nil { sid = &sessionID }

        result, err := service.CreateReadingLangGraph(req.Question, req.SpreadType, sid, req.ModelName)
        if err != nil {
            conn.WriteJSON(map[string]string{"error": err.Error(), "status": "error"})
        } else {
            sessionID = result.SessionID
            conn.WriteJSON(result)
        }
    }
}
```

### 8.5 Health Handler

| Method | Path | Handler | 说明 |
|--------|------|---------|------|
| GET | `/` | `Root` | 返回欢迎信息 |
| GET | `/health` | `HealthCheck` | 返回 `{"status":"healthy"}` |

---

## 9. 路由注册 (cmd/server/main.go)

```go
func main() {
    // 1. 加载配置
    cfg := config.Load()

    // 2. 初始化数据库
    db := database.Init(cfg.DatabaseURL)
    database.AutoMigrate(db)

    // 3. 初始化 Eino graph
    chatModel := agent.NewChatModel("", cfg.DashScopeAPIKey, cfg.DashScopeBaseURL)
    graph := agent.CreateReadingGraph(chatModel)

    // 4. 初始化 service
    cardService := service.NewCardService(db)
    readingService := service.NewReadingService(db, graph)

    // 5. 初始化 Gin
    r := gin.Default()
    r.Use(middleware.CORS(cfg.BackendCORSOrigins))

    // 6. 注册路由
    r.GET("/", handler.Root(cfg))
    r.GET("/health", handler.HealthCheck)

    v1 := r.Group(cfg.APIV1Str)
    {
        cards := v1.Group("/cards")
        cards.GET("/", handler.GETAllCards(cardService))
        cards.GET("/:id", handler.GetCard(cardService))
        cards.GET("/random/", handler.GetRandomCard(cardService))

        readings := v1.Group("/readings")
        readings.POST("/", handler.CreateReading(readingService))
        readings.POST("/langgraph", handler.CreateReadingLangGraph(readingService))
        readings.GET("/:id", handler.GetReading(readingService))
        readings.GET("/ws", handler.WebSocketReading(readingService))
    }

    // 7. 启动
    r.Run(":8000")
}
```

---

## 10. 数据库初始化脚本

### scripts/import_cards/main.go

从 `data/tarot_cards.json` 读取数据并插入 SQLite：

```go
func main() {
    db := database.Init("data/taro.db")
    database.AutoMigrate(db)

    data, _ := os.ReadFile("data/tarot_cards.json")
    var cards []db.TaroCard
    json.Unmarshal(data, &cards)

    for _, card := range cards {
        db.Where("name = ?", card.Name).FirstOrCreate(&card)
    }
    fmt.Println("导入完成")
}
```

---

## 11. 依赖 (go.mod)

```
module backend-go

go 1.22

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/gorilla/websocket v1.5.3
    github.com/google/uuid v1.6.0
    gorm.io/gorm v1.25.12
    gorm.io/driver/sqlite v1.5.7
    github.com/caarlos0/env/v11 v11.2.0
    github.com/cloudwego/eino v0.9.12
    github.com/cloudwego/eino-ext/components/model/openai v0.9.12
)
```

---

## 12. Eino 与 LangGraph 对照

| LangGraph 概念 | Eino 等价 |
|----------------|-----------|
| StateGraph(State) | `compose.NewGraph[*Input, *Output]()` |
| add_node(name, func) | `graph.AddLambdaNode(name, compose.InvokableLambda(func))` |
| set_entry_point(node) | `graph.AddEdge(compose.START, node)` |
| add_edge(from, to) | `graph.AddEdge(from, to)` |
| add_conditional_edges(from, condition_fn) | `graph.AddBranch(from, compose.NewGraphBranch(fn, map[endpoints]bool))` |
| compile() | `graph.Compile(ctx)` |
| ainvoke(state) | `runnable.Invoke(ctx, input)` |
| State (TypedDict) | Go struct (指针传递/修改 field) |

---

## 13. API 兼容性要求

所有端点路径、请求/响应 JSON 结构必须与 Python 版 **完全一致**：

- `GET /api/v1/cards/` → `[{id, name, arcana_type, suit, number, keywords, image_url}, ...]`
- `GET /api/v1/cards/{id}` → `{id, name, arcana_type, ..., description}`
- `GET /api/v1/cards/random/` → `{id, name, ..., is_reversed, meaning}`
- `POST /api/v1/readings/` → `{reading_id, session_id, question, spread_type, created_at, cards: [...]}`
- `POST /api/v1/readings/langgraph` → 同上
- `GET /api/v1/readings/{id}` → 同上
- `WS /api/v1/readings/ws` → 接收 `{question, spread_type, model_name, session_id}`，返回 ReadingResponse 或 `{error, status}`
- `GET /` → `{"message": "欢迎使用星光塔罗AI助手", "version", "docs"}`
- `GET /health` → `{"status": "healthy"}`

---

## 14. 构建和运行

```bash
# 初始化
cd backend-go
go mod tidy

# 导入塔罗牌数据
cp ../backend/data/tarot_cards.json data/
go run scripts/import_cards/main.go

# 启动服务
go run cmd/server/main.go

# 或构建二进制
go build -o taro-agent cmd/server/main.go
./taro-agent
```

默认监听 `:8000`。

---

## 15. 执行步骤

AI 执行时按以下顺序进行：

1. 创建 `backend-go/` 目录结构
2. 初始化 `go.mod` 和所有 `go` 文件框架
3. 实现 `internal/config/config.go`
4. 实现 `internal/db/models.go` 和 `internal/db/database.go`
5. 复制 `data/tarot_cards.json`
6. 实现 `internal/agent/prompt.go` 和 `internal/agent/fallback.go`
7. 实现 `internal/agent/llm.go` (Eino ChatModel 或 OpenAI 兼容客户端)
8. 实现 `internal/agent/graph.go` (Eino compose graph)
9. 实现 `internal/service/card_service.go`
10. 实现 `internal/service/reading_service.go`
11. 实现 `internal/handler/` 各 handler 文件
12. 实现 `internal/middleware/cors.go`
13. 实现 `cmd/server/main.go` 路由与启动
14. 实现 `scripts/import_cards/main.go`
15. 创建 `.env.example`
16. `go mod tidy` 验证编译通过

---

## 16. 踩坑记录 (已知问题与注意事项)

### 16.1 Eino 版本依赖管理
- **问题**：`eino` 核心库与 `eino-ext`（插件库）版本不同步，盲目对齐版本号会导致下载失败。
- **现状**：使用 `github.com/cloudwego/eino v0.9.12` + `github.com/openai/openai-go` 直连 LLM（绕过 eino-ext）。
- **注意**：未来接入 Ollama/DeepSeek 等非 OpenAI 协议时，引入 `eino-ext/components/model/xxx` 需单独查版本号，不可照抄主库版本。

### 16.2 StateGraph vs DataFlow 范式差异
- **LangGraph**：全局可变状态，节点直接修改 State（类似 Redis）。
- **Eino**：无副作用数据流，节点通过 Return 传递数据。
- **兼容写法**：`internal/agent/graph.go` 中传 `*ReadingState` 指针，利用内存共享修改字段。
- **风险**：破坏了函数式无副作用原则；若未来 Eino 增加分布式执行或快照恢复，可能不兼容。
- **建议**：后续迭代可改为纯函数式（每个节点返回新对象），但当前指针方式在本地部署下性能更好。

### 16.3 代码中的隐患（待修复）
- **`internal/agent/graph.go`**：多处使用 `_ = graph.AddLambdaNode(...)` 忽略错误。建议改为 `if err := ...; err != nil { return nil, err }`。
- **`cmd/server/main.go`**：Graph 编译失败时仅打印 `Warning` 未退出，后端启动后所有 `/langgraph` 请求将 500。建议关键依赖失败时 `log.Fatal`。

### 16.4 数据库模型类型对齐
- **Peewee `AutoField`**：默认从 1 开始自增 Int。
- **GORM `uint`**：Go 默认使用 `uint` 作为主键类型，JSON 序列化时输出数字。注意前端是否依赖字符串 ID。
- **SQLite WAL 模式**：Go 版默认开启 WAL+`journal_mode=WAL`，提升并发性能，但会额外生成 `-wal` 和 `-shm` 文件，部署时需一并备份。
