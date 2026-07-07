package service

import (
	"backend-go/internal/db"

	"gorm.io/gorm"
)

type CardItem struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	ArcanaType string  `json:"arcana_type"`
	Suit       *string `json:"suit"`
	Number     *int    `json:"number"`
	Keywords   string  `json:"keywords"`
	ImageURL   *string `json:"image_url"`
}

type CardDetail struct {
	ID              uint    `json:"id"`
	Name            string  `json:"name"`
	ArcanaType      string  `json:"arcana_type"`
	Suit            *string `json:"suit"`
	Number          *int    `json:"number"`
	MeaningUpright  string  `json:"meaning_upright"`
	MeaningReversed string  `json:"meaning_reversed"`
	Keywords        string  `json:"keywords"`
	Element         *string `json:"element"`
	ZodiacSign      *string `json:"zodiac_sign"`
	ImageURL        *string `json:"image_url"`
	Description     string  `json:"description"`
}

type RandomCardItem struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	ArcanaType string  `json:"arcana_type"`
	Suit       *string `json:"suit"`
	Number     *int    `json:"number"`
	IsReversed bool    `json:"is_reversed"`
	Meaning    string  `json:"meaning"`
	Keywords   string  `json:"keywords"`
	ImageURL   *string `json:"image_url"`
}

type CardService struct {
	DB *gorm.DB
}

func NewCardService(db *gorm.DB) *CardService {
	return &CardService{DB: db}
}

func (s *CardService) GetAllCards() []CardItem {
	var cards []db.TarotCard
	s.DB.Find(&cards)
	result := make([]CardItem, len(cards))
	for i, c := range cards {
		result[i] = CardItem{
			ID:         c.ID,
			Name:       c.Name,
			ArcanaType: c.ArcanaType,
			Suit:       c.Suit,
			Number:     c.Number,
			Keywords:   c.Keywords,
			ImageURL:   c.ImageURL,
		}
	}
	return result
}

func (s *CardService) GetCardByID(id uint) (*CardDetail, error) {
	var card db.TarotCard
	if err := s.DB.First(&card, id).Error; err != nil {
		return nil, err
	}
	return &CardDetail{
		ID:              card.ID,
		Name:            card.Name,
		ArcanaType:      card.ArcanaType,
		Suit:            card.Suit,
		Number:          card.Number,
		MeaningUpright:  card.MeaningUpright,
		MeaningReversed: card.MeaningReversed,
		Keywords:        card.Keywords,
		Element:         card.Element,
		ZodiacSign:      card.ZodiacSign,
		ImageURL:        card.ImageURL,
		Description:     card.Description,
	}, nil
}

func (s *CardService) GetRandomCard() (*RandomCardItem, error) {
	var card db.TarotCard
	if err := s.DB.Order("RANDOM()").First(&card).Error; err != nil {
		return nil, err
	}
	isReversed := false
	var seed int64
	s.DB.Raw("SELECT ABS(RANDOM()) % 2").Row().Scan(&seed)
	isReversed = seed == 1

	meaning := card.MeaningUpright
	if isReversed {
		meaning = card.MeaningReversed
	}
	return &RandomCardItem{
		ID:         card.ID,
		Name:       card.Name,
		ArcanaType: card.ArcanaType,
		Suit:       card.Suit,
		Number:     card.Number,
		IsReversed: isReversed,
		Meaning:    meaning,
		Keywords:   card.Keywords,
		ImageURL:   card.ImageURL,
	}, nil
}
