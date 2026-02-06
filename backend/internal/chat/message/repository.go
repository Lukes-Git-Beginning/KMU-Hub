package message

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for message persistence
type Repository interface {
	// Message CRUD
	Create(ctx context.Context, message *models.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error)
	List(ctx context.Context, filter ListFilter) ([]*models.MessageWithSender, error)
	Update(ctx context.Context, message *models.Message) error
	Delete(ctx context.Context, id uuid.UUID) error // Soft delete

	// Channel checks
	ChannelExists(ctx context.Context, channelID uuid.UUID) (bool, error)
	IsChannelArchived(ctx context.Context, channelID uuid.UUID) (bool, error)
	IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	GetMemberRole(ctx context.Context, channelID, userID uuid.UUID) (models.ChannelRole, error)

	// User info
	GetUserInfo(ctx context.Context, userID uuid.UUID) (firstName, lastName string, err error)
}

// ListFilter contains filtering options for listing messages
type ListFilter struct {
	ChannelID uuid.UUID
	Limit     int
	Before    *uuid.UUID // Get messages before this message ID (by created_at)
	After     *uuid.UUID // Get messages after this message ID (by created_at)
}
