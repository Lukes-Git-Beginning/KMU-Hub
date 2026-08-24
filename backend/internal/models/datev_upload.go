package models

import (
	"time"

	"github.com/google/uuid"
)

// DatevUploadConfig holds per-tenant DATEV API upload configuration.
type DatevUploadConfig struct {
	ID                uuid.UUID `json:"id"`
	ConfigID          uuid.UUID `json:"config_id"`
	ClientNumber      string    `json:"client_number"`
	AutoUploadEnabled bool      `json:"auto_upload_enabled"`
	UploadAfterExport bool      `json:"upload_after_export"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// DatevUploadLog records a single DATEV upload operation.
type DatevUploadLog struct {
	ID            uuid.UUID      `json:"id"`
	ConfigID      uuid.UUID      `json:"config_id"`
	UploadType    string         `json:"upload_type"`
	Status        string         `json:"status"`
	FileSize      int            `json:"file_size"`
	DocumentCount int            `json:"document_count"`
	ErrorMessage  *string        `json:"error_message,omitempty"`
	StartedAt     time.Time      `json:"started_at"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
	Metadata      map[string]any `json:"metadata"`
	// IsStale is derived at read time, never persisted: true when Status is
	// "uploading" and StartedAt is older than the worst-case real transfer
	// time (see datevUploadStaleThreshold in internal/biz/datev).
	IsStale bool `json:"is_stale"`
}
