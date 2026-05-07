package team

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles team inbox business logic including creation, membership,
// manual claim, and round-robin auto-assignment.
type Service struct {
	repo        Repository
	messageRepo message.Repository
}

// NewService creates a new team inbox service.
func NewService(repo Repository, messageRepo message.Repository) *Service {
	return &Service{
		repo:        repo,
		messageRepo: messageRepo,
	}
}

// CreateTeamInbox creates a new team inbox.
// The creator is automatically added as an admin member.
func (s *Service) CreateTeamInbox(ctx context.Context, inbox *models.TeamInbox) error {
	if inbox.Name == "" {
		return fmt.Errorf("team inbox name cannot be empty")
	}
	if inbox.ID == uuid.Nil {
		inbox.ID = uuid.New()
	}
	if inbox.AssignmentMode == "" {
		inbox.AssignmentMode = "manual"
	}
	if inbox.Visibility == "" {
		inbox.Visibility = "open"
	}

	if err := s.repo.CreateTeamInbox(ctx, inbox); err != nil {
		return fmt.Errorf("create team inbox: %w", err)
	}

	// Auto-add creator as admin member
	member := &models.TeamInboxMember{
		TeamInboxID: inbox.ID,
		UserID:      inbox.CreatedBy,
		Role:        "admin",
	}
	if err := s.repo.AddMember(ctx, member); err != nil {
		slog.Error("failed to add creator as admin member",
			"team_inbox_id", inbox.ID,
			"user_id", inbox.CreatedBy,
			"error", err,
		)
		return fmt.Errorf("add creator as admin: %w", err)
	}

	return nil
}

// UpdateTeamInbox updates a team inbox. Only admin members can update.
func (s *Service) UpdateTeamInbox(ctx context.Context, inbox *models.TeamInbox, userID uuid.UUID) error {
	role, err := s.repo.GetMemberRole(ctx, inbox.ID, userID)
	if err != nil {
		return err
	}
	if role != "admin" {
		return ErrNotTeamAdmin
	}

	return s.repo.UpdateTeamInbox(ctx, inbox)
}

// DeleteTeamInbox deletes a team inbox. Only admin members can delete.
func (s *Service) DeleteTeamInbox(ctx context.Context, tenantID, id uuid.UUID, userID uuid.UUID) error {
	role, err := s.repo.GetMemberRole(ctx, id, userID)
	if err != nil {
		return err
	}
	if role != "admin" {
		return ErrNotTeamAdmin
	}

	return s.repo.DeleteTeamInbox(ctx, tenantID, id)
}

// GetTeamInbox retrieves a team inbox by ID within a tenant.
func (s *Service) GetTeamInbox(ctx context.Context, tenantID, id uuid.UUID) (*models.TeamInbox, error) {
	return s.repo.GetTeamInbox(ctx, tenantID, id)
}

// ListByUser returns all team inboxes where the user is a member within a tenant.
func (s *Service) ListByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*models.TeamInbox, error) {
	return s.repo.ListByUser(ctx, tenantID, userID)
}

// ClaimMessage allows a user to manually claim an unassigned message in a team inbox.
// Manual claiming is not allowed in round-robin assignment mode.
func (s *Service) ClaimMessage(ctx context.Context, tenantID, teamInboxID uuid.UUID, messageID uuid.UUID, userID uuid.UUID) error {
	// Verify team inbox exists and check assignment mode
	inbox, err := s.repo.GetTeamInbox(ctx, tenantID, teamInboxID)
	if err != nil {
		return err
	}

	if inbox.AssignmentMode == "round_robin" {
		return ErrManualClaimInRoundRobin
	}

	// Verify user is a member
	isMember, err := s.repo.IsMember(ctx, teamInboxID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotTeamMember
	}

	// Atomic claim: only succeeds if assigned_to IS NULL
	assigned, err := s.messageRepo.AssignMessage(ctx, messageID, userID)
	if err != nil {
		return err
	}
	if !assigned {
		return message.ErrAlreadyAssigned
	}

	return nil
}

// AutoAssignMessage assigns a message using round-robin among team inbox members.
// If the selected member's claim fails (race condition), retries with the next index (max 3 retries).
func (s *Service) AutoAssignMessage(ctx context.Context, tenantID, teamInboxID uuid.UUID, messageID uuid.UUID) error {
	inbox, err := s.repo.GetTeamInbox(ctx, tenantID, teamInboxID)
	if err != nil {
		return err
	}

	if inbox.AssignmentMode != "round_robin" {
		// Not round-robin mode; skip auto-assign
		return nil
	}

	members, err := s.repo.ListMembers(ctx, teamInboxID)
	if err != nil {
		return fmt.Errorf("list members for auto-assign: %w", err)
	}

	if len(members) == 0 {
		slog.Warn("auto-assign: no members in team inbox",
			"team_inbox_id", teamInboxID,
		)
		return nil
	}

	// Retry up to 3 times for race conditions
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		index, err := s.repo.IncrementAssigneeIndex(ctx, tenantID, teamInboxID)
		if err != nil {
			return fmt.Errorf("increment assignee index: %w", err)
		}

		// Pick member at index % member_count
		memberIdx := (index - 1) % len(members) // -1 because index is post-increment
		if memberIdx < 0 {
			memberIdx = 0
		}
		selectedMember := members[memberIdx]

		assigned, err := s.messageRepo.AssignMessage(ctx, messageID, selectedMember.UserID)
		if err != nil {
			return fmt.Errorf("assign message: %w", err)
		}
		if assigned {
			slog.Info("auto-assigned message",
				"team_inbox_id", teamInboxID,
				"message_id", messageID,
				"assigned_to", selectedMember.UserID,
			)
			return nil
		}

		// Already assigned by someone else; retry with next index
		slog.Debug("auto-assign: message already claimed, retrying",
			"attempt", i+1,
			"message_id", messageID,
		)
	}

	slog.Warn("auto-assign: max retries exceeded, message remains unassigned",
		"team_inbox_id", teamInboxID,
		"message_id", messageID,
	)
	return nil
}

// AddMember adds a user as a member of a team inbox.
func (s *Service) AddMember(ctx context.Context, member *models.TeamInboxMember) error {
	if member.Role == "" {
		member.Role = "member"
	}
	return s.repo.AddMember(ctx, member)
}

// RemoveMember removes a user from a team inbox.
// Cannot remove the last admin.
func (s *Service) RemoveMember(ctx context.Context, teamInboxID, userID uuid.UUID) error {
	// Check if the user being removed is an admin
	role, err := s.repo.GetMemberRole(ctx, teamInboxID, userID)
	if err != nil {
		return err
	}
	if role == "" {
		return ErrNotTeamMember
	}

	// If removing an admin, ensure at least one admin remains
	if role == "admin" {
		adminCount, err := s.repo.CountAdmins(ctx, teamInboxID)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return ErrCannotRemoveLastAdmin
		}
	}

	return s.repo.RemoveMember(ctx, teamInboxID, userID)
}

// ListMembers returns all members of a team inbox.
func (s *Service) ListMembers(ctx context.Context, teamInboxID uuid.UUID) ([]*models.TeamInboxMember, error) {
	return s.repo.ListMembers(ctx, teamInboxID)
}

// IsMember checks if a user is a member of a team inbox.
func (s *Service) IsMember(ctx context.Context, teamInboxID, userID uuid.UUID) (bool, error) {
	return s.repo.IsMember(ctx, teamInboxID, userID)
}
