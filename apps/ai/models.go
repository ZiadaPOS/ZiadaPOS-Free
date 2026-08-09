// Package ai implements the Ziada AI assistant (conversational BI).
package ai

import (
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/models"
)

// Conversation is a thread of messages between a user and Ziada AI.
type Conversation struct {
	models.BaseModel
	UserID       uuid.UUID `gorm:"type:uuid;not null;index:idx_conv_user_active,priority:1" json:"user_id"`
	StoreID      uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	OrgID        uuid.UUID `gorm:"type:uuid;not null;index" json:"organisation_id"`
	Title        string    `gorm:"size:200;default:New conversation" json:"title"`
	FirstMessage string    `gorm:"size:200;default:''" json:"first_message_preview"`
	IsActive     bool      `gorm:"default:true;index:idx_conv_user_active,priority:2" json:"is_active"`
}

// TableName maps to the ai_conversations table.
func (Conversation) TableName() string { return "ai_conversations" }

// Message is one chat turn within a conversation.
type Message struct {
	models.BaseModel
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index" json:"conversation_id"`
	Role           string    `gorm:"size:20;not null" json:"role"` // user | assistant | system
	Content        string    `gorm:"type:text;not null" json:"content"`
	ModelUsed      string    `gorm:"size:100;default:''" json:"model_used"`
	PromptTokens   int       `gorm:"default:0" json:"prompt_tokens"`
	Completion     int       `gorm:"default:0" json:"completion_tokens"`
	SourcesUsed    string    `gorm:"size:500;default:''" json:"sources_used"`
}

// TableName maps to the ai_messages table.
func (Message) TableName() string { return "ai_messages" }

// AICredit tracks the org's monthly AI allowance.
type AICredit struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OrgID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_org_month" json:"organisation_id"`
	Year      int       `gorm:"uniqueIndex:idx_org_month" json:"year"`
	Month     int       `gorm:"uniqueIndex:idx_org_month" json:"month"`
	Used      int       `gorm:"default:0" json:"used"`
	Allocated int       `gorm:"default:0" json:"allocated"`
}

// TableName maps to the shared ai_credits table.
func (AICredit) TableName() string { return "ai_credits" }

// Remaining returns unused credits for the month.
func (a *AICredit) Remaining() int {
	if r := a.Allocated - a.Used; r > 0 {
		return r
	}
	return 0
}
