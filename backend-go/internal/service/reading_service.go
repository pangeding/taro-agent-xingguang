package service

import (
	"context"
	"math/rand"
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
	CardID       uint    `json:"card_id"`
	Name         string  `json:"name"`
	Position     int     `json:"position"`
	IsReversed   bool    `json:"is_reversed"`
	Interpretation string `json:"interpretation"`
	ImageURL     *string `json:"image_url"`
}

type ReadingResult struct {
	ReadingID  uint                   `json:"reading_id"`
	SessionID  string                 `json:"session_id"`
	Question   string                 `json:"question"`
	SpreadType string                 `json:"spread_type"`
	CreatedAt  *time.Time             `json:"created_at"`
	Cards      []ReadingCardResponse  `json:"cards"`
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

func (s *ReadingService) CreateReading(question, spreadType string, sessionID *string) (*ReadingResult, error) {
	sid := uuid.New().String()
	if sessionID != nil && *sessionID != "" {
		sid = *sessionID
	}

	reading := db.Reading{SessionID: sid, Question: question, SpreadType: spreadType}
	s.DB.Create(&reading)

	count := cardCount(spreadType)
	drawnCards, err := s.DrawCards(count)
	if err != nil {
		return nil, err
	}

	readingCardIDs := make([]uint, len(drawnCards))
	for i, dc := range drawnCards {
		rc := db.ReadingCard{
			ReadingID:  reading.ID,
			CardID:     dc.ID,
			Position:   i,
			IsReversed: dc.IsReversed,
		}
		s.DB.Create(&rc)
		readingCardIDs[i] = rc.ID
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

	var resultCards []db.ReadingCard
	s.DB.Where("id IN ?", readingCardIDs).Find(&resultCards)

	for i := range resultCards {
		rc := &resultCards[i]
		card := cardMap[rc.CardID]
		rc.Interpretation = agent.GetBasicInterpretation(card, rc.IsReversed)
		s.DB.Save(rc)
	}

	return s.GetReadingByID(reading.ID)
}

func (s *ReadingService) CreateReadingLangGraph(question, spreadType string, sessionID, modelName *string) (*ReadingResult, error) {
	sid := uuid.New().String()
	if sessionID != nil && *sessionID != "" {
		sid = *sessionID
	}

	reading := db.Reading{SessionID: sid, Question: question, SpreadType: spreadType}
	s.DB.Create(&reading)

	count := cardCount(spreadType)
	drawnCards, err := s.DrawCards(count)
	if err != nil {
		return nil, err
	}

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
		lastID := readingCards[len(readingCards)-1].ID
		var lastRC db.ReadingCard
		s.DB.First(&lastRC, lastID)
		lastRC.Interpretation += "\n\n---\n**综合解读**: " + state.Synthesis
		s.DB.Save(&lastRC)
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
			CardID:       rc.CardID,
			Name:         rc.Card.Name,
			Position:     rc.Position,
			IsReversed:   rc.IsReversed,
			Interpretation: rc.Interpretation,
			ImageURL:     rc.Card.ImageURL,
		}
	}

	return &ReadingResult{
		ReadingID:  reading.ID,
		SessionID:  reading.SessionID,
		Question:   reading.Question,
		SpreadType: reading.SpreadType,
		CreatedAt:  &reading.CreatedAt,
		Cards:      cards,
	}, nil
}
