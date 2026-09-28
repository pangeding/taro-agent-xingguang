package main

import (
	"context"
	"log"

	"backend-go/internal/agent"
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

	driver, dsn := cfg.EffectiveDB()
	log.Printf("database driver: %s", driver)
	d := db.Init(driver, dsn)
	db.AutoMigrate(d)
	db.BackfillConversationChannel(d)

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
	v1.Use(middleware.UserID())
	{
		v1.POST("/user/init", handler.InitUser())

		cards := v1.Group("/cards")
		cards.GET("/", handler.GETAllCards(cardService))
		cards.GET("/:id", handler.GetCard(cardService))
		cards.GET("/random/", handler.GetRandomCard(cardService))

		readings := v1.Group("/readings")
		readings.POST("/", handler.CreateReading(readingService))
		readings.POST("/langgraph", handler.CreateReadingLangGraph(readingService))
		readings.GET("/:id", handler.GetReading(readingService))
		readings.GET("/ws", handler.WebSocketReading(readingService))

		conversations := v1.Group("/conversations")
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

	if err := r.Run(":8000"); err != nil {
		log.Fatal(err)
	}
}
