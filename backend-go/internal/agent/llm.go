package agent

import (
	"context"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type ChatClient struct {
	client      openai.Client
	model       string
	temperature float64
	maxTokens   int
}

func NewChatModel(modelName, apiKey, baseURL string) (*ChatClient, error) {
	if modelName == "" {
		modelName = "qwen-plus"
	}
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}
	client := openai.NewClient(opts...)
	return &ChatClient{
		client:      client,
		model:       modelName,
		temperature: 0.7,
		maxTokens:   1000,
	}, nil
}

func (c *ChatClient) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	chatCompletion, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		Temperature: openai.Float(c.temperature),
		MaxTokens:   openai.Int(int64(c.maxTokens)),
	})
	if err != nil {
		return "", err
	}
	if len(chatCompletion.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}
	return chatCompletion.Choices[0].Message.Content, nil
}
