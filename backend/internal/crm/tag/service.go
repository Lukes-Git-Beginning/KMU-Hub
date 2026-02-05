package tag

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles tag business logic
type Service struct {
	repo Repository
}

// NewService creates a new tag service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a tag
type CreateInput struct {
	Name       string
	Color      string
	EntityType models.EntityType
}

// Create creates a new tag
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.Tag, error) {
	// Validate entity type
	if !input.EntityType.IsValid() {
		return nil, ErrInvalidEntityType
	}

	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrTagNameRequired
	}

	// Validate color
	color := input.Color
	if color == "" {
		color = "#6b7280" // Default gray
	}
	if !models.IsValidHexColor(color) {
		return nil, ErrInvalidColor
	}

	// Check for duplicate
	existing, _ := s.repo.GetByEntityAndName(ctx, input.EntityType, name)
	if existing != nil {
		return nil, ErrTagExists
	}

	tag := &models.Tag{
		ID:         uuid.New(),
		Name:       name,
		Color:      color,
		EntityType: input.EntityType,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, tag); err != nil {
		return nil, err
	}

	slog.Info("tag created",
		"tag_id", tag.ID,
		"entity_type", tag.EntityType,
		"name", tag.Name,
	)

	return tag, nil
}

// GetByID retrieves a tag by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.Tag, error) {
	tag, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTagNotFound
	}
	return tag, nil
}

// ListInput contains filtering options for listing tags
type ListInput struct {
	EntityType *models.EntityType
	Page       int
	PageSize   int
}

// List retrieves tags with optional filtering
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.Tag, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 50
	}

	offset := (input.Page - 1) * input.PageSize
	return s.repo.List(ctx, input.EntityType, offset, input.PageSize)
}

// UpdateInput contains the data that can be updated on a tag
type UpdateInput struct {
	Name  *string
	Color *string
}

// Update updates an existing tag
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*models.Tag, error) {
	tag, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTagNotFound
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrTagNameRequired
		}

		// Check for duplicate if name is changing
		if name != tag.Name {
			existing, _ := s.repo.GetByEntityAndName(ctx, tag.EntityType, name)
			if existing != nil {
				return nil, ErrTagExists
			}
		}
		tag.Name = name
	}

	if input.Color != nil {
		if !models.IsValidHexColor(*input.Color) {
			return nil, ErrInvalidColor
		}
		tag.Color = *input.Color
	}

	if updateErr := s.repo.Update(ctx, tag); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("tag updated", "tag_id", tag.ID)

	return tag, nil
}

// Delete removes a tag
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrTagNotFound
	}

	// Check if tag is in use
	inUse, err := s.repo.IsInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return ErrTagInUse
	}

	if deleteErr := s.repo.Delete(ctx, id); deleteErr != nil {
		return deleteErr
	}

	slog.Info("tag deleted",
		"tag_id", tag.ID,
		"entity_type", tag.EntityType,
		"name", tag.Name,
	)

	return nil
}
