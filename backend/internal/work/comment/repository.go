package comment

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TaskCommentWithAuthor extends TaskComment with author display name and quoted content preview
type TaskCommentWithAuthor struct {
	models.TaskComment
	QuotedAuthorName *string `json:"quoted_author_name,omitempty"`
}

// Repository defines the interface for comment persistence
type Repository interface {
	Create(ctx context.Context, comment *models.TaskComment) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.TaskComment, error)
	List(ctx context.Context, taskID uuid.UUID, page, pageSize int) ([]TaskCommentWithAuthor, int, error)
	Update(ctx context.Context, id uuid.UUID, content string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
