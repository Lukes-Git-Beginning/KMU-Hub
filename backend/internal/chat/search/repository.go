package search

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for chat search persistence
type Repository interface {
	SearchMessages(ctx context.Context, query string, lang string, channelIDs []uuid.UUID, limit, offset int) ([]models.ChatSearchResult, int, error)
	SearchFiles(ctx context.Context, query string, channelIDs []uuid.UUID, limit, offset int) ([]models.ChatSearchResult, int, error)
	GetUserChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
