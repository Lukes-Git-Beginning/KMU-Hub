package preference

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for notification preference persistence.
type Repository interface {
	// Preferences
	GetEventTypePreference(ctx context.Context, userID uuid.UUID, eventTypeKey string) (*models.NotificationPreference, error)
	GetModuleDefault(ctx context.Context, userID uuid.UUID, moduleID string) (*models.NotificationPreference, error)
	ListPreferences(ctx context.Context, userID uuid.UUID, moduleID *string) ([]*models.NotificationPreference, error)
	UpsertPreference(ctx context.Context, pref *models.NotificationPreference) error

	// Mutes
	IsResourceMuted(ctx context.Context, userID uuid.UUID, moduleID, resourceID string) (bool, error)
	CreateMute(ctx context.Context, mute *models.NotificationMute) error
	DeleteMute(ctx context.Context, userID uuid.UUID, moduleID, resourceID string) error
	ListMutes(ctx context.Context, userID uuid.UUID, moduleID *string, offset, limit int) ([]*models.NotificationMute, int, error)

	// Quiet Hours
	GetQuietHours(ctx context.Context, userID uuid.UUID) (*models.QuietHours, error)
	UpsertQuietHours(ctx context.Context, qh *models.QuietHours) error
}
