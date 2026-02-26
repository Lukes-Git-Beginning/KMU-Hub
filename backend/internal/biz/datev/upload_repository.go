package datev

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

type UploadRepository interface {
	GetUploadConfig(ctx context.Context, configID uuid.UUID) (*models.DatevUploadConfig, error)
	UpsertUploadConfig(ctx context.Context, config *models.DatevUploadConfig) error

	CreateUploadLog(ctx context.Context, log *models.DatevUploadLog) error
	UpdateUploadLog(ctx context.Context, log *models.DatevUploadLog) error
	ListUploadLogs(ctx context.Context, configID uuid.UUID, limit int) ([]models.DatevUploadLog, error)
}

type IntegrationConfigRepo interface {
	GetByPlatform(ctx context.Context, platform string) (*IntegrationConfig, error)
	Upsert(ctx context.Context, config *IntegrationConfig) error
	Deactivate(ctx context.Context, id uuid.UUID) error
}

type IntegrationConfig struct {
	ID                  uuid.UUID      `json:"id"`
	Platform            string         `json:"platform"`
	IsActive            bool           `json:"is_active"`
	CredentialsVaultKey string         `json:"credentials_vault_key"`
	Metadata            map[string]any `json:"metadata"`
	CreatedBy           uuid.UUID      `json:"created_by"`
}
