package customfield

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles custom field business logic
type Service struct {
	repo Repository
}

// NewService creates a new custom field service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a custom field
type CreateInput struct {
	TenantID     uuid.UUID
	EntityType   models.EntityType
	FieldName    string
	FieldLabel   string
	FieldType    models.FieldType
	Options      []string
	DefaultValue *string
	IsRequired   bool
	SortOrder    int
	CreatedBy    uuid.UUID
}

// Create creates a new custom field definition
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.CustomFieldDefinition, error) {
	// Validate entity type
	if !input.EntityType.IsValid() {
		return nil, ErrInvalidEntityType
	}

	// Validate field type
	if !input.FieldType.IsValid() {
		return nil, ErrInvalidFieldType
	}

	// Validate required fields
	if input.FieldName == "" {
		return nil, ErrFieldNameRequired
	}
	if input.FieldLabel == "" {
		return nil, ErrFieldLabelRequired
	}

	// Validate options for select/multiselect
	if input.FieldType.RequiresOptions() && len(input.Options) == 0 {
		return nil, ErrOptionsRequired
	}

	// Check for duplicate within tenant
	existing, _ := s.repo.GetByEntityAndName(ctx, input.TenantID, input.EntityType, input.FieldName)
	if existing != nil {
		return nil, ErrFieldExists
	}

	field := &models.CustomFieldDefinition{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		EntityType:   input.EntityType,
		FieldName:    input.FieldName,
		FieldLabel:   input.FieldLabel,
		FieldType:    input.FieldType,
		Options:      input.Options,
		DefaultValue: input.DefaultValue,
		IsRequired:   input.IsRequired,
		SortOrder:    input.SortOrder,
		CreatedBy:    input.CreatedBy,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, field); err != nil {
		return nil, err
	}

	slog.Info("custom field created",
		"field_id", field.ID,
		"entity_type", field.EntityType,
		"field_name", field.FieldName,
	)

	return field, nil
}

// GetByID retrieves a custom field by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.CustomFieldDefinition, error) {
	field, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, ErrFieldNotFound
	}
	return field, nil
}

// ListInput contains filtering options for listing custom fields
type ListInput struct {
	TenantID   uuid.UUID
	EntityType *models.EntityType
	Page       int
	PageSize   int
}

// List retrieves custom fields with optional filtering
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.CustomFieldDefinition, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := (input.Page - 1) * input.PageSize
	return s.repo.List(ctx, input.TenantID, input.EntityType, offset, input.PageSize)
}

// UpdateInput contains the data that can be updated on a custom field
type UpdateInput struct {
	TenantID     uuid.UUID
	FieldLabel   *string
	Options      []string
	DefaultValue *string
	IsRequired   *bool
	SortOrder    *int
}

// Update updates an existing custom field definition
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*models.CustomFieldDefinition, error) {
	field, err := s.repo.GetByID(ctx, id, input.TenantID)
	if err != nil {
		return nil, ErrFieldNotFound
	}

	if input.FieldLabel != nil {
		if *input.FieldLabel == "" {
			return nil, ErrFieldLabelRequired
		}
		field.FieldLabel = *input.FieldLabel
	}

	if input.Options != nil {
		// Validate options if field type requires them
		if field.FieldType.RequiresOptions() && len(input.Options) == 0 {
			return nil, ErrOptionsRequired
		}
		field.Options = input.Options
	}

	if input.DefaultValue != nil {
		field.DefaultValue = input.DefaultValue
	}

	if input.IsRequired != nil {
		field.IsRequired = *input.IsRequired
	}

	if input.SortOrder != nil {
		field.SortOrder = *input.SortOrder
	}

	field.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, field); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("custom field updated",
		"field_id", field.ID,
	)

	return field, nil
}

// Delete removes a custom field definition
func (s *Service) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	field, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return ErrFieldNotFound
	}

	if deleteErr := s.repo.Delete(ctx, id, tenantID); deleteErr != nil {
		return deleteErr
	}

	slog.Info("custom field deleted",
		"field_id", field.ID,
		"entity_type", field.EntityType,
		"field_name", field.FieldName,
	)

	return nil
}
