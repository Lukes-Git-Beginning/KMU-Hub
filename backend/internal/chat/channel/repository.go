package channel

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for channel persistence
type Repository interface {
	// Channel CRUD
	Create(ctx context.Context, channel *models.Channel) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Channel, int, error)
	Update(ctx context.Context, channel *models.Channel) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Membership operations
	AddMember(ctx context.Context, membership *models.ChannelMembership) error
	RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error
	GetMembership(ctx context.Context, channelID, userID uuid.UUID) (*models.ChannelMembership, error)
	UpdateMembership(ctx context.Context, membership *models.ChannelMembership) error
	ListMembers(ctx context.Context, channelID uuid.UUID, offset, limit int) ([]*models.ChannelMember, int, error)
	GetMemberCount(ctx context.Context, channelID uuid.UUID) (int, error)

	// User info for denormalization
	GetUserInfo(ctx context.Context, userID uuid.UUID) (firstName, lastName, email string, err error)
	UserExists(ctx context.Context, userID uuid.UUID) (bool, error)

	// Last message for channel list
	GetLastMessage(ctx context.Context, channelID uuid.UUID) (*models.MessageWithSender, error)
}

// ListFilter contains filtering options for listing channels
type ListFilter struct {
	UserID          uuid.UUID // Required: list channels this user is a member of
	IncludeArchived bool
	IsDM            *bool
	Search          string // Searches channel name
}
