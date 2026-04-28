package comment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/work/task"
)

// mentionRegex matches @mentions in comment content.
// Supports both @username and @{uuid} patterns.
var mentionRegex = regexp.MustCompile(`@\{([0-9a-fA-F-]{36})\}`)

// Service handles comment business logic
type Service struct {
	repo         Repository
	taskRepo     task.Repository
	eventEmitter task.EventEmitter
}

// NewService creates a new comment service.
// taskRepo is required for activity logging via CreateActivity.
func NewService(repo Repository, taskRepo task.Repository) *Service {
	return &Service{
		repo:     repo,
		taskRepo: taskRepo,
	}
}

// SetEventEmitter sets the optional event emitter for notification events.
func (s *Service) SetEventEmitter(emitter task.EventEmitter) {
	s.eventEmitter = emitter
}

// CreateInput contains the data needed to create a comment
type CreateInput struct {
	TenantID        uuid.UUID
	TaskID          uuid.UUID
	Content         string
	QuotedCommentID *uuid.UUID
	AuthorID        uuid.UUID
}

// Create creates a new comment on a task
func (s *Service) Create(ctx context.Context, input CreateInput) (*TaskCommentWithAuthor, error) {
	// Validate content
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, ErrContentRequired
	}
	if len(content) > 10000 {
		return nil, ErrContentTooLong
	}

	// Verify task exists (tenant-scoped)
	taskWithRel, err := s.taskRepo.GetByID(ctx, input.TaskID, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("task lookup: %w", err)
	}

	// Validate quoted comment if provided
	if input.QuotedCommentID != nil {
		quoted, quotedErr := s.repo.GetByID(ctx, *input.QuotedCommentID)
		if quotedErr != nil {
			return nil, ErrQuotedCommentNotFound
		}
		if quoted.TaskID != input.TaskID {
			return nil, ErrQuotedCommentWrongTask
		}
	}

	now := time.Now()
	comment := &models.TaskComment{
		ID:              uuid.New(),
		TaskID:          input.TaskID,
		AuthorID:        input.AuthorID,
		Content:         content,
		QuotedCommentID: input.QuotedCommentID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if createErr := s.repo.Create(ctx, comment); createErr != nil {
		return nil, createErr
	}

	// Log task activity
	preview := content
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	newVal := jsonString(preview)
	_ = s.taskRepo.CreateActivity(ctx, &models.TaskActivity{
		ID:        uuid.New(),
		TaskID:    input.TaskID,
		ActorID:   input.AuthorID,
		Action:    models.TaskActionCommented,
		NewValue:  newVal,
		CreatedAt: now,
	})

	slog.Info("comment created",
		"comment_id", comment.ID,
		"task_id", comment.TaskID,
	)

	// Emit comment event to task assignee (if not the commenter)
	if s.eventEmitter != nil && taskWithRel.AssigneeID != nil && *taskWithRel.AssigneeID != input.AuthorID {
		deepLink := fmt.Sprintf("/work/tasks/%s", input.TaskID)
		if taskWithRel.ProjectID != nil {
			deepLink = fmt.Sprintf("/work/projects/%s/tasks/%s", taskWithRel.ProjectID, input.TaskID)
		}
		_ = s.eventEmitter.EmitTaskEvent(ctx, models.EventPayload{
			Type:          event.EventWorkTaskCommented,
			Priority:      models.PriorityNormal,
			ResourceID:    input.TaskID.String(),
			ModuleID:      event.ModuleWork,
			Title:         "New comment on: " + taskWithRel.Title,
			Body:          preview,
			DeepLink:      deepLink,
			GroupKey:      "task-comment:" + input.TaskID.String(),
			TargetUserIDs: []string{taskWithRel.AssigneeID.String()},
			Timestamp:     now,
		})
	}

	// Parse @mentions and emit mention events
	if s.eventEmitter != nil {
		mentions := parseMentions(content)
		for _, mentionedID := range mentions {
			if mentionedID == input.AuthorID {
				continue // Don't notify self-mention
			}
			deepLink := fmt.Sprintf("/work/tasks/%s", input.TaskID)
			if taskWithRel.ProjectID != nil {
				deepLink = fmt.Sprintf("/work/projects/%s/tasks/%s", taskWithRel.ProjectID, input.TaskID)
			}
			_ = s.eventEmitter.EmitTaskEvent(ctx, models.EventPayload{
				Type:          event.EventWorkTaskMentioned,
				Priority:      models.PriorityNormal,
				ResourceID:    input.TaskID.String(),
				ModuleID:      event.ModuleWork,
				Title:         "You were mentioned in: " + taskWithRel.Title,
				Body:          preview,
				DeepLink:      deepLink,
				TargetUserIDs: []string{mentionedID.String()},
				Timestamp:     now,
			})
		}
	}

	// Fetch the comment with author info
	result, _, listErr := s.repo.List(ctx, input.TaskID, 1, 1000)
	if listErr != nil {
		// Return a basic response if list fails
		return &TaskCommentWithAuthor{TaskComment: *comment}, nil
	}
	for _, c := range result {
		if c.ID == comment.ID {
			return &c, nil
		}
	}

	return &TaskCommentWithAuthor{TaskComment: *comment}, nil
}

// Update updates an existing comment. Only the author can update.
func (s *Service) Update(ctx context.Context, commentID uuid.UUID, content string, actorID uuid.UUID) error {
	comment, err := s.repo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	if comment.AuthorID != actorID {
		return ErrCannotEditOthersComment
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return ErrContentRequired
	}
	if len(content) > 10000 {
		return ErrContentTooLong
	}

	if updateErr := s.repo.Update(ctx, commentID, content); updateErr != nil {
		return updateErr
	}

	slog.Info("comment updated",
		"comment_id", commentID,
	)

	return nil
}

// Delete removes a comment. Only the author (or admin) can delete.
func (s *Service) Delete(ctx context.Context, commentID uuid.UUID, actorID uuid.UUID, isAdmin bool) error {
	comment, err := s.repo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	if comment.AuthorID != actorID && !isAdmin {
		return ErrCannotDeleteOthersComment
	}

	if deleteErr := s.repo.Delete(ctx, commentID); deleteErr != nil {
		return deleteErr
	}

	slog.Info("comment deleted",
		"comment_id", commentID,
	)

	return nil
}

// List retrieves comments for a task
func (s *Service) List(ctx context.Context, taskID uuid.UUID, page, pageSize int) ([]TaskCommentWithAuthor, int, error) {
	return s.repo.List(ctx, taskID, page, pageSize)
}

// parseMentions extracts mentioned user UUIDs from comment content
func parseMentions(content string) []uuid.UUID {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	var ids []uuid.UUID
	seen := make(map[uuid.UUID]bool)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		id, err := uuid.Parse(match[1])
		if err != nil {
			continue
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

// jsonString converts a string to a json.RawMessage pointer
func jsonString(s string) *json.RawMessage {
	data, _ := json.Marshal(s)
	raw := json.RawMessage(data)
	return &raw
}
