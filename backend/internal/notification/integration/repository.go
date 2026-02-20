package integration

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for integration persistence.
type Repository interface {
	// Config CRUD
	CreateConfig(ctx context.Context, cfg *IntegrationConfig) error
	GetConfigByPlatform(ctx context.Context, platform string) (*IntegrationConfig, error)
	UpdateConfig(ctx context.Context, cfg *IntegrationConfig) error
	ListConfigs(ctx context.Context) ([]*IntegrationConfig, error)
	DeleteConfig(ctx context.Context, id uuid.UUID) error

	// Channel Mapping CRUD
	CreateMapping(ctx context.Context, m *ChannelMapping) error
	GetMapping(ctx context.Context, id uuid.UUID) (*ChannelMapping, error)
	ListMappingsByConfig(ctx context.Context, configID uuid.UUID) ([]*ChannelMapping, error)
	ListActiveMappingsForModule(ctx context.Context, moduleID string) ([]*ChannelMapping, error)
	UpdateMapping(ctx context.Context, m *ChannelMapping) error
	DeleteMapping(ctx context.Context, id uuid.UUID) error

	// Account Links
	CreateAccountLink(ctx context.Context, link *AccountLink) error
	GetAccountLink(ctx context.Context, platform, externalUserID string) (*AccountLink, error)
	GetAccountLinkByKMUHubUser(ctx context.Context, platform string, kmuhubUserID uuid.UUID) (*AccountLink, error)
	DeleteAccountLink(ctx context.Context, id uuid.UUID) error

	// Link Tokens
	CreateLinkToken(ctx context.Context, token *LinkToken) error
	GetLinkTokenByHash(ctx context.Context, tokenHash string) (*LinkToken, error)
	MarkLinkTokenUsed(ctx context.Context, id uuid.UUID) error
	CleanupExpiredTokens(ctx context.Context) (int, error)

	// Delivery Log
	LogDelivery(ctx context.Context, entry *DeliveryLogEntry) error
	GetRecentFailures(ctx context.Context, mappingID uuid.UUID, limit int) ([]*DeliveryLogEntry, error)
	CleanupOldLogs(ctx context.Context, olderThan time.Time) (int, error)
}
