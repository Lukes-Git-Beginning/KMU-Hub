package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/work/project"
)

// Service handles task business logic
type Service struct {
	repo         Repository
	projectRepo  project.Repository
	eventEmitter EventEmitter
	// openProjects: a project member-less caller may work on the tasks of any
	// project of the tenant (see SetOpenProjects).
	openProjects bool
	// fileSigner presigns task file downloads; nil = no object store.
	fileSigner FileURLSigner
}

// FileURLSigner hands out a time-limited download URL for a stored object.
// chat/file.MinIOStore satisfies it.
type FileURLSigner interface {
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// fileURLExpiry matches the chat file download URLs.
const fileURLExpiry = 15 * time.Minute

// NewService creates a new task service
func NewService(repo Repository, projectRepo project.Repository) *Service {
	return &Service{
		repo:        repo,
		projectRepo: projectRepo,
	}
}

// SetEventEmitter sets the optional event emitter for notification events.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.eventEmitter = emitter
}

// SetFileSigner wires the object store task file downloads are presigned with.
func (s *Service) SetFileSigner(signer FileURLSigner) {
	s.fileSigner = signer
}

// AttachFile records a file the caller uploaded beforehand through
// POST /files/upload (without a channel). The storage keys are client-supplied,
// so they must be uploads of the caller's own tenant — otherwise a task could
// point at another tenant's object and hand out a download URL for it. Like
// every other task write it needs project edit rights on the task's project.
// lean: file_size is taken as sent, not checked against the object; the quota
// was charged with the real size at upload. Stat the object if the size ever
// feeds billing.
func (s *Service) AttachFile(ctx context.Context, tenantID, actorID uuid.UUID, f *models.TaskFile) error {
	if !chatfile.IsTenantUpload(tenantID, f.StorageKey) ||
		(f.ThumbnailKey != nil && !chatfile.IsTenantUpload(tenantID, *f.ThumbnailKey)) {
		return ErrForeignStorageKey
	}
	if err := s.requireTaskEditor(ctx, tenantID, f.TaskID, actorID); err != nil {
		return err
	}
	f.TenantID = tenantID
	return s.repo.AttachFile(ctx, f)
}

// RemoveFile deletes a task file record of the caller's tenant. Same edit-right
// rule as AttachFile; a file of another tenant is ErrFileNotFound.
// lean: the stored object and the quota are not released here; see the
// document cleanup path if task files ever need reclaiming.
//
// A non-nil ownerID is the own scope of work:task:edit (see RequireOwnTask).
func (s *Service) RemoveFile(ctx context.Context, tenantID, actorID, fileID uuid.UUID, ownerID *uuid.UUID) error {
	f, err := s.repo.GetFile(ctx, tenantID, fileID)
	if err != nil {
		return err
	}
	if err := s.RequireOwnTask(ctx, tenantID, f.TaskID, ownerID); err != nil {
		return err
	}
	if err := s.requireTaskEditor(ctx, tenantID, f.TaskID, actorID); err != nil {
		return err
	}
	return s.repo.RemoveFile(ctx, tenantID, fileID)
}

// ownsTask is the own scope of work:task:edit: the caller created the task or
// is its assignee (the owners the desktop's useTaskCan checks, so an assignee
// can still complete it).
func ownsTask(t *models.TaskWithRelations, ownerID uuid.UUID) bool {
	return t.CreatedBy == ownerID || (t.AssigneeID != nil && *t.AssigneeID == ownerID)
}

// RequireOwnTask enforces the own scope of work:task:edit on a task: a nil
// ownerID passes, otherwise the task must be one ownerID owns (ownsTask), any
// other answers ErrNotFound so a caller cannot probe foreign task ids.
func (s *Service) RequireOwnTask(ctx context.Context, tenantID, taskID uuid.UUID, ownerID *uuid.UUID) error {
	if ownerID == nil {
		return nil
	}
	existing, err := s.repo.GetByID(ctx, taskID, tenantID)
	if err != nil {
		return err
	}
	if !ownsTask(existing, *ownerID) {
		return ErrNotFound
	}
	return nil
}

// requireTaskEditor applies the rule Update uses: a task with a project needs
// project edit rights, a project-less task is open to every caller of the tenant.
func (s *Service) requireTaskEditor(ctx context.Context, tenantID, taskID, actorID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, taskID, tenantID)
	if err != nil {
		return err
	}
	if existing.ProjectID == nil {
		return nil
	}
	return s.requireProjectEditor(ctx, *existing.ProjectID, tenantID, actorID)
}

// GetFileDownloadURL presigns the download of a task file of the caller's
// tenant. Reading the file needs no more than reading the task's file list.
func (s *Service) GetFileDownloadURL(ctx context.Context, tenantID, fileID uuid.UUID) (string, error) {
	if s.fileSigner == nil {
		return "", ErrFileStoreUnavailable
	}
	f, err := s.repo.GetFile(ctx, tenantID, fileID)
	if err != nil {
		return "", err
	}
	return s.fileSigner.GetPresignedURL(ctx, f.StorageKey, fileURLExpiry)
}

// SetOpenProjects makes every project of the tenant editable for any caller who
// reached the service through a project permission: a missing membership then
// counts as editor. An explicit viewer membership still blocks writes. The
// server passes the same switch it hands the project service as isAdmin, so
// "I can see this project" and "I can work in it" never diverge.
func (s *Service) SetOpenProjects(open bool) {
	s.openProjects = open
}

// requireProjectEditor is the one membership check for every task write.
// Closed (default): the actor must be a member and not a viewer. Open: a
// non-member is an editor, but the project must exist in the tenant — the
// membership lookup used to be the only thing that kept a foreign project ID out.
func (s *Service) requireProjectEditor(ctx context.Context, projectID, tenantID, actorID uuid.UUID) error {
	member, err := s.projectRepo.GetMember(ctx, projectID, actorID)
	if err != nil {
		if !s.openProjects {
			return ErrNotProjectMember
		}
		if !errors.Is(err, project.ErrNotProjectMember) {
			return fmt.Errorf("project member: %w", err)
		}
		if _, getErr := s.projectRepo.GetByID(ctx, projectID, tenantID); getErr != nil {
			return fmt.Errorf("project: %w", getErr)
		}
		return nil
	}
	if member.Role == models.ProjectRoleViewer {
		return ErrViewerCannotEdit
	}
	return nil
}

// resolveActorName best-effort resolves a user's display name for event actor
// metadata. Returns "" on any error so emitting a notification event is never
// blocked by name resolution.
func (s *Service) resolveActorName(ctx context.Context, actorID uuid.UUID) string {
	name, err := s.repo.GetUserDisplayName(ctx, actorID)
	if err != nil {
		return ""
	}
	return name
}

// CreateInput contains the data needed to create a task
type CreateInput struct {
	TenantID     uuid.UUID
	ProjectID    *uuid.UUID
	Title        string
	Description  *string
	StatusID     *uuid.UUID
	Priority     string
	AssigneeID   *uuid.UUID
	ParentTaskID *uuid.UUID
	SortOrder    float64
	StartDate    *time.Time
	DueDate      *time.Time
	CreatedBy    uuid.UUID
}

// Create creates a new task
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.TaskWithRelations, error) {
	// Validate title
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrTitleRequired
	}
	if len(title) > 500 {
		return nil, ErrTitleTooLong
	}

	// Validate priority
	priority := input.Priority
	if priority == "" {
		priority = models.TaskPriorityNormal
	}
	if !models.ValidTaskPriorities[priority] {
		return nil, ErrInvalidPriority
	}

	if input.StartDate != nil && input.DueDate != nil && input.StartDate.After(*input.DueDate) {
		return nil, ErrInvalidDateRange
	}

	// Check project membership if project_id is set
	if input.ProjectID != nil {
		if err := s.requireProjectEditor(ctx, *input.ProjectID, input.TenantID, input.CreatedBy); err != nil {
			return nil, err
		}
	}

	// Handle nesting
	var depth int
	if input.ParentTaskID != nil {
		parentTask, err := s.repo.GetByID(ctx, *input.ParentTaskID, input.TenantID)
		if err != nil {
			return nil, fmt.Errorf("parent task: %w", err)
		}

		// Ensure parent is in the same project
		if input.ProjectID != nil && parentTask.ProjectID != nil {
			if *input.ProjectID != *parentTask.ProjectID {
				return nil, ErrParentInDiffProject
			}
		} else if input.ProjectID != nil || parentTask.ProjectID != nil {
			return nil, ErrParentInDiffProject
		}

		depth = parentTask.Depth + 1
		if depth > MaxNestingDepth {
			return nil, ErrMaxDepthExceeded
		}
	}

	// Get task number
	var taskNumber int
	if input.ProjectID != nil {
		num, err := s.repo.GetNextTaskNumber(ctx, input.TenantID, *input.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("task number: %w", err)
		}
		taskNumber = num
	} else {
		// Standalone tasks get 0 as task_number (no project counter)
		taskNumber = 0
	}

	now := time.Now()
	task := &models.Task{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		ProjectID:    input.ProjectID,
		TaskNumber:   taskNumber,
		Title:        title,
		Description:  input.Description,
		StatusID:     input.StatusID,
		Priority:     priority,
		AssigneeID:   input.AssigneeID,
		ParentTaskID: input.ParentTaskID,
		Depth:        depth,
		SortOrder:    input.SortOrder,
		StartDate:    input.StartDate,
		DueDate:      input.DueDate,
		CreatedBy:    input.CreatedBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	// Log activity
	_ = s.logActivity(ctx, task.ID, input.CreatedBy, models.TaskActionCreated, nil, nil, nil)

	slog.Info("task created",
		"task_id", task.ID,
		"title", task.Title,
		"project_id", task.ProjectID,
	)

	// Emit event if assignee differs from creator
	if s.eventEmitter != nil && input.AssigneeID != nil && *input.AssigneeID != input.CreatedBy {
		deepLink := ""
		if input.ProjectID != nil {
			deepLink = fmt.Sprintf("/work/projects/%s/tasks/%s", input.ProjectID, task.ID)
		} else {
			deepLink = fmt.Sprintf("/work/tasks/%s", task.ID)
		}
		_ = s.eventEmitter.EmitTaskEvent(ctx, models.EventPayload{
			Type:          event.EventWorkTaskCreated,
			Priority:      models.PriorityNormal,
			ActorID:       input.CreatedBy.String(),
			ActorName:     s.resolveActorName(ctx, input.CreatedBy),
			ResourceID:    task.ID.String(),
			ModuleID:      event.ModuleWork,
			Title:         "New task assigned: " + task.Title,
			Body:          "You have been assigned a new task",
			DeepLink:      deepLink,
			TargetUserIDs: []string{input.AssigneeID.String()},
			Timestamp:     now,
		})
	}

	return s.repo.GetByID(ctx, task.ID, input.TenantID)
}

// statusForClosedFlag resolves "complete"/"reopen" to a concrete status of the task's
// project. nil means the task already is in the requested state. The task must have a
// project; project-less tasks have no statuses and are completed via completed_at alone.
func (s *Service) statusForClosedFlag(ctx context.Context, tenantID uuid.UUID, existing *models.TaskWithRelations, closed bool) (*uuid.UUID, error) {
	if existing.StatusID != nil && existing.StatusIsClosed == closed {
		return nil, nil
	}
	id, err := s.repo.FirstStatusByClosed(ctx, tenantID, *existing.ProjectID, closed)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// UpdateInput contains the data that can be updated on a task
type UpdateInput struct {
	Title        *string
	Description  *string
	StatusID     *uuid.UUID
	Priority     *string
	AssigneeID   *uuid.UUID // uuid.Nil to clear
	StartDate    *time.Time
	DueDate      *time.Time
	ClearDueDate bool // removes the due date; wins over DueDate
	// IsClosed completes (true) or reopens (false) the task by moving it to the
	// project's first closed/open status; ignored when StatusID is set.
	IsClosed *bool
	// ProjectID moves a task that has no project (and no parent/subtasks) into the project.
	// The task gets the next task number and the first open status of that project.
	ProjectID *uuid.UUID
	SortOrder    *float64
	// OwnerID is the own scope of work:task:edit, not a field to change: a
	// non-nil value updates only a task OwnerID created or is assigned to (the
	// owners the desktop's useTaskCan checks, so an assignee can still
	// complete), any other answers ErrNotFound.
	OwnerID *uuid.UUID
}

// Update updates an existing task
func (s *Service) Update(ctx context.Context, tenantID, taskID uuid.UUID, input UpdateInput, actorID uuid.UUID) (*models.TaskWithRelations, error) {
	existing, err := s.repo.GetByID(ctx, taskID, tenantID)
	if err != nil {
		return nil, err
	}
	if input.OwnerID != nil && !ownsTask(existing, *input.OwnerID) {
		return nil, ErrNotFound
	}

	// Check project membership
	if existing.ProjectID != nil {
		if err := s.requireProjectEditor(ctx, *existing.ProjectID, tenantID, actorID); err != nil {
			return nil, err
		}
	}

	// lean: no race guard against two concurrent moves of the same task; the (project_id,
	// task_number) unique index keeps numbers consistent, the last move wins. Guard in
	// Repository.Update (WHERE project_id IS NOT DISTINCT FROM old) if that ever matters.
	moveTo := input.ProjectID
	if moveTo != nil && existing.ProjectID != nil && *existing.ProjectID == *moveTo {
		moveTo = nil // already there
	}
	if moveTo != nil {
		// lean: a task with a parent or subtasks stays put; moving a subtree needs
		// per-task numbers and is its own decision.
		if existing.ProjectID != nil || existing.ParentTaskID != nil || existing.SubtaskCount > 0 {
			return nil, ErrCannotMoveToProject
		}
		// Tenant-scoped lookup first: a foreign project id must read as not found.
		if _, getErr := s.projectRepo.GetByID(ctx, *moveTo, tenantID); getErr != nil {
			return nil, fmt.Errorf("project: %w", getErr)
		}
		if err := s.requireProjectEditor(ctx, *moveTo, tenantID, actorID); err != nil {
			return nil, err
		}
		openStatus, statusErr := s.repo.FirstStatusByClosed(ctx, tenantID, *moveTo, false)
		if statusErr != nil {
			return nil, statusErr
		}
		input.StatusID = &openStatus
	}

	// A project-less task has no statuses: completion is completed_at alone.
	completeWithoutStatus := input.IsClosed != nil && input.StatusID == nil && existing.ProjectID == nil
	if input.IsClosed != nil && input.StatusID == nil && !completeWithoutStatus {
		statusID, resolveErr := s.statusForClosedFlag(ctx, tenantID, existing, *input.IsClosed)
		if resolveErr != nil {
			return nil, resolveErr
		}
		input.StatusID = statusID
	}

	task := &existing.Task
	// GetByID does not scan tenant_id, but repo.Update filters on it — restore
	// it from the (already tenant-verified) lookup, or the UPDATE matches 0 rows.
	task.TenantID = tenantID

	if moveTo != nil {
		number, numErr := s.repo.GetNextTaskNumber(ctx, tenantID, *moveTo)
		if numErr != nil {
			return nil, numErr
		}
		task.ProjectID = moveTo
		task.TaskNumber = number
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return nil, ErrTitleRequired
		}
		if len(title) > 500 {
			return nil, ErrTitleTooLong
		}
		if title != task.Title {
			_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionTitleChanged, "title", task.Title, title)
			task.Title = title
		}
	}

	if input.Description != nil {
		oldDesc := ""
		if task.Description != nil {
			oldDesc = *task.Description
		}
		newDesc := *input.Description
		if oldDesc != newDesc {
			_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionDescriptionChanged, "description", oldDesc, newDesc)
			task.Description = input.Description
		}
	}

	var statusChanged bool
	var oldStatusName string
	if input.StatusID != nil {
		if task.StatusID == nil || *input.StatusID != *task.StatusID {
			statusChanged = true
			if task.StatusID != nil && existing.StatusName != nil {
				oldStatusName = *existing.StatusName
			}

			// Check if new status is closed
			newStatusName, isClosed, statusErr := s.repo.GetStatusByID(ctx, *input.StatusID)
			if statusErr != nil {
				return nil, statusErr
			}

			_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionStatusChanged, "status", oldStatusName, newStatusName)
			task.StatusID = input.StatusID

			if isClosed {
				now := time.Now()
				task.CompletedAt = &now
			} else {
				task.CompletedAt = nil
			}
		}
	}

	if completeWithoutStatus {
		switch {
		case *input.IsClosed && task.CompletedAt == nil:
			now := time.Now()
			task.CompletedAt = &now
		case !*input.IsClosed:
			task.CompletedAt = nil
		}
	}

	var assigneeChanged bool
	if input.AssigneeID != nil {
		if *input.AssigneeID == uuid.Nil {
			if task.AssigneeID != nil {
				_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionAssigned, "assignee", task.AssigneeID.String(), "")
				task.AssigneeID = nil
				assigneeChanged = true
			}
		} else if task.AssigneeID == nil || *input.AssigneeID != *task.AssigneeID {
			old := ""
			if task.AssigneeID != nil {
				old = task.AssigneeID.String()
			}
			_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionAssigned, "assignee", old, input.AssigneeID.String())
			task.AssigneeID = input.AssigneeID
			assigneeChanged = true
		}
	}

	if input.Priority != nil && *input.Priority != task.Priority {
		if !models.ValidTaskPriorities[*input.Priority] {
			return nil, ErrInvalidPriority
		}
		_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionPriorityChanged, "priority", task.Priority, *input.Priority)
		task.Priority = *input.Priority
	}

	if input.ClearDueDate {
		if task.DueDate != nil {
			_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionDueDateChanged, "due_date", task.DueDate.Format(time.DateOnly), "")
			task.DueDate = nil
		}
	} else if input.DueDate != nil {
		old := ""
		if task.DueDate != nil {
			old = task.DueDate.Format(time.DateOnly)
		}
		_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionDueDateChanged, "due_date", old, input.DueDate.Format(time.DateOnly))
		task.DueDate = input.DueDate
	}

	if input.StartDate != nil {
		task.StartDate = input.StartDate
	}

	if task.StartDate != nil && task.DueDate != nil && task.StartDate.After(*task.DueDate) {
		return nil, ErrInvalidDateRange
	}

	if input.SortOrder != nil {
		task.SortOrder = *input.SortOrder
	}

	task.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, task); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("task updated", "task_id", task.ID)

	// Emit events
	if s.eventEmitter != nil {
		deepLink := fmt.Sprintf("/work/tasks/%s", task.ID)
		if task.ProjectID != nil {
			deepLink = fmt.Sprintf("/work/projects/%s/tasks/%s", task.ProjectID, task.ID)
		}

		// Resolve the actor's display name once if an event will be emitted.
		actorName := ""
		if (statusChanged || assigneeChanged) && task.AssigneeID != nil && *task.AssigneeID != actorID {
			actorName = s.resolveActorName(ctx, actorID)
		}

		if statusChanged && task.AssigneeID != nil && *task.AssigneeID != actorID {
			_ = s.eventEmitter.EmitTaskEvent(ctx, models.EventPayload{
				Type:          event.EventWorkTaskStatusChanged,
				Priority:      models.PriorityNormal,
				ActorID:       actorID.String(),
				ActorName:     actorName,
				ResourceID:    task.ID.String(),
				ModuleID:      event.ModuleWork,
				Title:         task.Title + " status changed",
				Body:          fmt.Sprintf("Status changed from %s", oldStatusName),
				DeepLink:      deepLink,
				GroupKey:      "task-status:" + task.ID.String(),
				TargetUserIDs: []string{task.AssigneeID.String()},
				Timestamp:     time.Now(),
			})
		}

		if assigneeChanged && task.AssigneeID != nil && *task.AssigneeID != actorID {
			_ = s.eventEmitter.EmitTaskEvent(ctx, models.EventPayload{
				Type:          event.EventWorkTaskAssigned,
				Priority:      models.PriorityNormal,
				ActorID:       actorID.String(),
				ActorName:     actorName,
				ResourceID:    task.ID.String(),
				ModuleID:      event.ModuleWork,
				Title:         "Assigned to task: " + task.Title,
				Body:          "You have been assigned to this task",
				DeepLink:      deepLink,
				TargetUserIDs: []string{task.AssigneeID.String()},
				Timestamp:     time.Now(),
			})
		}
	}

	return s.repo.GetByID(ctx, task.ID, tenantID)
}

// MoveTask moves a task to a different status column (Kanban move)
func (s *Service) MoveTask(ctx context.Context, tenantID, taskID uuid.UUID, newStatusID uuid.UUID, newSortOrder float64, actorID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, taskID, tenantID)
	if err != nil {
		return err
	}

	// Check membership
	if existing.ProjectID != nil {
		if err := s.requireProjectEditor(ctx, *existing.ProjectID, tenantID, actorID); err != nil {
			return err
		}
	}

	oldStatusName := ""
	if existing.StatusName != nil {
		oldStatusName = *existing.StatusName
	}

	// Get new status info
	newStatusName, isClosed, statusErr := s.repo.GetStatusByID(ctx, newStatusID)
	if statusErr != nil {
		return statusErr
	}

	// completed_at tracks the closed-status transition; unaffected transitions keep the
	// existing value so it's folded into the same write as the status/sort_order move
	// instead of a second, separately-failable Update call.
	newCompletedAt := existing.CompletedAt
	if isClosed && existing.CompletedAt == nil {
		now := time.Now()
		newCompletedAt = &now
	} else if !isClosed && existing.CompletedAt != nil {
		newCompletedAt = nil
	}

	if moveErr := s.repo.MoveTask(ctx, tenantID, taskID, newStatusID, newSortOrder, newCompletedAt); moveErr != nil {
		return moveErr
	}

	// Log activity
	_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionStatusChanged, "status", oldStatusName, newStatusName)

	slog.Info("task moved",
		"task_id", taskID,
		"new_status", newStatusName,
	)

	// Emit event
	if s.eventEmitter != nil && existing.AssigneeID != nil && *existing.AssigneeID != actorID {
		deepLink := fmt.Sprintf("/work/tasks/%s", taskID)
		if existing.ProjectID != nil {
			deepLink = fmt.Sprintf("/work/projects/%s/tasks/%s", existing.ProjectID, taskID)
		}
		_ = s.eventEmitter.EmitTaskEvent(ctx, models.EventPayload{
			Type:          event.EventWorkTaskStatusChanged,
			Priority:      models.PriorityNormal,
			ActorID:       actorID.String(),
			ActorName:     s.resolveActorName(ctx, actorID),
			ResourceID:    taskID.String(),
			ModuleID:      event.ModuleWork,
			Title:         existing.Title + " moved to " + newStatusName,
			Body:          fmt.Sprintf("Status changed from %s to %s", oldStatusName, newStatusName),
			DeepLink:      deepLink,
			GroupKey:      "task-status:" + taskID.String(),
			TargetUserIDs: []string{existing.AssigneeID.String()},
			Timestamp:     time.Now(),
		})
	}

	return nil
}

// Delete removes a task. A non-nil ownerID is the own scope of
// work:task:delete: only a task ownerID created is deleted, any other answers
// ErrNotFound so a member cannot probe foreign task ids.
func (s *Service) Delete(ctx context.Context, tenantID, taskID uuid.UUID, actorID uuid.UUID, ownerID *uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, taskID, tenantID)
	if err != nil {
		return err
	}
	if ownerID != nil && existing.CreatedBy != *ownerID {
		return ErrNotFound
	}

	// Check project membership
	if existing.ProjectID != nil {
		if err := s.requireProjectEditor(ctx, *existing.ProjectID, tenantID, actorID); err != nil {
			return err
		}
	}

	if deleteErr := s.repo.Delete(ctx, taskID, tenantID); deleteErr != nil {
		return deleteErr
	}

	slog.Info("task deleted",
		"task_id", taskID,
		"title", existing.Title,
	)

	return nil
}

// GetByID retrieves a task by ID, scoped to the given tenant.
func (s *Service) GetByID(ctx context.Context, tenantID, taskID uuid.UUID) (*models.TaskWithRelations, error) {
	return s.repo.GetByID(ctx, taskID, tenantID)
}

// List retrieves tasks with filtering and pagination, scoped to the given tenant.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, filters TaskFilters) ([]models.TaskWithRelations, int, error) {
	return s.repo.List(ctx, tenantID, filters)
}

// DepInput contains the data needed to create a task dependency
type DepInput struct {
	TenantID       uuid.UUID
	SourceTaskID   uuid.UUID
	TargetTaskID   uuid.UUID
	DependencyType string
	CreatedBy      uuid.UUID
}

// CreateDependency creates a dependency between two tasks
func (s *Service) CreateDependency(ctx context.Context, input DepInput) (*models.TaskDependency, error) {
	// Validate dependency type
	if !models.ValidDependencyTypes[input.DependencyType] {
		return nil, ErrInvalidDependencyType
	}

	// Check self-dependency
	if input.SourceTaskID == input.TargetTaskID {
		return nil, ErrSelfDependency
	}

	// Check for duplicates
	existing, err := s.repo.ListDependencies(ctx, input.SourceTaskID)
	if err != nil {
		return nil, err
	}
	for _, dep := range existing {
		if dep.SourceTaskID == input.SourceTaskID && dep.TargetTaskID == input.TargetTaskID && dep.DependencyType == input.DependencyType {
			return nil, ErrDuplicateDependency
		}
	}

	// Check for cycles on blocking dependencies
	if input.DependencyType == models.DepTypeBlocks || input.DependencyType == models.DepTypeBlockedBy {
		hasCycle, cycleErr := s.repo.HasCycle(ctx, input.SourceTaskID, input.TargetTaskID)
		if cycleErr != nil {
			return nil, cycleErr
		}
		if hasCycle {
			return nil, ErrCircularDependency
		}
	}

	dep := &models.TaskDependency{
		ID:             uuid.New(),
		TenantID:       input.TenantID,
		SourceTaskID:   input.SourceTaskID,
		TargetTaskID:   input.TargetTaskID,
		DependencyType: input.DependencyType,
		CreatedBy:      input.CreatedBy,
		CreatedAt:      time.Now(),
	}

	if createErr := s.repo.CreateDependency(ctx, dep); createErr != nil {
		return nil, createErr
	}

	slog.Info("task dependency created",
		"source_task_id", dep.SourceTaskID,
		"target_task_id", dep.TargetTaskID,
		"type", dep.DependencyType,
	)

	return dep, nil
}

// DeleteDependency removes a task dependency. A non-nil ownerID is the own
// scope of work:task:edit: the dependency shows on both its tasks, so owning
// either one is enough; any other answers ErrNotFound.
func (s *Service) DeleteDependency(ctx context.Context, tenantID, depID uuid.UUID, ownerID *uuid.UUID) error {
	if ownerID != nil {
		source, target, err := s.repo.DependencyTaskIDs(ctx, tenantID, depID)
		if err != nil {
			return err
		}
		if s.RequireOwnTask(ctx, tenantID, source, ownerID) != nil {
			if err := s.RequireOwnTask(ctx, tenantID, target, ownerID); err != nil {
				return err
			}
		}
	}
	return s.repo.DeleteDependency(ctx, tenantID, depID)
}

// UnlinkEntity removes an entity link of the tenant. A non-nil ownerID is the
// own scope of work:task:edit on the link's task (see RequireOwnTask).
func (s *Service) UnlinkEntity(ctx context.Context, tenantID, linkID uuid.UUID, ownerID *uuid.UUID) error {
	if ownerID != nil {
		taskID, err := s.repo.EntityLinkTaskID(ctx, tenantID, linkID)
		if err != nil {
			return err
		}
		if err := s.RequireOwnTask(ctx, tenantID, taskID, ownerID); err != nil {
			return err
		}
	}
	return s.repo.UnlinkEntity(ctx, tenantID, linkID)
}

// ListMyTasks returns tasks assigned to or created by the user, scoped to the given tenant.
func (s *Service) ListMyTasks(ctx context.Context, tenantID, userID uuid.UUID, filters TaskFilters) ([]models.TaskWithRelations, int, error) {
	// Override filter to target the user
	filters.AssigneeID = &userID
	return s.repo.List(ctx, tenantID, filters)
}

// CreateFromTemplate copies all tasks (and their dependencies) from a source
// project into a target project. statusMap translates a source status ID to the
// matching status of the target project (statuses belong to one project); a task
// whose status has no entry lands without a status instead of pointing into the
// source project. Assignee and dates are deliberately not carried over.
func (s *Service) CreateFromTemplate(ctx context.Context, tenantID, sourceProjectID, targetProjectID uuid.UUID, actorID uuid.UUID, statusMap map[uuid.UUID]uuid.UUID) error {
	// Both projects must belong to the caller's tenant before anything is read or written.
	for _, id := range []uuid.UUID{sourceProjectID, targetProjectID} {
		if _, err := s.projectRepo.GetByID(ctx, id, tenantID); err != nil {
			return fmt.Errorf("project: %w", err)
		}
	}

	// Get all tasks from source project (ordered by depth ASC to process parents first)
	sourceTasks, err := s.repo.ListByProject(ctx, sourceProjectID)
	if err != nil {
		return fmt.Errorf("list source tasks: %w", err)
	}

	if len(sourceTasks) == 0 {
		return nil
	}

	// Build old_id -> new_id mapping
	idMapping := make(map[uuid.UUID]uuid.UUID)
	now := time.Now()

	for _, src := range sourceTasks {
		taskNumber, numErr := s.repo.GetNextTaskNumber(ctx, tenantID, targetProjectID)
		if numErr != nil {
			return fmt.Errorf("get task number: %w", numErr)
		}

		newID := uuid.New()
		idMapping[src.ID] = newID

		// Remap parent_task_id
		var parentID *uuid.UUID
		if src.ParentTaskID != nil {
			if mapped, ok := idMapping[*src.ParentTaskID]; ok {
				parentID = &mapped
			}
		}

		var statusID *uuid.UUID
		if src.StatusID != nil {
			if mapped, ok := statusMap[*src.StatusID]; ok {
				statusID = &mapped
			}
		}

		newTask := &models.Task{
			ID:           newID,
			TenantID:     tenantID,
			ProjectID:    &targetProjectID,
			TaskNumber:   taskNumber,
			Title:        src.Title,
			Description:  src.Description,
			StatusID:     statusID,
			Priority:     src.Priority,
			ParentTaskID: parentID,
			Depth:        src.Depth,
			SortOrder:    src.SortOrder,
			CreatedBy:    actorID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if createErr := s.repo.Create(ctx, newTask); createErr != nil {
			return fmt.Errorf("create template task: %w", createErr)
		}
	}

	// Copy dependencies
	sourceDeps, err := s.repo.ListDependenciesByProject(ctx, sourceProjectID)
	if err != nil {
		return fmt.Errorf("list source dependencies: %w", err)
	}

	for _, dep := range sourceDeps {
		newSourceID, okS := idMapping[dep.SourceTaskID]
		newTargetID, okT := idMapping[dep.TargetTaskID]
		if !okS || !okT {
			continue
		}

		newDep := &models.TaskDependency{
			ID:             uuid.New(),
			TenantID:       tenantID,
			SourceTaskID:   newSourceID,
			TargetTaskID:   newTargetID,
			DependencyType: dep.DependencyType,
			CreatedBy:      actorID,
			CreatedAt:      now,
		}
		if createErr := s.repo.CreateDependency(ctx, newDep); createErr != nil {
			return fmt.Errorf("create template dependency: %w", createErr)
		}
	}

	slog.Info("tasks copied from template",
		"source_project_id", sourceProjectID,
		"target_project_id", targetProjectID,
		"task_count", len(sourceTasks),
	)

	return nil
}

// logActivity creates a task activity entry
func (s *Service) logActivity(ctx context.Context, taskID, actorID uuid.UUID, action string, fieldName *string, oldValue, newValue *json.RawMessage) error {
	activity := &models.TaskActivity{
		ID:        uuid.New(),
		TaskID:    taskID,
		ActorID:   actorID,
		Action:    action,
		FieldName: fieldName,
		OldValue:  oldValue,
		NewValue:  newValue,
		CreatedAt: time.Now(),
	}
	return s.repo.CreateActivity(ctx, activity)
}

// logFieldChange is a convenience method to log a field change with string values
func (s *Service) logFieldChange(ctx context.Context, taskID, actorID uuid.UUID, action, fieldName, oldVal, newVal string) error {
	fn := fieldName
	oldJSON := jsonString(oldVal)
	newJSON := jsonString(newVal)
	return s.logActivity(ctx, taskID, actorID, action, &fn, oldJSON, newJSON)
}

// jsonString converts a string to a json.RawMessage pointer
func jsonString(s string) *json.RawMessage {
	data, _ := json.Marshal(s)
	raw := json.RawMessage(data)
	return &raw
}
