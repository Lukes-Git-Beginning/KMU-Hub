package message

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles message business logic
type Service struct {
	repo Repository
}

// NewService creates a new message service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a message
type CreateInput struct {
	ChannelID uuid.UUID
	Content   string
	CreatedBy uuid.UUID
}

// Create creates a new message
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.MessageWithSender, error) {
	// Validate content
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, ErrContentRequired
	}

	// Check channel exists
	exists, err := s.repo.ChannelExists(ctx, input.ChannelID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrChannelNotFound
	}

	// Check channel not archived
	archived, err := s.repo.IsChannelArchived(ctx, input.ChannelID)
	if err != nil {
		return nil, err
	}
	if archived {
		return nil, ErrChannelArchived
	}

	// Check user is a member
	isMember, err := s.repo.IsMember(ctx, input.ChannelID, input.CreatedBy)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotChannelMember
	}

	now := time.Now()
	message := &models.Message{
		ID:        uuid.New(),
		ChannelID: input.ChannelID,
		Content:   content,
		IsDeleted: false,
		EditedAt:  nil,
		CreatedBy: input.CreatedBy,
		CreatedAt: now,
	}

	if createErr := s.repo.Create(ctx, message); createErr != nil {
		return nil, createErr
	}

	slog.Info("message created",
		"message_id", message.ID,
		"channel_id", message.ChannelID,
		"created_by", message.CreatedBy,
	)

	return s.getWithSender(ctx, message)
}

// GetByID retrieves a message by ID
func (s *Service) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.MessageWithSender, error) {
	message, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check user is a member of the channel
	isMember, memberErr := s.repo.IsMember(ctx, message.ChannelID, userID)
	if memberErr != nil {
		return nil, memberErr
	}
	if !isMember {
		return nil, ErrNotChannelMember
	}

	return s.getWithSender(ctx, message)
}

// ListInput contains options for listing messages
type ListInput struct {
	ChannelID uuid.UUID
	UserID    uuid.UUID // For membership check
	Limit     int
	Before    *uuid.UUID
	After     *uuid.UUID
}

// List retrieves messages from a channel
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.MessageWithSender, bool, error) {
	// Check channel exists
	exists, err := s.repo.ChannelExists(ctx, input.ChannelID)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, ErrChannelNotFound
	}

	// Check user is a member
	isMember, memberErr := s.repo.IsMember(ctx, input.ChannelID, input.UserID)
	if memberErr != nil {
		return nil, false, memberErr
	}
	if !isMember {
		return nil, false, ErrNotChannelMember
	}

	if input.Limit < 1 || input.Limit > 100 {
		input.Limit = 50
	}

	filter := ListFilter{
		ChannelID: input.ChannelID,
		Limit:     input.Limit + 1, // Fetch one extra to determine if there are more
		Before:    input.Before,
		After:     input.After,
	}

	messages, listErr := s.repo.List(ctx, filter)
	if listErr != nil {
		return nil, false, listErr
	}

	// Check if there are more messages
	hasMore := len(messages) > input.Limit
	if hasMore {
		messages = messages[:input.Limit] // Remove the extra
	}

	return messages, hasMore, nil
}

// UpdateInput contains the data for updating a message
type UpdateInput struct {
	Content string
}

// Update updates an existing message
func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, input UpdateInput) (*models.MessageWithSender, error) {
	message, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if message.IsDeleted {
		return nil, ErrMessageDeleted
	}

	// Only author can edit
	if message.CreatedBy != userID {
		return nil, ErrNotMessageAuthor
	}

	// Validate content
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, ErrContentRequired
	}

	now := time.Now()
	message.Content = content
	message.EditedAt = &now

	if updateErr := s.repo.Update(ctx, message); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("message updated", "message_id", message.ID, "updated_by", userID)

	return s.getWithSender(ctx, message)
}

// Delete soft-deletes a message
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	message, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if message.IsDeleted {
		return nil // Already deleted, idempotent
	}

	// Check authorization: author or channel admin/owner
	if message.CreatedBy != userID {
		role, roleErr := s.repo.GetMemberRole(ctx, message.ChannelID, userID)
		if roleErr != nil {
			return ErrNotChannelMember
		}
		if !role.CanModerate() {
			return ErrNotAuthorized
		}
	}

	if deleteErr := s.repo.Delete(ctx, id); deleteErr != nil {
		return deleteErr
	}

	slog.Info("message deleted",
		"message_id", message.ID,
		"channel_id", message.ChannelID,
		"deleted_by", userID,
	)

	return nil
}

// getWithSender loads sender info for a message
func (s *Service) getWithSender(ctx context.Context, message *models.Message) (*models.MessageWithSender, error) {
	firstName, lastName, err := s.repo.GetUserInfo(ctx, message.CreatedBy)
	if err != nil {
		return nil, err
	}

	return &models.MessageWithSender{
		Message:         *message,
		SenderFirstName: firstName,
		SenderLastName:  lastName,
	}, nil
}
