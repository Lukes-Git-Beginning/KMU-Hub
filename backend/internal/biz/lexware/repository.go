package lexware

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for Lexware integration data access.
type Repository interface {
	// Sync Config
	GetSyncConfig(ctx context.Context, configID uuid.UUID) (*models.LexwareSyncConfig, error)
	UpsertSyncConfig(ctx context.Context, config *models.LexwareSyncConfig) error
	UpdateLastSyncTime(ctx context.Context, configID uuid.UUID, syncType string, syncedAt time.Time) error

	// Entity Mappings (LexwareID is string, not int)
	GetEntityMapping(ctx context.Context, configID uuid.UUID, entityType string, kmuhubID uuid.UUID) (*models.LexwareEntityMapping, error)
	GetEntityMappingByLexwareID(ctx context.Context, configID uuid.UUID, entityType string, lexwareID string) (*models.LexwareEntityMapping, error)
	UpsertEntityMapping(ctx context.Context, mapping *models.LexwareEntityMapping) error
	ListEntityMappings(ctx context.Context, configID uuid.UUID, entityType string) ([]models.LexwareEntityMapping, error)
	DeleteEntityMapping(ctx context.Context, id uuid.UUID) error

	// Field Mappings
	GetFieldMappings(ctx context.Context, configID uuid.UUID, entityType string) (*models.LexwareFieldMapping, error)
	UpsertFieldMappings(ctx context.Context, mapping *models.LexwareFieldMapping) error

	// Sync Log
	CreateSyncLog(ctx context.Context, log *models.LexwareSyncLog) error
	UpdateSyncLog(ctx context.Context, log *models.LexwareSyncLog) error
	ListSyncLogs(ctx context.Context, configID uuid.UUID, limit int) ([]models.LexwareSyncLog, error)
	GetLatestSyncLog(ctx context.Context, configID uuid.UUID, syncType string) (*models.LexwareSyncLog, error)

	// Webhook Subscriptions
	UpsertWebhookSubscription(ctx context.Context, tenantID, configID uuid.UUID, subscriptionID, eventType, callbackURL string) error
	ListWebhookSubscriptions(ctx context.Context, configID uuid.UUID) ([]WebhookSubscriptionRecord, error)
	DeleteWebhookSubscription(ctx context.Context, id uuid.UUID) error
}

// WebhookSubscriptionRecord represents a stored webhook subscription.
type WebhookSubscriptionRecord struct {
	ID             uuid.UUID `json:"id"`
	ConfigID       uuid.UUID `json:"config_id"`
	SubscriptionID string    `json:"subscription_id"`
	EventType      string    `json:"event_type"`
	CallbackURL    string    `json:"callback_url"`
	IsActive       bool      `json:"is_active"`
}

// IntegrationConfigRepo manages integration_configs rows for platform='lexware'.
type IntegrationConfigRepo interface {
	GetByPlatform(ctx context.Context, platform string) (*IntegrationConfig, error)
	Upsert(ctx context.Context, config *IntegrationConfig) error
	Deactivate(ctx context.Context, id uuid.UUID) error
}

// IntegrationConfig represents a row in the integration_configs table.
type IntegrationConfig struct {
	ID                  uuid.UUID      `json:"id"`
	TenantID            uuid.UUID      `json:"tenant_id"`
	Platform            string         `json:"platform"`
	IsActive            bool           `json:"is_active"`
	CredentialsVaultKey string         `json:"credentials_vault_key"`
	Metadata            map[string]any `json:"metadata"`
	CreatedBy           uuid.UUID      `json:"created_by"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// ConnectionStatus reports the current Lexware connection state.
type ConnectionStatus struct {
	Connected   bool       `json:"connected"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
	ConfigID    *uuid.UUID `json:"config_id,omitempty"`
}
