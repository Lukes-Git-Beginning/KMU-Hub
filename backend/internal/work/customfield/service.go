package customfield

import (
	"context"
	"strings"
)

// Service handles custom field definition business logic.
type Service struct {
	repo Repository
}

// NewService creates a new custom field service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create validates and creates a new custom field definition.
func (s *Service) Create(ctx context.Context, tenantID, name, fieldType string, options []string, position int) (*Definition, error) {
	if err := s.validate(name, fieldType); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, tenantID, name, fieldType, options, position)
}

// GetByID returns a definition by ID.
func (s *Service) GetByID(ctx context.Context, tenantID, id string) (*Definition, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List returns all definitions for the tenant.
func (s *Service) List(ctx context.Context, tenantID string) ([]*Definition, error) {
	return s.repo.List(ctx, tenantID)
}

// Update validates and updates an existing definition.
func (s *Service) Update(ctx context.Context, tenantID, id, name, fieldType string, options []string, position int) (*Definition, error) {
	if err := s.validate(name, fieldType); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, tenantID, id, name, fieldType, options, position)
}

// Delete removes a custom field definition.
func (s *Service) Delete(ctx context.Context, tenantID, id string) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *Service) validate(name, fieldType string) error {
	if strings.TrimSpace(name) == "" {
		return ErrNameRequired
	}
	if _, ok := ValidFieldTypes[fieldType]; !ok {
		return ErrInvalidFieldType
	}
	return nil
}
