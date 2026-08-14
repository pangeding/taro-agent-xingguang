package agent

import (
	"context"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
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
		maxTokens:   2000,
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

func (c *ChatClient) ChatStream(ctx context.Context, systemPrompt string, messages []openai.ChatCompletionMessageParamUnion) (*StreamHandler, error) {
	allMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	if systemPrompt != "" {
		allMessages = append(allMessages, openai.SystemMessage(systemPrompt))
	}
	allMessages = append(allMessages, messages...)

	stream := c.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:       c.model,
		Messages:    allMessages,
		Temperature: openai.Float(c.temperature),
		MaxTokens:   openai.Int(int64(c.maxTokens)),
	})

	return &StreamHandler{stream: stream}, nil
}

func (c *ChatClient) ChatStreamSimple(ctx context.Context, systemPrompt, userPrompt string) (*StreamHandler, error) {
	stream := c.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		Temperature: openai.Float(c.temperature),
		MaxTokens:   openai.Int(int64(c.maxTokens)),
	})

	return &StreamHandler{stream: stream}, nil
}

type StreamHandler struct {
	stream       *ssestream.Stream[openai.ChatCompletionChunk]
	fullContent  string
	done         bool
}

func (s *StreamHandler) Next() (string, error) {
	if s.done {
		return "", fmt.Errorf("stream completed")
	}
	if !s.stream.Next() {
		s.done = true
		if err := s.stream.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("stream completed")
	}
	evt := s.stream.Current()
	for _, choice := range evt.Choices {
		if choice.Delta.Content != "" {
			s.fullContent += choice.Delta.Content
			return choice.Delta.Content, nil
		}
	}
	return "", nil
}

func (s *StreamHandler) IsDone() bool {
	return s.done
}

func (s *StreamHandler) FullContent() string {
	return s.fullContent
}

func (s *StreamHandler) Close() {
	s.stream.Close()
}
