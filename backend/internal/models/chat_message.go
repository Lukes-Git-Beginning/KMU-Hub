package models

import (
	"time"

	"github.com/google/uuid"
)

// Message represents a chat message
type Message struct {
	ID              uuid.UUID  `json:"id"`
	ChannelID       uuid.UUID  `json:"channel_id"`
	Content         string     `json:"content"`
	IsDeleted       bool       `json:"is_deleted"`
	EditedAt        *time.Time `json:"edited_at,omitempty"`
	ParentMessageID *uuid.UUID `json:"parent_message_id,omitempty"`
	ReplyCount      int        `json:"reply_count"`
	Lang            string     `json:"lang"`
	CreatedBy       uuid.UUID  `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
}

// MessageWithSender includes sender details for API responses
type MessageWithSender struct {
	Message
	SenderFirstName string                 `json:"sender_first_name"`
	SenderLastName  string                 `json:"sender_last_name"`
	Mentions        []MentionWithUser      `json:"mentions,omitempty"`
	Files           []ChatFileWithUploader `json:"files,omitempty"`
}
