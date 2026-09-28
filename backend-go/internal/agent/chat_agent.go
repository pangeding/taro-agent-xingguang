package agent

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

type ChatAgentConfig struct {
	Name             string
	APIKey           string
	BaseURL          string
	Model            string
	Instruction      string
	MaxIterations    int
	Temperature      float64
	FrequencyPenalty float64
	PresencePenalty  float64
}

// NewChatAgent 构建基于 Eino ADK 的聊天 Agent（内建 ReAct loop，可自主调用工具）。
func NewChatAgent(ctx context.Context, cfg ChatAgentConfig, tools []tool.BaseTool) (*adk.ChatModelAgent, error) {
	if cfg.Name == "" {
		cfg.Name = "tarot-chat"
	}
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 8
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:           cfg.APIKey,
		BaseURL:          cfg.BaseURL,
		Model:            cfg.Model,
		Temperature:      float32Ptr(cfg.Temperature),
		FrequencyPenalty: float32Ptr(cfg.FrequencyPenalty),
		PresencePenalty:  float32Ptr(cfg.PresencePenalty),
	})
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        cfg.Name,
		Description: "星语塔罗师：可以正常对话，也能真实抽牌并解读",
		Instruction: cfg.Instruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: cfg.MaxIterations,
	})
}

func float32Ptr(v float64) *float32 {
	f := float32(v)
	return &f
}
