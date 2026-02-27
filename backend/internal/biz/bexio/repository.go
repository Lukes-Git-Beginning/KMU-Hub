package bexio

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for Bexio integration data access.
type Repository interface {
	// Sync Config
	GetSyncConfig(ctx context.Context, configID uuid.UUID) (*models.BexioSyncConfig, error)
	UpsertSyncConfig(ctx context.Context, config *models.BexioSyncConfig) error
	UpdateLastSyncTime(ctx context.Context, configID uuid.UUID, syncType string, syncedAt time.Time) error

	// Entity Mappings
	GetEntityMapping(ctx context.Context, configID uuid.UUID, entityType string, kmuhubID uuid.UUID) (*models.BexioEntityMapping, error)
	GetEntityMappingByBexioID(ctx context.Context, configID uuid.UUID, entityType string, bexioID int) (*models.BexioEntityMapping, error)
	UpsertEntityMapping(ctx context.Context, mapping *models.BexioEntityMapping) error
	ListEntityMappings(ctx context.Context, configID uuid.UUID, entityType string) ([]models.BexioEntityMapping, error)
	DeleteEntityMapping(ctx context.Context, id uuid.UUID) error

	// Field Mappings
	GetFieldMappings(ctx context.Context, configID uuid.UUID, entityType string) (*models.BexioFieldMapping, error)
	UpsertFieldMappings(ctx context.Context, mapping *models.BexioFieldMapping) error

	// Sync Log
	CreateSyncLog(ctx context.Context, log *models.BexioSyncLog) error
	UpdateSyncLog(ctx context.Context, log *models.BexioSyncLog) error
	ListSyncLogs(ctx context.Context, configID uuid.UUID, limit int) ([]models.BexioSyncLog, error)
	GetLatestSyncLog(ctx context.Context, configID uuid.UUID, syncType string) (*models.BexioSyncLog, error)
}
