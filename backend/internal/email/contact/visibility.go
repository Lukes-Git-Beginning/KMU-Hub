package contact

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
)

// ContactRepository defines the minimal repository interface needed by VisibilityService.
type ContactRepository interface {
	UpdateVisibility(ctx context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID) error
}

// VisibilityService manages contact visibility (shared/personal) with admin override.
type VisibilityService struct {
	contactRepo ContactRepository
	logger      *slog.Logger
}

// NewVisibilityService creates a new VisibilityService.
func NewVisibilityService(repo ContactRepository, logger *slog.Logger) *VisibilityService {
	if logger == nil {
		logger = slog.Default()
	}
	return &VisibilityService{
		contactRepo: repo,
		logger:      logger,
	}
}

// Visibility constants
const (
	VisibilityShared   = "shared"
	VisibilityPersonal = "personal"
)

// Visibility errors
var (
	ErrInvalidVisibility = errors.New("visibility must be 'shared' or 'personal'")
	ErrNotAuthorized     = errors.New("not authorized to change visibility")
)

// SetVisibility sets the visibility of a contact. The userID must be the owner of the contact.
func (s *VisibilityService) SetVisibility(ctx context.Context, contactID uuid.UUID, visibility string, userID uuid.UUID) error {
	if !isValidVisibility(visibility) {
		return ErrInvalidVisibility
	}

	var ownerID *uuid.UUID
	if visibility == VisibilityPersonal {
		ownerID = &userID
	}

	if err := s.contactRepo.UpdateVisibility(ctx, contactID, visibility, ownerID); err != nil {
		return err
	}

	s.logger.Info("contact visibility updated",
		"contact_id", contactID,
		"visibility", visibility,
		"user_id", userID,
	)

	return nil
}

// SetOwner sets the owner of a personal contact.
func (s *VisibilityService) SetOwner(ctx context.Context, contactID uuid.UUID, ownerID uuid.UUID) error {
	if err := s.contactRepo.UpdateVisibility(ctx, contactID, VisibilityPersonal, &ownerID); err != nil {
		return err
	}

	s.logger.Info("contact owner updated",
		"contact_id", contactID,
		"owner_id", ownerID,
	)

	return nil
}

// AdminOverride allows an admin to change visibility regardless of ownership.
func (s *VisibilityService) AdminOverride(ctx context.Context, contactID uuid.UUID, visibility string) error {
	if !isValidVisibility(visibility) {
		return ErrInvalidVisibility
	}

	// Admin override clears the owner when setting to shared
	var ownerID *uuid.UUID
	if err := s.contactRepo.UpdateVisibility(ctx, contactID, visibility, ownerID); err != nil {
		return err
	}

	s.logger.Info("admin visibility override",
		"contact_id", contactID,
		"visibility", visibility,
	)

	return nil
}

// isValidVisibility checks if a visibility string is valid.
func isValidVisibility(v string) bool {
	return v == VisibilityShared || v == VisibilityPersonal
}
