package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"backend-go/internal/agent"
	"backend-go/internal/db"

	openai "github.com/openai/openai-go"
	"gorm.io/gorm"
)

type ConversationService struct {
	DB    *gorm.DB
	LLM   *agent.ChatClient
	MsgID uint
}

func NewConversationService(d *gorm.DB) *ConversationService {
	return &ConversationService{DB: d}
}

func (s *ConversationService) SetLLM(llm *agent.ChatClient) {
	s.LLM = llm
}

type ChatChunk struct {
	Delta string `json:"delta"`
}

type ChatDone struct {
	Delta   string `json:"delta"`
	FullText string `json:"full_text"`
	Done    bool   `json:"done"`
	MsgID   uint   `json:"msg_id"`
}

func (s *ConversationService) StreamChat(ctx context.Context, conversationID uint, userID, content string, writer func(event string, data string)) error {
	userMsg, err := s.AddMessage(conversationID, userID, "user", content, "text")
	if err != nil {
		return err
	}
	_ = userMsg

	if s.LLM == nil {
		return errors.New("LLM not configured")
	}

	var messages []db.Message
	s.DB.Where("conversation_id = ?", conversationID).Order("created_at ASC").Find(&messages)

	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "user" {
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		} else if msg.Role == "assistant" {
			openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
		}
	}

	stream, err := s.LLM.ChatStream(ctx, agent.CHAT_SYSTEM_PROMPT, openaiMessages)
	if err != nil {
		return err
	}
	defer stream.Close()

	var fullContent string
	for {
		delta, err := stream.Next()
		if err != nil {
			break
		}
		fullContent += delta
		writer("message", fmt.Sprintf(`{"delta": %s}`, toRawJSON(delta)))
	}

	if writer != nil {
		assistantMsg, _ := s.AddMessage(conversationID, userID, "assistant", fullContent, "text")
		writer("done", fmt.Sprintf(`{"done": true, "full_text": %s, "msg_id": %d}`, toRawJSON(fullContent), assistantMsg.ID))
	}

	return nil
}

func toRawJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (s *ConversationService) CreateConversation(userID string) (*db.Conversation, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	conv := db.Conversation{UserID: userID, Title: "新对话"}
	if err := s.DB.Create(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func (s *ConversationService) ListConversations(userID string) ([]db.Conversation, error) {
	var conversations []db.Conversation
	err := s.DB.Where("user_id = ?", userID).Order("updated_at DESC").Find(&conversations).Error
	return conversations, err
}

func (s *ConversationService) GetConversation(id uint, userID string) (*db.Conversation, error) {
	var conv db.Conversation
	err := s.DB.Where("id = ? AND user_id = ?", id, userID).Preload("Messages").First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (s *ConversationService) DeleteConversation(id uint, userID string) error {
	result := s.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&db.Conversation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("conversation not found")
	}
	return nil
}

func (s *ConversationService) UpdateTitle(id uint, userID, title string) error {
	result := s.DB.Model(&db.Conversation{}).Where("id = ? AND user_id = ?", id, userID).Update("title", title)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("conversation not found")
	}
	return nil
}

func (s *ConversationService) AddMessage(conversationID uint, userID, role, content, messageType string) (*db.Message, error) {
	var conv db.Conversation
	if err := s.DB.Where("id = ? AND user_id = ?", conversationID, userID).First(&conv).Error; err != nil {
		return nil, err
	}

	msg := db.Message{
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		Type:           messageType,
	}
	if err := s.DB.Create(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *ConversationService) GetMessages(conversationID uint, userID string, limit, offset int) ([]db.Message, error) {
	var conv db.Conversation
	if err := s.DB.Where("id = ? AND user_id = ?", conversationID, userID).First(&conv).Error; err != nil {
		return nil, err
	}

	var messages []db.Message
	err := s.DB.Where("conversation_id = ?", conversationID).Order("created_at ASC").Limit(limit).Offset(offset).Find(&messages).Error
	return messages, err
}

func (s *ConversationService) GetMessageCount(conversationID uint) (int64, error) {
	var count int64
	err := s.DB.Model(&db.Message{}).Where("conversation_id = ?", conversationID).Count(&count).Error
	return count, err
}

type TarotStreamEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

func (s *ConversationService) StreamTarotReading(ctx context.Context, conversationID uint, userID, question, spreadType string, writer func(event string, data string), readingSvc *ReadingService) error {
	if s.LLM == nil || readingSvc == nil {
		return errors.New("LLM or ReadingService not configured")
	}

	if spreadType == "" {
		spreadType = "single"
	}

	userMsg, err := s.AddMessage(conversationID, userID, "user", "🎴 塔罗占卜："+question, "reading")
	if err != nil {
		return err
	}
	_ = userMsg

	count := 1
	if spreadType == "three" {
		count = 3
	}

	drawnCards, err := readingSvc.DrawCards(count)
	if err != nil {
		return err
	}

	for i, dc := range drawnCards {
		cardData := fmt.Sprintf(`{"name": %s, "is_reversed": %t, "position": %d}`, toRawJSON(dc.Name), dc.IsReversed, i)
		writer("card_drawn", cardData)
	}

	var readingResult *ReadingResult
	sessionID := fmt.Sprintf("conv_%d", conversationID)

	if spreadType == "three" {
		sid := &sessionID
		readingResult, err = readingSvc.CreateReadingLangGraph(question, spreadType, sid, nil)
	} else {
		sid := &sessionID
		readingResult, err = readingSvc.CreateReading(question, spreadType, sid)
	}
	if err != nil {
		return err
	}

	contextMsgs := "对话历史："
	if readingResult != nil && len(readingResult.Cards) > 0 {
		for _, c := range readingResult.Cards {
			contextMsgs += fmt.Sprintf("\n- %s(%s)：%s", c.Name, positionName(c.Position, spreadType), c.Interpretation)
		}
	}

	openaiMessages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(question),
		openai.AssistantMessage("已为您抽取塔罗牌。" + contextMsgs),
	}

	stream, err := s.LLM.ChatStream(ctx, agent.TAROT_READING_SYSTEM_PROMPT, openaiMessages)
	if err != nil {
		return err
	}
	defer stream.Close()

	var fullContent string
	for {
		delta, err := stream.Next()
		if err != nil {
			break
		}
		fullContent += delta
		writer("message", fmt.Sprintf(`{"delta": %s}`, toRawJSON(delta)))
	}

	if readingResult != nil {
		readingMsg, _ := s.AddMessage(conversationID, userID, "assistant", fullContent, "reading")

		s.DB.Model(&db.Message{}).Where("id = ?", readingMsg.ID).Update("reading_id", readingResult.ReadingID)

		writer("done", fmt.Sprintf(`{"done": true, "full_text": %s, "reading_id": %d, "msg_id": %d}`, toRawJSON(fullContent), readingResult.ReadingID, readingMsg.ID))
	}

	msgCount, _ := s.GetMessageCount(conversationID)
	if int(msgCount) == 4 {
		go s.GenerateTitleAsync(conversationID)
	}

	return nil
}

func positionName(pos int, spreadType string) string {
	if spreadType == "three" {
		positions := []string{"过去", "现在", "未来"}
		if pos < len(positions) {
			return positions[pos]
		}
	}
	return ""
}

func (s *ConversationService) GenerateTitleAsync(conversationID uint) {
	go func() {
		var conv db.Conversation
		if err := s.DB.First(&conv, conversationID).Error; err != nil {
			return
		}

		if conv.Title != "新对话" {
			return
		}

		var messages []db.Message
		s.DB.Where("conversation_id = ?", conversationID).Order("created_at ASC").Limit(2).Find(&messages)

		if len(messages) < 2 {
			return
		}

		summaryPrompt := fmt.Sprintf("请用3-8个中文汉字总结以下对话的主题，只返回标题文字，不要加引号或其他内容。\n用户问题：%s", messages[0].Content)

		ctx := context.Background()
		title, err := s.LLM.Chat(ctx, "你擅长总结对话主题。请根据用户的问题生成一个简短的对话标题，3-8个汉字。只返回标题内容。", summaryPrompt)
		if err != nil {
			return
		}

		if title != "" {
			s.DB.Model(&db.Conversation{}).Where("id = ?", conversationID).Update("title", title)
		}
	}()
}
