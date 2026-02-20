package team

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for team inbox persistence.
type Repository interface {
	// CreateTeamInbox persists a new team inbox.
	CreateTeamInbox(ctx context.Context, inbox *models.TeamInbox) error

	// UpdateTeamInbox updates an existing team inbox.
	UpdateTeamInbox(ctx context.Context, inbox *models.TeamInbox) error

	// DeleteTeamInbox deletes a team inbox by ID.
	DeleteTeamInbox(ctx context.Context, id uuid.UUID) error

	// GetTeamInbox retrieves a team inbox by ID.
	GetTeamInbox(ctx context.Context, id uuid.UUID) (*models.TeamInbox, error)

	// ListByUser returns all team inboxes where the user is a member.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.TeamInbox, error)

	// AddMember adds a user as a member of a team inbox.
	AddMember(ctx context.Context, member *models.TeamInboxMember) error

	// RemoveMember removes a user from a team inbox.
	RemoveMember(ctx context.Context, teamInboxID, userID uuid.UUID) error

	// ListMembers returns all members of a team inbox.
	ListMembers(ctx context.Context, teamInboxID uuid.UUID) ([]*models.TeamInboxMember, error)

	// IsMember checks if a user is a member of a team inbox.
	IsMember(ctx context.Context, teamInboxID, userID uuid.UUID) (bool, error)

	// GetMemberRole returns the role of a user in a team inbox, or empty string if not a member.
	GetMemberRole(ctx context.Context, teamInboxID, userID uuid.UUID) (string, error)

	// IncrementAssigneeIndex atomically increments and returns the next assignee index for round-robin.
	IncrementAssigneeIndex(ctx context.Context, teamInboxID uuid.UUID) (int, error)

	// GetMemberCount returns the number of members in a team inbox.
	GetMemberCount(ctx context.Context, teamInboxID uuid.UUID) (int, error)

	// CountAdmins returns the number of admin members in a team inbox.
	CountAdmins(ctx context.Context, teamInboxID uuid.UUID) (int, error)
}
