package db

import (
	"time"
)

type TarotCard struct {
	ID              uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string  `gorm:"size:50;uniqueIndex" json:"name"`
	ArcanaType      string  `gorm:"size:10" json:"arcana_type"`
	Suit            *string `gorm:"size:20" json:"suit"`
	Number          *int    `json:"number"`
	MeaningUpright  string  `gorm:"type:text" json:"meaning_upright"`
	MeaningReversed string  `gorm:"type:text" json:"meaning_reversed"`
	Keywords        string  `gorm:"size:255" json:"keywords"`
	Element         *string `gorm:"size:20" json:"element"`
	ZodiacSign      *string `gorm:"size:20" json:"zodiac_sign"`
	ImageURL        *string `gorm:"size:255" json:"image_url"`
	Description     string  `gorm:"type:text" json:"description"`
}

func (TarotCard) TableName() string {
	return "tarot_cards"
}

type Reading struct {
	ID         uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID  string        `gorm:"size:100;index" json:"session_id"`
	Question   string        `gorm:"type:text" json:"question"`
	SpreadType string        `gorm:"size:50;default:single" json:"spread_type"`
	CreatedAt  time.Time     `gorm:"index;autoCreateTime" json:"created_at"`
	Cards      []ReadingCard `gorm:"foreignKey:ReadingID" json:"cards"`
}

func (Reading) TableName() string {
	return "readings"
}

type ReadingCard struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ReadingID      uint      `gorm:"index" json:"reading_id"`
	CardID         uint      `json:"card_id"`
	Position       int       `gorm:"default:0" json:"position"`
	IsReversed     bool      `gorm:"default:false" json:"is_reversed"`
	Interpretation string    `gorm:"type:text" json:"interpretation"`
	Card           TarotCard `gorm:"foreignKey:ID;references:CardID" json:"card"`
}

func (ReadingCard) TableName() string {
	return "reading_cards"
}

type Feedback struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ReadingID uint      `gorm:"index" json:"reading_id"`
	Rating    int       `json:"rating"`
	Comment   *string   `json:"comment"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Feedback) TableName() string {
	return "feedbacks"
}

type Conversation struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    string    `gorm:"size:100;index" json:"user_id"`
	Title     string    `gorm:"size:200;default:新对话" json:"title"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoCreateTime;autoUpdateTime" json:"updated_at"`
	Messages  []Message `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

func (Conversation) TableName() string {
	return "conversations"
}

type Message struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID uint      `gorm:"index" json:"conversation_id"`
	Role           string    `gorm:"size:20" json:"role"`
	Content        string    `gorm:"type:text" json:"content"`
	Type           string    `gorm:"size:20;default:text" json:"type"`
	ReadingID      *uint     `json:"reading_id"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Message) TableName() string {
	return "messages"
}
