package models

import (
	"time"

	"github.com/google/uuid"
)

// ChatFile represents a file uploaded in a chat channel
type ChatFile struct {
	ID           uuid.UUID  `json:"id"`
	MessageID    *uuid.UUID `json:"message_id,omitempty"`
	ChannelID    uuid.UUID  `json:"channel_id"`
	Filename     string     `json:"filename"`
	MimeType     string     `json:"mime_type"`
	FileSize     int64      `json:"file_size"`
	StorageKey   string     `json:"storage_key"`
	ThumbnailKey *string    `json:"thumbnail_key,omitempty"`
	UploadedBy   uuid.UUID  `json:"uploaded_by"`
	IsDeleted    bool       `json:"is_deleted"`
	CreatedAt    time.Time  `json:"created_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// ChatFileWithUploader includes uploader details for API responses
type ChatFileWithUploader struct {
	ChatFile
	UploaderFirstName string `json:"uploader_first_name"`
	UploaderLastName  string `json:"uploader_last_name"`
}

// StorageQuota tracks org-level storage usage
type StorageQuota struct {
	ID        uuid.UUID `json:"id"`
	MaxBytes  int64     `json:"max_bytes"`
	UsedBytes int64     `json:"used_bytes"`
	UpdatedAt time.Time `json:"updated_at"`
}
