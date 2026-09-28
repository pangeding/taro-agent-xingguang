package service

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"strings"
	"time"

	"backend-go/internal/agent"
	"backend-go/internal/db"

	"github.com/cloudwego/eino/compose"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DrawnCard struct {
	ID         uint
	Name       string
	IsReversed bool
}

type ReadingCardResponse struct {
	CardID         uint    `json:"card_id"`
	Name           string  `json:"name"`
	Position       int     `json:"position"`
	IsReversed     bool    `json:"is_reversed"`
	Interpretation string  `json:"interpretation"`
	ImageURL       *string `json:"image_url"`

	// 牌面本身的信息（此前只在 GET /cards/:id 的 CardDetail 中返回，
	// 导致占卜结果页只能显示与具体牌无关的模板文案）。仅新增字段，不改既有字段。
	ArcanaType      string  `json:"arcana_type"`
	Suit            *string `json:"suit"`
	Number          *int    `json:"number"`
	Keywords        string  `json:"keywords"`
	Element         *string `json:"element"`
	ZodiacSign      *string `json:"zodiac_sign"`
	MeaningUpright  string  `json:"meaning_upright"`
	MeaningReversed string  `json:"meaning_reversed"`
	Description     string  `json:"description"`
}

type ReadingResult struct {
	ReadingID  uint                  `json:"reading_id"`
	SessionID  string                `json:"session_id"`
	Question   string                `json:"question"`
	SpreadType string                `json:"spread_type"`
	Synthesis  string                `json:"synthesis"`
	CreatedAt  *time.Time            `json:"created_at"`
	Cards      []ReadingCardResponse `json:"cards"`
}

type ReadingService struct {
	DB    *gorm.DB
	Graph compose.Runnable[*agent.ReadingState, *agent.ReadingState]
	LLM   *agent.ChatClient
}

func NewReadingService(db *gorm.DB, graph compose.Runnable[*agent.ReadingState, *agent.ReadingState], llm *agent.ChatClient) *ReadingService {
	return &ReadingService{DB: db, Graph: graph, LLM: llm}
}

func cardCount(spreadType string) int {
	if spreadType == "three" {
		return 3
	}
	return 1
}

func (s *ReadingService) DrawCards(count int) ([]DrawnCard, error) {
	var cardIDs []uint
	s.DB.Model(&db.TarotCard{}).Pluck("id", &cardIDs)
	if len(cardIDs) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if count > len(cardIDs) {
		count = len(cardIDs)
	}

	selectedMap := make(map[uint]bool)
	var selected []uint
	for len(selected) < count {
		idx := r.Intn(len(cardIDs))
		id := cardIDs[idx]
		if !selectedMap[id] {
			selectedMap[id] = true
			selected = append(selected, id)
		}
	}

	var cards []db.TarotCard
	s.DB.Where("id IN ?", selected).Find(&cards)

	result := make([]DrawnCard, len(cards))
	for i, c := range cards {
		isReversed := r.Intn(2) == 1
		result[i] = DrawnCard{ID: c.ID, Name: c.Name, IsReversed: isReversed}
	}
	return result, nil
}

// CreateReading 自行抽牌后解读。
func (s *ReadingService) CreateReading(question, spreadType string, sessionID *string) (*ReadingResult, error) {
	drawnCards, err := s.DrawCards(cardCount(spreadType))
	if err != nil {
		return nil, err
	}
	return s.CreateReadingWithCards(question, spreadType, sessionID, drawnCards)
}

// synthesisMarker 是历史版本把综合解读拼进最后一张牌解读末尾时使用的分隔符。
// 仅用于读取旧数据时拆分，新数据写入 readings.synthesis 列。
const synthesisMarker = "\n\n---\n**综合解读**: "

// PreparedReading 是已落库、等待写入解读的占卜记录。
// 抽牌与解读拆开，是为了让流式占卜能在抽牌后立刻推送牌面、再逐张推进度。
type PreparedReading struct {
	Reading  *db.Reading
	Cards    []db.ReadingCard       // 按 position 升序
	CardData map[uint]*db.TarotCard // card_id → 牌库数据
}

// PrepareReading 建 readings / reading_cards 行，不调用 LLM。
func (s *ReadingService) PrepareReading(question, spreadType string, sessionID *string, drawnCards []DrawnCard) (*PreparedReading, error) {
	sid := uuid.New().String()
	if sessionID != nil && *sessionID != "" {
		sid = *sessionID
	}

	reading := db.Reading{SessionID: sid, Question: question, SpreadType: spreadType}
	if err := s.DB.Create(&reading).Error; err != nil {
		return nil, err
	}

	for i, dc := range drawnCards {
		rc := db.ReadingCard{
			ReadingID:  reading.ID,
			CardID:     dc.ID,
			Position:   i,
			IsReversed: dc.IsReversed,
		}
		if err := s.DB.Create(&rc).Error; err != nil {
			return nil, err
		}
	}

	var rcs []db.ReadingCard
	s.DB.Where("reading_id = ?", reading.ID).Order("position ASC").Find(&rcs)

	cardIDs := make([]uint, 0, len(drawnCards))
	for _, dc := range drawnCards {
		cardIDs = append(cardIDs, dc.ID)
	}

	cardData := make(map[uint]*db.TarotCard, len(cardIDs))
	if len(cardIDs) > 0 {
		var cards []db.TarotCard
		s.DB.Where("id IN ?", cardIDs).Find(&cards)
		for i := range cards {
			cardData[cards[i].ID] = &cards[i]
		}
	}

	return &PreparedReading{Reading: &reading, Cards: rcs, CardData: cardData}, nil
}

// InterpretCard 非流式解读单张牌；LLM 不可用或失败时降级为牌义库文本。
func (s *ReadingService) InterpretCard(ctx context.Context, p *PreparedReading, rc *db.ReadingCard) string {
	card := p.CardData[rc.CardID]
	if s.LLM == nil || card == nil {
		return agent.GetBasicInterpretation(card, rc.IsReversed)
	}

	prompt := agent.BuildCardPrompt(card, rc.IsReversed, p.Reading.Question, rc.Position, p.Reading.SpreadType)
	text, err := s.LLM.Chat(ctx, agent.SYSTEM_PROMPT, prompt)
	if err != nil || strings.TrimSpace(text) == "" {
		return agent.GetBasicInterpretation(card, rc.IsReversed)
	}
	return strings.TrimSpace(text)
}

// StreamCardInterpretation 流式解读单张牌，onDelta 逐字回调，返回最终文本。
//
// 降级路径也会通过 onDelta 一次性输出，保证界面上一定有内容可见。
// 返回的 error 仅供上层记日志，调用方无需中断流程。
func (s *ReadingService) StreamCardInterpretation(ctx context.Context, p *PreparedReading, rc *db.ReadingCard, onDelta func(string)) (string, error) {
	card := p.CardData[rc.CardID]
	if s.LLM == nil || card == nil {
		text := agent.GetBasicInterpretation(card, rc.IsReversed)
		onDelta(text)
		return text, nil
	}

	prompt := agent.BuildCardPrompt(card, rc.IsReversed, p.Reading.Question, rc.Position, p.Reading.SpreadType)
	stream, err := s.LLM.ChatStreamSimple(ctx, agent.SYSTEM_PROMPT, prompt)
	if err != nil {
		text := agent.GetBasicInterpretation(card, rc.IsReversed)
		onDelta(text)
		return text, err
	}
	defer stream.Close()

	text, streamErr := drainStream(stream, onDelta)
	if text == "" {
		text = agent.GetBasicInterpretation(card, rc.IsReversed)
		onDelta(text)
	}
	return text, streamErr
}

// StreamSynthesis 流式生成综合解读，onDelta 逐字回调，返回最终文本。
// 降级行为同 StreamCardInterpretation。
func (s *ReadingService) StreamSynthesis(ctx context.Context, p *PreparedReading, interps []agent.CardInterpretation, onDelta func(string)) (string, error) {
	const fallback = "（综合解读暂时不可用，请先参考上方各张牌的解读）"

	if s.LLM == nil {
		onDelta(fallback)
		return fallback, nil
	}

	prompt := agent.BuildSynthesisPrompt(interps, p.Reading.Question)
	stream, err := s.LLM.ChatStreamSimple(ctx, agent.SYSTEM_PROMPT, prompt)
	if err != nil {
		onDelta(fallback)
		return fallback, err
	}
	defer stream.Close()

	text, streamErr := drainStream(stream, onDelta)
	if text == "" {
		text = fallback
		onDelta(text)
	}
	return text, streamErr
}

// drainStream 消费一个流，逐字回调并返回去掉首尾空白的全文。
func drainStream(stream *agent.StreamHandler, onDelta func(string)) (string, error) {
	var full strings.Builder
	for {
		delta, err := stream.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return strings.TrimSpace(full.String()), err
		}
		if delta == "" {
			continue
		}
		full.WriteString(delta)
		onDelta(delta)
	}
	return strings.TrimSpace(full.String()), nil
}

// SaveCardInterpretation 把解读写回 reading_cards。
func (s *ReadingService) SaveCardInterpretation(rc *db.ReadingCard, text string) {
	rc.Interpretation = text
	s.DB.Save(rc)
}

// SaveSynthesis 把综合解读写入 readings.synthesis。
func (s *ReadingService) SaveSynthesis(p *PreparedReading, text string) {
	p.Reading.Synthesis = text
	s.DB.Model(&db.Reading{}).Where("id = ?", p.Reading.ID).Update("synthesis", text)
}

// CreateReadingWithCards 用调用方给定的牌生成解读。
//
// 流式占卜（StreamTarotReading）需要先把抽到的牌推给前端、再解读，
// 若此处再次抽牌，就会出现「推给前端的牌」与「实际解读的牌」不是同一副的错位。
func (s *ReadingService) CreateReadingWithCards(question, spreadType string, sessionID *string, drawnCards []DrawnCard) (*ReadingResult, error) {
	p, err := s.PrepareReading(question, spreadType, sessionID, drawnCards)
	if err != nil {
		return nil, err
	}

	for i := range p.Cards {
		text := s.InterpretCard(context.Background(), p, &p.Cards[i])
		s.SaveCardInterpretation(&p.Cards[i], text)
	}

	return s.GetReadingByID(p.Reading.ID)
}

// CreateReadingLangGraph 自行抽牌后走 Eino 图解读（三张牌阵会额外生成综合解读）。
func (s *ReadingService) CreateReadingLangGraph(question, spreadType string, sessionID, modelName *string) (*ReadingResult, error) {
	drawnCards, err := s.DrawCards(cardCount(spreadType))
	if err != nil {
		return nil, err
	}
	return s.CreateReadingLangGraphWithCards(question, spreadType, sessionID, modelName, drawnCards)
}

// CreateReadingLangGraphWithCards 用调用方给定的牌走 Eino 图解读。
// 理由同 CreateReadingWithCards。
func (s *ReadingService) CreateReadingLangGraphWithCards(question, spreadType string, sessionID, modelName *string, drawnCards []DrawnCard) (*ReadingResult, error) {
	sid := uuid.New().String()
	if sessionID != nil && *sessionID != "" {
		sid = *sessionID
	}

	reading := db.Reading{SessionID: sid, Question: question, SpreadType: spreadType}
	s.DB.Create(&reading)

	readingCards := make([]db.ReadingCard, len(drawnCards))
	for i, dc := range drawnCards {
		rc := db.ReadingCard{
			ReadingID:  reading.ID,
			CardID:     dc.ID,
			Position:   i,
			IsReversed: dc.IsReversed,
		}
		s.DB.Create(&rc)
		readingCards[i] = rc
	}

	cardIDs := make([]uint, len(drawnCards))
	for i, dc := range drawnCards {
		cardIDs[i] = dc.ID
	}
	var cards []db.TarotCard
	s.DB.Where("id IN ?", cardIDs).Find(&cards)

	cardMap := make(map[uint]*db.TarotCard)
	for i := range cards {
		cardMap[cards[i].ID] = &cards[i]
	}

	cardsInfo := make([]agent.CardInfo, len(readingCards))
	for i, rc := range readingCards {
		cardsInfo[i] = agent.CardInfo{
			ID:         rc.CardID,
			Name:       cardMap[rc.CardID].Name,
			IsReversed: rc.IsReversed,
			Position:   rc.Position,
			Card:       cardMap[rc.CardID],
		}
	}

	state := &agent.ReadingState{
		Question:        question,
		SpreadType:      spreadType,
		SessionID:       sid,
		CardsInfo:       cardsInfo,
		Interpretations: []agent.CardInterpretation{},
		Status:          "success",
	}

	if s.Graph != nil {
		result, err := s.Graph.Invoke(context.Background(), state)
		if err != nil {
			return nil, err
		}
		state = result
	} else {
		state.Interpretations = make([]agent.CardInterpretation, len(cardsInfo))
		for i, ci := range cardsInfo {
			state.Interpretations[i] = agent.CardInterpretation{
				CardID:         ci.ID,
				CardName:       ci.Name,
				IsReversed:     ci.IsReversed,
				Position:       ci.Position,
				Interpretation: agent.GetBasicInterpretation(ci.Card, ci.IsReversed),
				Status:         "fallback",
			}
		}
	}

	for i, interp := range state.Interpretations {
		s.DB.Model(&db.ReadingCard{}).Where("id = ?", readingCards[i].ID).Update("interpretation", interp.Interpretation)
	}

	if state.Synthesis != "" && spreadType == "three" {
		s.DB.Model(&db.Reading{}).Where("id = ?", reading.ID).Update("synthesis", state.Synthesis)
	}

	return s.GetReadingByID(reading.ID)
}

func (s *ReadingService) GetReadingByID(id uint) (*ReadingResult, error) {
	var reading db.Reading
	if err := s.DB.Preload("Cards").Preload("Cards.Card").First(&reading, id).Error; err != nil {
		return nil, err
	}

	cards := make([]ReadingCardResponse, len(reading.Cards))
	for i, rc := range reading.Cards {
		cards[i] = ReadingCardResponse{
			CardID:         rc.CardID,
			Name:           rc.Card.Name,
			Position:       rc.Position,
			IsReversed:     rc.IsReversed,
			Interpretation: rc.Interpretation,
			ImageURL:       rc.Card.ImageURL,

			ArcanaType:      rc.Card.ArcanaType,
			Suit:            rc.Card.Suit,
			Number:          rc.Card.Number,
			Keywords:        rc.Card.Keywords,
			Element:         rc.Card.Element,
			ZodiacSign:      rc.Card.ZodiacSign,
			MeaningUpright:  rc.Card.MeaningUpright,
			MeaningReversed: rc.Card.MeaningReversed,
			Description:     rc.Card.Description,
		}
	}

	synthesis := reading.Synthesis
	if synthesis == "" && len(cards) > 0 {
		// 兼容历史数据：旧版本把综合解读拼在最后一张牌的解读末尾，
		// 不拆开的话同一段综合解读会同时出现在牌面卡片和综合解读面板里。
		last := &cards[len(cards)-1]
		if idx := strings.Index(last.Interpretation, synthesisMarker); idx >= 0 {
			synthesis = strings.TrimSpace(last.Interpretation[idx+len(synthesisMarker):])
			last.Interpretation = strings.TrimSpace(last.Interpretation[:idx])
		}
	}

	return &ReadingResult{
		ReadingID:  reading.ID,
		SessionID:  reading.SessionID,
		Question:   reading.Question,
		SpreadType: reading.SpreadType,
		Synthesis:  synthesis,
		CreatedAt:  &reading.CreatedAt,
		Cards:      cards,
	}, nil
}
