package virtual

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles virtual folder business logic.
// Virtual folders surface files from other subsystems (chat, email, task)
// as read-only views without file duplication.
type Service struct {
	repo Repository
}

// NewService creates a new virtual folder Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListVirtualFiles returns files from all or a specific source, scoped to the user.
// If sourceType is empty, all sources are returned. Valid sourceType values:
// "chat", "email", "task".
func (s *Service) ListVirtualFiles(ctx context.Context, userID uuid.UUID, sourceType string, limit, offset int) ([]*models.VirtualFile, int, error) {
	files, total, err := s.repo.ListAll(ctx, userID, sourceType, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	slog.Info("virtual files listed",
		"user_id", userID,
		"source_type", sourceType,
		"returned", len(files),
		"total", total,
	)

	return files, total, nil
}
