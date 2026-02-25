package guest

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the data access interface for guest sessions and config
type Repository interface {
	// Session CRUD
	CreateSession(ctx context.Context, session *GuestSession) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*GuestSession, error)
	GetSessionByID(ctx context.Context, id uuid.UUID) (*GuestSession, error)
	ListSessionsByChannel(ctx context.Context, channelID uuid.UUID) ([]*GuestSession, error)
	UpdateLastActivity(ctx context.Context, id uuid.UUID) error
	DeactivateSession(ctx context.Context, id uuid.UUID) error
	CleanupExpiredSessions(ctx context.Context) (int, error)

	// Channel Config CRUD
	CreateConfig(ctx context.Context, config *GuestChannelConfig) error
	GetConfigByChannel(ctx context.Context, channelID uuid.UUID) (*GuestChannelConfig, error)
	UpdateConfig(ctx context.Context, config *GuestChannelConfig) error
	DeleteConfig(ctx context.Context, channelID uuid.UUID) error
}
