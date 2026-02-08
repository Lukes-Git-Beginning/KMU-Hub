package status

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles project status business logic
type Service struct {
	repo Repository
}

// NewService creates a new status service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a project status
type CreateInput struct {
	ProjectID uuid.UUID
	Name      string
	Color     *string
	IsDefault bool
	IsClosed  bool
}

// Create creates a new project status
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.ProjectStatus, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrNameRequired
	}
	if len(name) > 100 {
		return nil, ErrNameTooLong
	}

	// Check duplicate name in project (case-insensitive)
	exists, err := s.repo.NameExistsInProject(ctx, input.ProjectID, name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateName
	}

	// Get next sort order
	sortOrder, err := s.repo.GetNextSortOrder(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	status := &models.ProjectStatus{
		ID:        uuid.New(),
		ProjectID: input.ProjectID,
		Name:      name,
		Color:     input.Color,
		SortOrder: sortOrder,
		IsDefault: input.IsDefault,
		IsClosed:  input.IsClosed,
		CreatedAt: time.Now(),
	}

	if createErr := s.repo.Create(ctx, status); createErr != nil {
		return nil, createErr
	}

	slog.Info("project status created",
		"status_id", status.ID,
		"project_id", status.ProjectID,
		"name", status.Name,
	)

	return status, nil
}

// GetByID retrieves a status by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.ProjectStatus, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByProject retrieves all statuses for a project
func (s *Service) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.ProjectStatus, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// UpdateInput contains the data that can be updated on a project status
type UpdateInput struct {
	Name      *string
	Color     *string
	IsDefault *bool
	IsClosed  *bool
}

// Update updates an existing project status
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*models.ProjectStatus, error) {
	status, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrNameRequired
		}
		if len(name) > 100 {
			return nil, ErrNameTooLong
		}

		// Check duplicate name (excluding self)
		exists, nameErr := s.repo.NameExistsInProject(ctx, status.ProjectID, name, &id)
		if nameErr != nil {
			return nil, nameErr
		}
		if exists {
			return nil, ErrDuplicateName
		}

		status.Name = name
	}

	if input.Color != nil {
		status.Color = input.Color
	}

	if input.IsDefault != nil {
		status.IsDefault = *input.IsDefault
	}

	if input.IsClosed != nil {
		status.IsClosed = *input.IsClosed
	}

	if updateErr := s.repo.Update(ctx, status); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("project status updated",
		"status_id", status.ID,
		"name", status.Name,
	)

	return status, nil
}

// Delete removes a project status.
// Cannot delete the default status. Tasks with this status get status_id set to NULL (per migration).
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	status, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Cannot delete default status
	if status.IsDefault {
		return ErrCannotDeleteDefault
	}

	if deleteErr := s.repo.Delete(ctx, id); deleteErr != nil {
		return deleteErr
	}

	slog.Info("project status deleted",
		"status_id", status.ID,
		"name", status.Name,
	)

	return nil
}

// Reorder updates the sort order of statuses within a project.
// All provided IDs must belong to the same project.
func (s *Service) Reorder(ctx context.Context, projectID uuid.UUID, statusIDs []uuid.UUID) error {
	// Validate that all IDs belong to the specified project
	for _, id := range statusIDs {
		status, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if status.ProjectID != projectID {
			return ErrInvalidSortOrder
		}
	}

	// Check for duplicates
	seen := make(map[uuid.UUID]bool)
	for _, id := range statusIDs {
		if seen[id] {
			return ErrInvalidSortOrder
		}
		seen[id] = true
	}

	if reorderErr := s.repo.Reorder(ctx, projectID, statusIDs); reorderErr != nil {
		return reorderErr
	}

	slog.Info("project statuses reordered",
		"project_id", projectID,
		"count", len(statusIDs),
	)

	return nil
}

// CopyForProject copies all statuses from one project to another (for template instantiation)
func (s *Service) CopyForProject(ctx context.Context, sourceProjectID, targetProjectID uuid.UUID) error {
	if copyErr := s.repo.CopyStatusesForProject(ctx, sourceProjectID, targetProjectID); copyErr != nil {
		return copyErr
	}

	slog.Info("project statuses copied",
		"source_project_id", sourceProjectID,
		"target_project_id", targetProjectID,
	)

	return nil
}
