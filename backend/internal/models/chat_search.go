package models

import (
	"time"

	"github.com/google/uuid"
)

// ChatSearchResultType distinguishes message vs file results
type ChatSearchResultType string

const (
	ChatSearchResultMessage ChatSearchResultType = "message"
	ChatSearchResultFile    ChatSearchResultType = "file"
)

// ChatSearchResult represents a single search result (message or file)
type ChatSearchResult struct {
	Type        ChatSearchResultType `json:"type"`
	ID          uuid.UUID            `json:"id"`
	ChannelID   uuid.UUID            `json:"channel_id"`
	ChannelName string               `json:"channel_name"`
	Score       float64              `json:"score"`
	Snippet     string               `json:"snippet"`
	CreatedAt   time.Time            `json:"created_at"`
	CreatedBy   uuid.UUID            `json:"created_by"`
	FirstName   string               `json:"first_name"`
	LastName    string               `json:"last_name"`

	// Message-specific (nil for files)
	MessageID       *uuid.UUID `json:"message_id,omitempty"`
	ParentMessageID *uuid.UUID `json:"parent_message_id,omitempty"`
	Lang            *string    `json:"lang,omitempty"`

	// File-specific (nil for messages)
	FileID   *uuid.UUID `json:"file_id,omitempty"`
	Filename *string    `json:"filename,omitempty"`
	MimeType *string    `json:"mime_type,omitempty"`
	FileSize *int64     `json:"file_size,omitempty"`
}
