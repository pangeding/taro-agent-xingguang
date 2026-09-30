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

// User 是登录主体。归属列 owner_id 全部指向 users.id。
//
// 密码相关字段一律 json:"-"，避免任何 handler 误序列化。
// Username 入库前必须经过 auth.NormalizeUsername（小写 + 去空白），
// 否则 "Admin" 与 "admin" 会同时存在，绕过保留字与唯一性检查。
type User struct {
	ID                 uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Username           string     `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash       string     `gorm:"size:100;not null" json:"-"`
	DisplayName        string     `gorm:"size:50" json:"display_name"`
	Role               string     `gorm:"size:20;not null;default:user" json:"role"` // user | admin
	Status             int8       `gorm:"not null;default:1" json:"status"`          // 1 启用 / 0 禁用
	MustChangePassword bool       `gorm:"not null;default:false" json:"must_change_password"`
	FailedAttempts     int        `gorm:"not null;default:0" json:"-"`
	LockedUntil        *time.Time `json:"-"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"autoCreateTime;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

// Session 是服务端会话。DB 只存 token 的 sha256，明文 token 只在 Cookie 里，
// 因此库被拖走也换不出可用会话。
type Session struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null"` // sha256 hex
	UserID    uint      `gorm:"index;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	UserAgent string    `gorm:"size:255"`
	IP        string    `gorm:"size:64"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (Session) TableName() string {
	return "sessions"
}

// Reading 是一次占卜记录。
//
// OwnerID：归属用户。注意 0 号用户不存在（自增从 1 起），
// 因此 owner_id = 0 恒不可见，是迁移未完成时的安全兜底。
//
// ConversationID：可空。历史数据靠 session_id 的 "conv_<id>" 约定反推关联，
// 这个列把它变成真实外键，迁移可回填 45/60 行。
type Reading struct {
	ID             uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerID        uint          `gorm:"index;not null;default:0" json:"owner_id"`
	ConversationID *uint         `gorm:"index" json:"conversation_id"`
	SessionID      string        `gorm:"size:100;index" json:"session_id"`
	Question       string        `gorm:"type:text" json:"question"`
	SpreadType     string        `gorm:"size:50;default:single" json:"spread_type"`
	Synthesis      string        `gorm:"type:text" json:"synthesis"`
	CreatedAt      time.Time     `gorm:"index;autoCreateTime" json:"created_at"`
	Cards          []ReadingCard `gorm:"foreignKey:ReadingID" json:"cards"`
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

// Feedback 当前全仓无任何读写代码，owner_id 仅为保持结构一致。
type Feedback struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerID   uint      `gorm:"index;not null;default:0" json:"owner_id"`
	ReadingID uint      `gorm:"index" json:"reading_id"`
	Rating    int       `json:"rating"`
	Comment   *string   `json:"comment"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Feedback) TableName() string {
	return "feedbacks"
}

// Conversation 是一个会话。
//
// OwnerID 取代了历史上的 UserID string（那里装的是浏览器 Cookie 里的匿名 UUID）。
// 老列在迁移脚本 --drop-legacy 时才删除，见 doc/2026-09-28-用户系统与数据隔离技术文档.md §5。
type Conversation struct {
	ID      uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerID uint   `gorm:"index;not null;default:0" json:"owner_id"`
	Title   string `gorm:"size:200;default:新对话" json:"title"`
	// Channel 区分会话归属的功能：chat（聊天页）/ reading（占卜页）。
	// 占卜页写入的会话不再出现在聊天页的列表里。
	Channel   string    `gorm:"size:20;default:chat;index" json:"channel"`
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
