package main

import (
	"context"
	"log"

	"backend-go/internal/agent"
	"backend-go/internal/bootstrap"
	"backend-go/internal/config"
	"backend-go/internal/db"
	"backend-go/internal/handler"
	"backend-go/internal/middleware"
	"backend-go/internal/service"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if cfg.IsProd() {
		log.Printf("运行模式: prod（生产保护已启用，监听 %s）", cfg.ListenAddr())
	} else {
		// 刻意不打印口令：BOOTSTRAP_ADMIN_PASSWORD 在 dev 也可能被改成自定义值，
		// 写进启动日志等于把口令落到日志文件里。
		log.Printf("运行模式: %s（本地开发；users 表为空时会创建默认引导账号）", cfg.AppEnv)
	}

	driver, dsn := cfg.EffectiveDB()
	log.Printf("database driver: %s", driver)
	d := db.Init(driver, dsn)
	db.AutoMigrate(d)
	db.BackfillConversationChannel(d)

	// 账号引导与生产环境保护（L1 引导口令校验 / L2 存量弱口令扫描）。
	// 失败即拒绝启动，绝不降级 —— 见技术文档 §3.3。
	if _, err := bootstrap.Users(d, cfg); err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	userService := service.NewUserService(d, cfg)
	userService.CleanupExpiredSessions()

	var llm *agent.ChatClient
	var graph compose.Runnable[*agent.ReadingState, *agent.ReadingState]
	var err error

	llm, err = agent.NewChatModel(cfg.DashScopeModel, cfg.DashScopeAPIKey, cfg.DashScopeBaseURL)
	if err != nil {
		log.Printf("Warning: failed to create LLM client: %v", err)
	} else {
		graph, err = agent.CreateReadingGraph(llm)
		if err != nil {
			log.Printf("Warning: failed to create reading graph: %v", err)
		}
	}

	cardService := service.NewCardService(d)
	readingService := service.NewReadingService(d, graph, llm)
	conversationService := service.NewConversationService(d)
	conversationService.SetLLM(llm)

	if llm != nil {
		drawTool, err := service.NewTarotDrawTool(readingService)
		if err != nil {
			log.Printf("Warning: failed to create draw_tarot tool: %v", err)
		} else {
			chatAgent, err := agent.NewChatAgent(context.Background(), agent.ChatAgentConfig{
				Name:             "tarot-chat",
				APIKey:           cfg.DashScopeAPIKey,
				BaseURL:          cfg.DashScopeBaseURL,
				Model:            cfg.DashScopeModel,
				Instruction:      agent.CHAT_SYSTEM_PROMPT,
				MaxIterations:    8,
				Temperature:      0.7,
				FrequencyPenalty: 0.4,
				PresencePenalty:  0.2,
			}, []tool.BaseTool{drawTool})
			if err != nil {
				log.Printf("Warning: failed to create chat agent: %v", err)
			} else {
				conversationService.SetAgent(chatAgent)
			}
		}
	}

	r := gin.Default()
	r.Use(middleware.CORS(cfg.BackendCORSOrigins))

	r.GET("/", handler.Root(cfg))
	r.GET("/health", handler.HealthCheck)

	v1 := r.Group(cfg.APIV1Str)
	{
		// —— 无需登录 ——
		// 登录是唯一的不认证入口；密码校验靠 bcrypt + 失败锁定。
		v1.POST("/auth/login", handler.Login(userService, cfg))

		// —— 以下全部需要登录 ——
		authed := v1.Group("")
		authed.Use(middleware.Auth(cfg, userService))
		{
			authed.POST("/auth/logout", handler.Logout(userService, cfg))
			authed.GET("/auth/me", handler.Me())
			authed.POST("/auth/password", handler.ChangePassword(userService, cfg))

			cards := authed.Group("/cards")
			cards.GET("/", handler.GETAllCards(cardService))
			cards.GET("/:id", handler.GetCard(cardService))
			cards.GET("/random/", handler.GetRandomCard(cardService))

			readings := authed.Group("/readings")
			readings.POST("/", handler.CreateReading(readingService))
			readings.POST("/langgraph", handler.CreateReadingLangGraph(readingService))
			readings.GET("/:id", handler.GetReading(readingService))
			readings.GET("/ws", handler.WebSocketReading(readingService, cfg))

			conversations := authed.Group("/conversations")
			{
				conversations.POST("", handler.CreateConversation(conversationService))
				conversations.GET("", handler.ListConversations(conversationService))
				conversations.GET("/:id", handler.GetConversation(conversationService))
				conversations.DELETE("/:id", handler.DeleteConversation(conversationService))
				conversations.PATCH("/:id/title", handler.UpdateConversationTitle(conversationService))
				conversations.GET("/:id/messages", handler.GetMessages(conversationService))
				conversations.POST("/:id/messages", handler.StreamChatMessage(conversationService))
				conversations.POST("/:id/tarot", handler.StreamTarotReading(conversationService, readingService))
			}
		}
	}

	if err := r.Run(cfg.ListenAddr()); err != nil {
		log.Fatal(err)
	}
}
