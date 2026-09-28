package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"backend-go/internal/agent"
	"backend-go/internal/db"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

type ConversationService struct {
	DB    *gorm.DB
	LLM   *agent.ChatClient
	Agent *adk.ChatModelAgent
	MsgID uint
}

// 会话频道：区分会话属于哪个功能，避免占卜记录混进聊天页。
const (
	ChannelChat    = "chat"
	ChannelReading = "reading"
)

func normalizeChannel(channel string) string {
	if channel == ChannelReading {
		return ChannelReading
	}
	return ChannelChat
}

// CardBrief 是挂在会话消息上的牌面摘要，供前端渲染牌条。
type CardBrief struct {
	Name         string `json:"name"`
	IsReversed   bool   `json:"is_reversed"`
	Position     int    `json:"position"`
	PositionName string `json:"position_name,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
}

// MessageDetail 是消息 + 该消息关联占卜记录（reading_id）的牌面。
// 没有 reading_id 的消息 Cards 为 nil，序列化后不出现 cards 字段。
type MessageDetail struct {
	db.Message
	Cards []CardBrief `json:"cards,omitempty"`
}

// DecorateMessages 批量补全消息上的牌面信息。
// 一次性取回所有相关 readings / reading_cards / tarot_cards，避免 N+1 查询。
func (s *ConversationService) DecorateMessages(msgs []db.Message) []MessageDetail {
	out := make([]MessageDetail, len(msgs))
	seen := make(map[uint]bool, len(msgs))
	readingIDs := make([]uint, 0, len(msgs))

	for i, m := range msgs {
		out[i] = MessageDetail{Message: m}
		if m.ReadingID != nil && !seen[*m.ReadingID] {
			seen[*m.ReadingID] = true
			readingIDs = append(readingIDs, *m.ReadingID)
		}
	}
	if len(readingIDs) == 0 {
		return out
	}

	var readings []db.Reading
	s.DB.Where("id IN ?", readingIDs).Find(&readings)
	spreadOf := make(map[uint]string, len(readings))
	for _, r := range readings {
		spreadOf[r.ID] = r.SpreadType
	}

	var rcs []db.ReadingCard
	s.DB.Where("reading_id IN ?", readingIDs).Order("position ASC").Find(&rcs)

	cardIDs := make([]uint, 0, len(rcs))
	for _, rc := range rcs {
		cardIDs = append(cardIDs, rc.CardID)
	}

	cardsByID := make(map[uint]db.TarotCard, len(cardIDs))
	if len(cardIDs) > 0 {
		var cards []db.TarotCard
		s.DB.Where("id IN ?", cardIDs).Find(&cards)
		for _, c := range cards {
			cardsByID[c.ID] = c
		}
	}

	byReading := make(map[uint][]CardBrief, len(readingIDs))
	for _, rc := range rcs {
		card := cardsByID[rc.CardID]
		brief := CardBrief{
			Name:         card.Name,
			IsReversed:   rc.IsReversed,
			Position:     rc.Position,
			PositionName: positionName(rc.Position, spreadOf[rc.ReadingID]),
		}
		if card.ImageURL != nil {
			brief.ImageURL = *card.ImageURL
		}
		byReading[rc.ReadingID] = append(byReading[rc.ReadingID], brief)
	}

	for i := range out {
		if out[i].ReadingID != nil {
			out[i].Cards = byReading[*out[i].ReadingID]
		}
	}
	return out
}

func NewConversationService(d *gorm.DB) *ConversationService {
	return &ConversationService{DB: d}
}

func (s *ConversationService) SetLLM(llm *agent.ChatClient) {
	s.LLM = llm
}

func (s *ConversationService) SetAgent(a *adk.ChatModelAgent) {
	s.Agent = a
}

type ChatChunk struct {
	Delta string `json:"delta"`
}

type ChatDone struct {
	Delta    string `json:"delta"`
	FullText string `json:"full_text"`
	Done     bool   `json:"done"`
	MsgID    uint   `json:"msg_id"`
}

func (s *ConversationService) StreamChat(ctx context.Context, conversationID uint, userID, content string, writer func(event string, data string)) error {
	userMsg, err := s.AddMessage(conversationID, userID, "user", content, "text")
	if err != nil {
		return err
	}
	_ = userMsg

	if s.Agent == nil {
		return errors.New("chat agent not configured")
	}

	var messages []db.Message
	s.DB.Where("conversation_id = ?", conversationID).Order("created_at ASC").Find(&messages)

	history := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			history = append(history, schema.UserMessage(msg.Content))
		case "assistant":
			history = append(history, schema.AssistantMessage(msg.Content, nil))
		}
	}

	runCtx := WithConversationID(ctx, conversationID)
	runner := adk.NewRunner(runCtx, adk.RunnerConfig{Agent: s.Agent, EnableStreaming: true})
	iter := runner.Run(runCtx, history)

	var fullContent strings.Builder
	// draw_tarot 产生的占卜记录 id，写到 assistant 消息上，
	// 前端刷新后才能凭 reading_id 还原牌面。
	var readingID uint
	for {
		ev, ok := iter.Next()
		if !ok {
			break
		}
		if ev.Err != nil {
			return ev.Err
		}
		if ev.Output == nil || ev.Output.MessageOutput == nil {
			continue
		}
		mo := ev.Output.MessageOutput
		switch mo.Role {
		case schema.Assistant:
			if mo.IsStreaming {
				for {
					chunk, err := mo.MessageStream.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						return err
					}
					if chunk == nil || chunk.Content == "" {
						continue
					}
					fullContent.WriteString(chunk.Content)
					writer("message", fmt.Sprintf(`{"delta": %s}`, toRawJSON(chunk.Content)))
				}
			} else if mo.Message != nil && mo.Message.Content != "" {
				fullContent.WriteString(mo.Message.Content)
				writer("message", fmt.Sprintf(`{"delta": %s}`, toRawJSON(mo.Message.Content)))
			}
		case schema.Tool:
			toolMsg, err := mo.GetMessage()
			if err != nil {
				return err
			}
			if toolMsg != nil && mo.ToolName == "draw_tarot" {
				if id := emitCardDrawn(writer, toolMsg.Content); id != 0 {
					readingID = id
				}
			}
		}
	}

	text := fullContent.String()
	assistantMsg, err := s.AddMessage(conversationID, userID, "assistant", text, "text")
	if err != nil {
		return err
	}
	if readingID != 0 {
		s.DB.Model(&db.Message{}).Where("id = ?", assistantMsg.ID).Update("reading_id", readingID)
	}
	writer("done", fmt.Sprintf(`{"done": true, "full_text": %s, "reading_id": %d, "msg_id": %d}`, toRawJSON(text), readingID, assistantMsg.ID))

	if count, err := s.GetMessageCount(conversationID); err == nil && int(count) == 2 {
		s.GenerateTitleAsync(conversationID)
	}

	return nil
}

// emitCardDrawn 把 draw_tarot 工具返回的真实牌面转成 SSE card_drawn 事件，
// 返回该次抽牌对应的 reading id（解析失败返回 0）。
func emitCardDrawn(writer func(event string, data string), toolResultJSON string) uint {
	var out DrawTarotOutput
	if err := json.Unmarshal([]byte(toolResultJSON), &out); err != nil {
		return 0
	}
	for _, c := range out.Cards {
		writer("card_drawn", toRawJSON(map[string]any{
			"name":          c.Name,
			"is_reversed":   c.IsReversed,
			"position":      c.Position,
			"position_name": c.PositionName,
			"reading_id":    out.ReadingID,
			"image_url":     c.ImageURL,
		}))
	}
	return out.ReadingID
}

func toRawJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (s *ConversationService) CreateConversation(userID, channel string) (*db.Conversation, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	conv := db.Conversation{UserID: userID, Title: "新对话", Channel: normalizeChannel(channel)}
	if err := s.DB.Create(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// ListConversations 按 channel 过滤。
// channel 为空时只返回 chat 频道（保持「不传参数 = 聊天」的直觉），
// channel 为 "all" 时不过滤。
func (s *ConversationService) ListConversations(userID, channel string) ([]db.Conversation, error) {
	q := s.DB.Where("user_id = ?", userID)
	if channel != "all" {
		q = q.Where("channel = ?", normalizeChannel(channel))
	}

	var conversations []db.Conversation
	err := q.Order("updated_at DESC").Find(&conversations).Error
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

	// 先推牌面：用户立刻看到抽到什么牌，不必等解读
	for i, dc := range drawnCards {
		cardData := fmt.Sprintf(`{"name": %s, "is_reversed": %t, "position": %d}`, toRawJSON(dc.Name), dc.IsReversed, i)
		writer("card_drawn", cardData)
	}

	sessionID := fmt.Sprintf("conv_%d", conversationID)
	p, err := readingSvc.PrepareReading(question, spreadType, &sessionID, drawnCards)
	if err != nil {
		return err
	}

	// 主输出（message 事件）按牌阵分派，两路互斥，界面上每段文字只出现一次：
	//   single —— 这张牌的解读本身，写入 reading_cards.interpretation
	//   three  —— 三张牌的综合解读，写入 readings.synthesis
	var mainText string

	if spreadType == "three" {
		interps := s.interpretAll(ctx, readingSvc, p, writer)
		mainText, _ = readingSvc.StreamSynthesis(ctx, p, interps, func(delta string) {
			writer("message", fmt.Sprintf(`{"delta": %s}`, toRawJSON(delta)))
		})
		readingSvc.SaveSynthesis(p, mainText)
	} else {
		mainText, _ = readingSvc.StreamCardInterpretation(ctx, p, &p.Cards[0], func(delta string) {
			writer("message", fmt.Sprintf(`{"delta": %s}`, toRawJSON(delta)))
		})
		readingSvc.SaveCardInterpretation(&p.Cards[0], mainText)
	}

	readingMsg, err := s.AddMessage(conversationID, userID, "assistant", mainText, "reading")
	if err != nil {
		return err
	}
	s.DB.Model(&db.Message{}).Where("id = ?", readingMsg.ID).Update("reading_id", p.Reading.ID)

	writer("done", fmt.Sprintf(`{"done": true, "full_text": %s, "reading_id": %d, "msg_id": %d}`, toRawJSON(mainText), p.Reading.ID, readingMsg.ID))

	if msgCount, err := s.GetMessageCount(conversationID); err == nil && int(msgCount) == 4 {
		go s.GenerateTitleAsync(conversationID)
	}

	return nil
}

// interpretAll 并发解读三张牌，每张牌解读完成即推一条 card_interpreted 事件，
// 界面因此能在综合解读之前就逐张显示内容。
// 返回按 position 升序排列的解读，可直接交给 BuildSynthesisPrompt。
func (s *ConversationService) interpretAll(ctx context.Context, readingSvc *ReadingService, p *PreparedReading, writer func(event string, data string)) []agent.CardInterpretation {
	interps := make([]agent.CardInterpretation, len(p.Cards))
	var wg sync.WaitGroup
	// writer 最终落到 gin.ResponseWriter，多 goroutine 并发写会交错出半截事件，必须串行化
	var mu sync.Mutex

	for i := range p.Cards {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			rc := &p.Cards[i]
			text := readingSvc.InterpretCard(ctx, p, rc)
			readingSvc.SaveCardInterpretation(rc, text)

			cardName := ""
			if card := p.CardData[rc.CardID]; card != nil {
				cardName = card.Name
			}

			interps[i] = agent.CardInterpretation{
				CardID:         rc.CardID,
				CardName:       cardName,
				IsReversed:     rc.IsReversed,
				Position:       rc.Position,
				Interpretation: text,
				Status:         "success",
			}

			mu.Lock()
			defer mu.Unlock()
			writer("card_interpreted", toRawJSON(map[string]any{
				"position":       rc.Position,
				"name":           cardName,
				"is_reversed":    rc.IsReversed,
				"interpretation": text,
			}))
		}(i)
	}

	wg.Wait()
	return interps
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
