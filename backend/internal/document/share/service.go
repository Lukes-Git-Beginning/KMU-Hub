package share

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Valid entity types for sharing.
var validEntityTypes = map[string]bool{
	models.ShareEntityFile:   true,
	models.ShareEntityFolder: true,
}

// Valid permission levels.
var validPermissions = map[string]bool{
	models.SharePermissionRead:  true,
	models.SharePermissionWrite: true,
}

// Service handles share business logic.
type Service struct {
	repo Repository
}

// NewService creates a new share service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ShareInput contains the data needed to share an entity.
type ShareInput struct {
	TenantID         uuid.UUID
	EntityType       string
	EntityID         uuid.UUID
	SharedWithUserID uuid.UUID
	Permission       string
	SharedBy         uuid.UUID
}

// ShareEntity creates a new share for a file or folder with a specific user.
func (s *Service) ShareEntity(ctx context.Context, input ShareInput) (*models.DocumentShare, error) {
	// Validate entity type
	if !validEntityTypes[input.EntityType] {
		return nil, ErrInvalidEntityType
	}

	// Validate permission
	if !validPermissions[input.Permission] {
		return nil, ErrInvalidPermission
	}

	// Cannot share with self
	if input.SharedBy == input.SharedWithUserID {
		return nil, ErrCannotShareWithSelf
	}

	// Check if already shared
	existing, err := s.repo.GetUserPermission(ctx, input.EntityType, input.EntityID, input.SharedWithUserID)
	if err == nil && existing != nil {
		return nil, ErrAlreadyShared
	}

	share := &models.DocumentShare{
		ID:               uuid.New(),
		TenantID:         input.TenantID,
		EntityType:       input.EntityType,
		EntityID:         input.EntityID,
		SharedWithUserID: input.SharedWithUserID,
		Permission:       input.Permission,
		SharedBy:         input.SharedBy,
		CreatedAt:        time.Now(),
	}

	if err := s.repo.Create(ctx, share); err != nil {
		return nil, err
	}

	slog.Info("entity shared",
		"share_id", share.ID,
		"entity_type", share.EntityType,
		"entity_id", share.EntityID,
		"shared_with", share.SharedWithUserID,
		"permission", share.Permission,
		"shared_by", share.SharedBy,
	)

	return share, nil
}

// UnshareEntity removes a share.
func (s *Service) UnshareEntity(ctx context.Context, shareID uuid.UUID) error {
	if err := s.repo.Delete(ctx, shareID); err != nil {
		return err
	}

	slog.Info("entity unshared",
		"share_id", shareID,
	)
	return nil
}

// ListByEntity returns all shares for a given entity (file or folder).
func (s *Service) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.DocumentShare, error) {
	return s.repo.ListByEntity(ctx, entityType, entityID)
}

// ListSharedWithMe returns all entities shared with a specific user.
func (s *Service) ListSharedWithMe(ctx context.Context, userID uuid.UUID, entityType string) ([]*models.DocumentShare, error) {
	return s.repo.ListSharedWithUser(ctx, userID, entityType)
}

// CheckAccess checks if a user has access to an entity and returns the permission level.
func (s *Service) CheckAccess(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID) (bool, string, error) {
	return s.repo.HasAccess(ctx, entityType, entityID, userID)
}
