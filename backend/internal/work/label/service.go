package label

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

var colorHexRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Validation errors
var (
	ErrNameRequired = errors.New("label name is required")
	ErrColorInvalid = errors.New("label color must be a valid hex color (e.g. #6b7280)")
)

// Service handles label business logic.
type Service struct {
	repo Repository
}

// NewService creates a new label service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create validates and creates a new label.
func (s *Service) Create(ctx context.Context, tenantID, name, color string) (*Label, error) {
	if err := validateLabel(name, color); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, tenantID, name, color)
}

// GetByID returns a label by ID scoped to the tenant.
func (s *Service) GetByID(ctx context.Context, tenantID, id string) (*Label, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List returns all labels for the tenant.
func (s *Service) List(ctx context.Context, tenantID string) ([]*Label, error) {
	return s.repo.List(ctx, tenantID)
}

// Update validates and updates an existing label.
func (s *Service) Update(ctx context.Context, tenantID, id, name, color string) (*Label, error) {
	if err := validateLabel(name, color); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, tenantID, id, name, color)
}

// Delete removes a label.
func (s *Service) Delete(ctx context.Context, tenantID, id string) error {
	return s.repo.Delete(ctx, tenantID, id)
}

// GetLabelsByTaskIDs batch-loads labels for a set of task IDs.
func (s *Service) GetLabelsByTaskIDs(ctx context.Context, tenantID string, taskIDs []string) (map[string][]*Label, error) {
	return s.repo.GetLabelsByTaskIDs(ctx, tenantID, taskIDs)
}

// SetTaskLabels replaces all label assignments for a task.
func (s *Service) SetTaskLabels(ctx context.Context, tenantID, taskID string, labelIDs []string) error {
	for _, id := range labelIDs {
		if strings.TrimSpace(id) == "" {
			return ErrNotFound
		}
	}
	return s.repo.SetTaskLabels(ctx, tenantID, taskID, labelIDs)
}

// GetTaskLabelIDs returns the label IDs assigned to a task.
func (s *Service) GetTaskLabelIDs(ctx context.Context, tenantID, taskID string) ([]string, error) {
	return s.repo.GetTaskLabelIDs(ctx, tenantID, taskID)
}

func validateLabel(name, color string) error {
	if strings.TrimSpace(name) == "" {
		return ErrNameRequired
	}
	if color != "" && !colorHexRe.MatchString(color) {
		return ErrColorInvalid
	}
	return nil
}
