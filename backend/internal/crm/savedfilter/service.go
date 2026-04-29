package savedfilter

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles saved filter business logic
type Service struct {
	repo Repository
}

// NewService creates a new saved filter service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a saved filter
type CreateInput struct {
	TenantID   uuid.UUID
	Name       string
	EntityType string
	FilterJSON string
	IsDefault  bool
	CreatedBy  uuid.UUID
}

// Create creates a new saved filter
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.SavedFilter, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrNameRequired
	}

	// Validate entity type
	entityType := models.EntityType(input.EntityType)
	if !entityType.IsValid() {
		return nil, ErrInvalidEntityType
	}

	// Validate filter_json is valid JSON
	if !json.Valid([]byte(input.FilterJSON)) {
		return nil, ErrInvalidFilterJSON
	}

	// If setting as default, clear any existing default first
	if input.IsDefault {
		if err := s.repo.ClearDefault(ctx, input.TenantID, input.CreatedBy, entityType); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	filter := &models.SavedFilter{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		Name:       name,
		EntityType: entityType,
		FilterJSON: input.FilterJSON,
		IsDefault:  input.IsDefault,
		CreatedBy:  input.CreatedBy,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.Create(ctx, filter); err != nil {
		return nil, err
	}

	slog.Info("saved filter created",
		"filter_id", filter.ID,
		"name", filter.Name,
		"entity_type", filter.EntityType,
	)

	return filter, nil
}

// GetByID retrieves a saved filter by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.SavedFilter, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// ListInput contains filtering options for listing saved filters
type ListInput struct {
	TenantID   uuid.UUID
	EntityType *string
	UserID     *uuid.UUID
}

// List retrieves saved filters with optional filtering
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.SavedFilter, error) {
	filter := ListFilter{
		UserID: input.UserID,
	}

	if input.EntityType != nil {
		et := models.EntityType(*input.EntityType)
		if !et.IsValid() {
			return nil, ErrInvalidEntityType
		}
		filter.EntityType = &et
	}

	return s.repo.List(ctx, input.TenantID, filter)
}

// UpdateInput contains the data that can be updated on a saved filter
type UpdateInput struct {
	Name       *string
	FilterJSON *string
	IsDefault  *bool
}

// Update updates an existing saved filter
func (s *Service) Update(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, userID uuid.UUID, input UpdateInput) (*models.SavedFilter, error) {
	filter, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if filter.CreatedBy != userID {
		return nil, ErrFilterNotOwned
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrNameRequired
		}
		filter.Name = name
	}

	if input.FilterJSON != nil {
		if !json.Valid([]byte(*input.FilterJSON)) {
			return nil, ErrInvalidFilterJSON
		}
		filter.FilterJSON = *input.FilterJSON
	}

	if input.IsDefault != nil {
		// If setting as default, clear any existing default first
		if *input.IsDefault && !filter.IsDefault {
			if clearErr := s.repo.ClearDefault(ctx, tenantID, filter.CreatedBy, filter.EntityType); clearErr != nil {
				return nil, clearErr
			}
		}
		filter.IsDefault = *input.IsDefault
	}

	filter.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, filter); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("saved filter updated", "filter_id", filter.ID)

	return filter, nil
}

// Delete removes a saved filter
func (s *Service) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, userID uuid.UUID) error {
	filter, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	// Check ownership
	if filter.CreatedBy != userID {
		return ErrFilterNotOwned
	}

	if deleteErr := s.repo.Delete(ctx, id, tenantID); deleteErr != nil {
		return deleteErr
	}

	slog.Info("saved filter deleted",
		"filter_id", filter.ID,
		"name", filter.Name,
	)

	return nil
}
