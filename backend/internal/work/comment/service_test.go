package comment

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/task"
)

// --- Mock Comment Repository ---

type mockCommentRepo struct {
	comments map[uuid.UUID]*models.TaskComment
}

func newMockCommentRepo() *mockCommentRepo {
	return &mockCommentRepo{
		comments: make(map[uuid.UUID]*models.TaskComment),
	}
}

func (m *mockCommentRepo) Create(_ context.Context, comment *models.TaskComment) error {
	m.comments[comment.ID] = comment
	return nil
}

func (m *mockCommentRepo) GetByID(_ context.Context, id uuid.UUID) (*models.TaskComment, error) {
	c, ok := m.comments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (m *mockCommentRepo) List(_ context.Context, taskID uuid.UUID, _, _ int) ([]TaskCommentWithAuthor, int, error) {
	var result []TaskCommentWithAuthor
	for _, c := range m.comments {
		if c.TaskID == taskID {
			result = append(result, TaskCommentWithAuthor{TaskComment: *c})
		}
	}
	return result, len(result), nil
}

func (m *mockCommentRepo) Update(_ context.Context, id uuid.UUID, content string) error {
	c, ok := m.comments[id]
	if !ok {
		return ErrNotFound
	}
	c.Content = content
	c.UpdatedAt = time.Now()
	return nil
}

func (m *mockCommentRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.comments, id)
	return nil
}

// --- Mock Task Repository (minimal, for activity logging) ---

type mockTaskRepo struct {
	tasks      map[uuid.UUID]*models.TaskWithRelations
	activities []models.TaskActivity
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{
		tasks: make(map[uuid.UUID]*models.TaskWithRelations),
	}
}

func (m *mockTaskRepo) addTask(tw *models.TaskWithRelations) {
	m.tasks[tw.ID] = tw
}

func (m *mockTaskRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*models.TaskWithRelations, error) {
	tw, ok := m.tasks[id]
	if !ok {
		return nil, task.ErrNotFound
	}
	return tw, nil
}

func (m *mockTaskRepo) CreateActivity(_ context.Context, activity *models.TaskActivity) error {
	m.activities = append(m.activities, *activity)
	return nil
}

// Unused methods - satisfy task.Repository interface
func (m *mockTaskRepo) Create(_ context.Context, _ *models.Task) error { return nil }
func (m *mockTaskRepo) List(_ context.Context, _ uuid.UUID, _ task.TaskFilters) ([]models.TaskWithRelations, int, error) {
	return nil, 0, nil
}
func (m *mockTaskRepo) Update(_ context.Context, _ *models.Task) error          { return nil }
func (m *mockTaskRepo) Delete(_ context.Context, _, _ uuid.UUID) error          { return nil }
func (m *mockTaskRepo) MoveTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ float64) error {
	return nil
}
func (m *mockTaskRepo) GetNextTaskNumber(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *mockTaskRepo) GetSubtasks(_ context.Context, _ uuid.UUID, _ int) ([]models.TaskWithRelations, error) {
	return nil, nil
}
func (m *mockTaskRepo) GetParentChain(_ context.Context, _ uuid.UUID) ([]models.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) GetDepth(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *mockTaskRepo) CreateDependency(_ context.Context, _ *models.TaskDependency) error {
	return nil
}
func (m *mockTaskRepo) DeleteDependency(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockTaskRepo) ListDependencies(_ context.Context, _ uuid.UUID) ([]models.TaskDependency, error) {
	return nil, nil
}
func (m *mockTaskRepo) HasCycle(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockTaskRepo) ListActivities(_ context.Context, _ uuid.UUID, _, _ int) ([]models.TaskActivity, int, error) {
	return nil, 0, nil
}
func (m *mockTaskRepo) LinkEntity(_ context.Context, _ *models.TaskEntityLink) error { return nil }
func (m *mockTaskRepo) UnlinkEntity(_ context.Context, _ uuid.UUID) error            { return nil }
func (m *mockTaskRepo) ListEntityLinks(_ context.Context, _ uuid.UUID) ([]models.TaskEntityLink, error) {
	return nil, nil
}
func (m *mockTaskRepo) ListTasksForEntity(_ context.Context, _ string, _ uuid.UUID) ([]models.TaskWithRelations, error) {
	return nil, nil
}
func (m *mockTaskRepo) AttachFile(_ context.Context, _ *models.TaskFile) error { return nil }
func (m *mockTaskRepo) RemoveFile(_ context.Context, _ uuid.UUID) error        { return nil }
func (m *mockTaskRepo) ListFiles(_ context.Context, _ uuid.UUID) ([]models.TaskFile, error) {
	return nil, nil
}
func (m *mockTaskRepo) SetCustomFieldValues(_ context.Context, _ uuid.UUID, _ map[uuid.UUID]any) error {
	return nil
}
func (m *mockTaskRepo) GetCustomFieldValues(_ context.Context, _ uuid.UUID) (map[string]any, error) {
	return nil, nil
}
func (m *mockTaskRepo) Search(_ context.Context, _ uuid.UUID, _ string, _ task.TaskSearchFilters) ([]models.TaskWithRelations, int, error) {
	return nil, 0, nil
}
func (m *mockTaskRepo) ListByProject(_ context.Context, _ uuid.UUID) ([]models.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) ListDependenciesByProject(_ context.Context, _ uuid.UUID) ([]models.TaskDependency, error) {
	return nil, nil
}
func (m *mockTaskRepo) GetStatusByID(_ context.Context, _ uuid.UUID) (string, bool, error) {
	return "", false, nil
}

// --- Mock Event Emitter ---

type mockEventEmitter struct {
	events []models.EventPayload
}

func (m *mockEventEmitter) EmitTaskEvent(_ context.Context, payload models.EventPayload) error {
	m.events = append(m.events, payload)
	return nil
}

// --- Tests ---

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	commentRepo := newMockCommentRepo()
	taskRepo := newMockTaskRepo()
	svc := NewService(commentRepo, taskRepo)
	emitter := &mockEventEmitter{}
	svc.SetEventEmitter(emitter)

	taskID := uuid.New()
	authorID := uuid.New()
	assigneeID := uuid.New()

	taskRepo.addTask(&models.TaskWithRelations{
		Task: models.Task{
			ID:         taskID,
			Title:      "Test Task",
			AssigneeID: &assigneeID,
			CreatedBy:  authorID,
		},
	})

	t.Run("create comment with valid content", func(t *testing.T) {
		result, err := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  "This is a comment",
			AuthorID: authorID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Content != "This is a comment" {
			t.Errorf("expected content 'This is a comment', got %q", result.Content)
		}
		if result.TaskID != taskID {
			t.Errorf("expected task_id %s, got %s", taskID, result.TaskID)
		}

		// Verify activity was logged via taskRepo.CreateActivity
		if len(taskRepo.activities) != 1 {
			t.Fatalf("expected 1 activity, got %d", len(taskRepo.activities))
		}
		if taskRepo.activities[0].Action != models.TaskActionCommented {
			t.Errorf("expected action 'commented', got %q", taskRepo.activities[0].Action)
		}
		if taskRepo.activities[0].TaskID != taskID {
			t.Errorf("expected activity task_id %s, got %s", taskID, taskRepo.activities[0].TaskID)
		}
		// Verify NewValue contains comment preview
		if taskRepo.activities[0].NewValue == nil {
			t.Error("expected NewValue to contain comment preview")
		} else {
			var preview string
			_ = json.Unmarshal(*taskRepo.activities[0].NewValue, &preview)
			if preview != "This is a comment" {
				t.Errorf("expected activity preview 'This is a comment', got %q", preview)
			}
		}
	})

	t.Run("create comment emits event to assignee", func(t *testing.T) {
		emitter.events = nil
		_, err := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  "Another comment",
			AuthorID: authorID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// Should emit because assignee != author
		if len(emitter.events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(emitter.events))
		}
		if emitter.events[0].Type != "work.task.commented" {
			t.Errorf("expected event type work.task.commented, got %q", emitter.events[0].Type)
		}
	})

	t.Run("create comment does not emit when author is assignee", func(t *testing.T) {
		emitter.events = nil
		_, err := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  "Self comment",
			AuthorID: assigneeID, // author IS the assignee
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(emitter.events) != 0 {
			t.Errorf("expected 0 events when author is assignee, got %d", len(emitter.events))
		}
	})
}

func TestService_CreateWithQuoteReply(t *testing.T) {
	ctx := context.Background()
	commentRepo := newMockCommentRepo()
	taskRepo := newMockTaskRepo()
	svc := NewService(commentRepo, taskRepo)

	taskID := uuid.New()
	authorID := uuid.New()

	taskRepo.addTask(&models.TaskWithRelations{
		Task: models.Task{
			ID:        taskID,
			Title:     "Test Task",
			CreatedBy: authorID,
		},
	})

	// Create original comment
	original, err := svc.Create(ctx, CreateInput{
		TaskID:   taskID,
		Content:  "Original comment",
		AuthorID: authorID,
	})
	if err != nil {
		t.Fatalf("create original: %v", err)
	}

	t.Run("valid quote reply", func(t *testing.T) {
		reply, replyErr := svc.Create(ctx, CreateInput{
			TaskID:          taskID,
			Content:         "Reply to original",
			QuotedCommentID: &original.ID,
			AuthorID:        authorID,
		})
		if replyErr != nil {
			t.Fatalf("expected no error, got %v", replyErr)
		}
		if reply.QuotedCommentID == nil {
			t.Error("expected quoted_comment_id to be set")
		} else if *reply.QuotedCommentID != original.ID {
			t.Errorf("expected quoted_comment_id %s, got %s", original.ID, *reply.QuotedCommentID)
		}
	})

	t.Run("invalid quoted_comment_id", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, replyErr := svc.Create(ctx, CreateInput{
			TaskID:          taskID,
			Content:         "Reply to nothing",
			QuotedCommentID: &nonExistentID,
			AuthorID:        authorID,
		})
		if replyErr != ErrQuotedCommentNotFound {
			t.Errorf("expected ErrQuotedCommentNotFound, got %v", replyErr)
		}
	})

	t.Run("quoted comment from different task", func(t *testing.T) {
		otherTaskID := uuid.New()
		taskRepo.addTask(&models.TaskWithRelations{
			Task: models.Task{
				ID:        otherTaskID,
				Title:     "Other Task",
				CreatedBy: authorID,
			},
		})
		otherComment, _ := svc.Create(ctx, CreateInput{
			TaskID:   otherTaskID,
			Content:  "Comment on other task",
			AuthorID: authorID,
		})

		_, replyErr := svc.Create(ctx, CreateInput{
			TaskID:          taskID,
			Content:         "Cross-task reply",
			QuotedCommentID: &otherComment.ID,
			AuthorID:        authorID,
		})
		if replyErr != ErrQuotedCommentWrongTask {
			t.Errorf("expected ErrQuotedCommentWrongTask, got %v", replyErr)
		}
	})
}

func TestService_CreateWithMentions(t *testing.T) {
	ctx := context.Background()
	commentRepo := newMockCommentRepo()
	taskRepo := newMockTaskRepo()
	svc := NewService(commentRepo, taskRepo)
	emitter := &mockEventEmitter{}
	svc.SetEventEmitter(emitter)

	taskID := uuid.New()
	authorID := uuid.New()
	mentionedID := uuid.New()

	taskRepo.addTask(&models.TaskWithRelations{
		Task: models.Task{
			ID:        taskID,
			Title:     "Test Task",
			CreatedBy: authorID,
		},
	})

	t.Run("mention emits event", func(t *testing.T) {
		emitter.events = nil
		content := "Hey @{" + mentionedID.String() + "} please review"
		_, err := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  content,
			AuthorID: authorID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// Should emit mention event
		found := false
		for _, e := range emitter.events {
			if e.Type == "work.task.mentioned" {
				found = true
				if len(e.TargetUserIDs) != 1 || e.TargetUserIDs[0] != mentionedID.String() {
					t.Errorf("expected target %s, got %v", mentionedID, e.TargetUserIDs)
				}
			}
		}
		if !found {
			t.Error("expected work.task.mentioned event")
		}
	})

	t.Run("self-mention does not emit", func(t *testing.T) {
		emitter.events = nil
		content := "Note to @{" + authorID.String() + "} self"
		_, err := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  content,
			AuthorID: authorID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		for _, e := range emitter.events {
			if e.Type == "work.task.mentioned" {
				t.Error("should not emit mention event for self-mention")
			}
		}
	})
}

func TestService_CreateValidation(t *testing.T) {
	ctx := context.Background()
	commentRepo := newMockCommentRepo()
	taskRepo := newMockTaskRepo()
	svc := NewService(commentRepo, taskRepo)

	taskID := uuid.New()
	authorID := uuid.New()

	taskRepo.addTask(&models.TaskWithRelations{
		Task: models.Task{
			ID:        taskID,
			Title:     "Test Task",
			CreatedBy: authorID,
		},
	})

	t.Run("empty content rejected", func(t *testing.T) {
		_, err := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  "",
			AuthorID: authorID,
		})
		if err != ErrContentRequired {
			t.Errorf("expected ErrContentRequired, got %v", err)
		}
	})

	t.Run("whitespace-only content rejected", func(t *testing.T) {
		_, err := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  "   \t\n  ",
			AuthorID: authorID,
		})
		if err != ErrContentRequired {
			t.Errorf("expected ErrContentRequired, got %v", err)
		}
	})

	t.Run("content too long rejected", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 10001; i++ {
			longContent += "a"
		}
		_, err := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  longContent,
			AuthorID: authorID,
		})
		if err != ErrContentTooLong {
			t.Errorf("expected ErrContentTooLong, got %v", err)
		}
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()
	commentRepo := newMockCommentRepo()
	taskRepo := newMockTaskRepo()
	svc := NewService(commentRepo, taskRepo)

	taskID := uuid.New()
	authorID := uuid.New()
	otherUserID := uuid.New()

	taskRepo.addTask(&models.TaskWithRelations{
		Task: models.Task{
			ID:        taskID,
			Title:     "Test Task",
			CreatedBy: authorID,
		},
	})

	comment, _ := svc.Create(ctx, CreateInput{
		TaskID:   taskID,
		Content:  "Original content",
		AuthorID: authorID,
	})

	t.Run("update own comment", func(t *testing.T) {
		err := svc.Update(ctx, comment.ID, "Updated content", authorID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		updated, _ := commentRepo.GetByID(ctx, comment.ID)
		if updated.Content != "Updated content" {
			t.Errorf("expected content 'Updated content', got %q", updated.Content)
		}
	})

	t.Run("cannot update others comment", func(t *testing.T) {
		err := svc.Update(ctx, comment.ID, "Hijacked", otherUserID)
		if err != ErrCannotEditOthersComment {
			t.Errorf("expected ErrCannotEditOthersComment, got %v", err)
		}
	})

	t.Run("empty update content rejected", func(t *testing.T) {
		err := svc.Update(ctx, comment.ID, "", authorID)
		if err != ErrContentRequired {
			t.Errorf("expected ErrContentRequired, got %v", err)
		}
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	commentRepo := newMockCommentRepo()
	taskRepo := newMockTaskRepo()
	svc := NewService(commentRepo, taskRepo)

	taskID := uuid.New()
	authorID := uuid.New()
	otherUserID := uuid.New()

	taskRepo.addTask(&models.TaskWithRelations{
		Task: models.Task{
			ID:        taskID,
			Title:     "Test Task",
			CreatedBy: authorID,
		},
	})

	t.Run("delete own comment", func(t *testing.T) {
		comment, _ := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  "To be deleted",
			AuthorID: authorID,
		})
		err := svc.Delete(ctx, comment.ID, authorID, false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		_, getErr := commentRepo.GetByID(ctx, comment.ID)
		if getErr != ErrNotFound {
			t.Error("expected comment to be deleted")
		}
	})

	t.Run("cannot delete others comment", func(t *testing.T) {
		comment, _ := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  "Protected",
			AuthorID: authorID,
		})
		err := svc.Delete(ctx, comment.ID, otherUserID, false)
		if err != ErrCannotDeleteOthersComment {
			t.Errorf("expected ErrCannotDeleteOthersComment, got %v", err)
		}
	})

	t.Run("admin can delete others comment", func(t *testing.T) {
		comment, _ := svc.Create(ctx, CreateInput{
			TaskID:   taskID,
			Content:  "Admin deletable",
			AuthorID: authorID,
		})
		err := svc.Delete(ctx, comment.ID, otherUserID, true) // isAdmin = true
		if err != nil {
			t.Fatalf("expected admin to delete, got %v", err)
		}
	})
}

func TestParseMentions(t *testing.T) {
	t.Run("single mention", func(t *testing.T) {
		id := uuid.New()
		content := "Hey @{" + id.String() + "} look"
		mentions := parseMentions(content)
		if len(mentions) != 1 {
			t.Fatalf("expected 1 mention, got %d", len(mentions))
		}
		if mentions[0] != id {
			t.Errorf("expected %s, got %s", id, mentions[0])
		}
	})

	t.Run("multiple mentions", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()
		content := "@{" + id1.String() + "} and @{" + id2.String() + "}"
		mentions := parseMentions(content)
		if len(mentions) != 2 {
			t.Fatalf("expected 2 mentions, got %d", len(mentions))
		}
	})

	t.Run("duplicate mentions deduplicated", func(t *testing.T) {
		id := uuid.New()
		content := "@{" + id.String() + "} again @{" + id.String() + "}"
		mentions := parseMentions(content)
		if len(mentions) != 1 {
			t.Errorf("expected 1 unique mention, got %d", len(mentions))
		}
	})

	t.Run("no mentions", func(t *testing.T) {
		mentions := parseMentions("no mentions here")
		if len(mentions) != 0 {
			t.Errorf("expected 0 mentions, got %d", len(mentions))
		}
	})

	t.Run("invalid uuid in mention ignored", func(t *testing.T) {
		content := "@{not-a-valid-uuid-at-all-here} test"
		mentions := parseMentions(content)
		if len(mentions) != 0 {
			t.Errorf("expected 0 mentions for invalid uuid, got %d", len(mentions))
		}
	})
}
