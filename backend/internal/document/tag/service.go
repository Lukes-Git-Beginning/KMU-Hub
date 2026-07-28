package tag

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/models"
)

// hexColorRegex validates #RRGGBB hex color format.
var hexColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Service handles document tag business logic.
type Service struct {
	repo Repository
}

// NewService creates a new tag Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateTag creates a new document tag.
// Validates name (1-100 chars) and optional color format (#RRGGBB or empty).
func (s *Service) CreateTag(ctx context.Context, name, color string, userID uuid.UUID, tenantID uuid.UUID) (*models.DocumentTag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrTagNameRequired
	}
	if len(name) > 100 {
		return nil, ErrTagNameTooLong
	}

	// Validate color format if provided
	var colorPtr *string
	color = strings.TrimSpace(color)
	if color != "" {
		if !hexColorRegex.MatchString(color) {
			return nil, ErrInvalidColor
		}
		colorPtr = &color
	}

	now := time.Now()
	tag := &models.DocumentTag{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		Color:     colorPtr,
		CreatedBy: userID,
		CreatedAt: now,
	}

	if err := s.repo.Create(ctx, tag); err != nil {
		return nil, err
	}

	slog.Info("tag created",
		"tag_id", tag.ID,
		"name", tag.Name,
		"user_id", userID,
	)

	return tag, nil
}

// ListTags returns all tags for the tenant ordered by name.
func (s *Service) ListTags(ctx context.Context, tenantID uuid.UUID) ([]*models.DocumentTag, error) {
	return s.repo.List(ctx, tenantID)
}

// DeleteTag removes a tag scoped to the tenant. Cascade deletes file-tag
// associations via FK.
func (s *Service) DeleteTag(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return err
	}

	slog.Info("tag deleted",
		"tag_id", id,
	)

	return nil
}

// TagFile creates a file-tag association. Idempotent -- no error if already tagged.
func (s *Service) TagFile(ctx context.Context, tenantID, fileID, tagID uuid.UUID) error {
	return s.repo.TagFile(ctx, tenantID, fileID, tagID)
}

// UntagFile removes a file-tag association.
func (s *Service) UntagFile(ctx context.Context, tenantID, fileID, tagID uuid.UUID) error {
	return s.repo.UntagFile(ctx, tenantID, fileID, tagID)
}
