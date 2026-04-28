package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/work/project"
)

// Service handles task business logic
type Service struct {
	repo         Repository
	projectRepo  project.Repository
	eventEmitter EventEmitter
}

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
		priority = models.TaskPriorityMedium
	}
	if !models.ValidTaskPriorities[priority] {
		return nil, ErrInvalidPriority
	}

	// Check project membership if project_id is set
	if input.ProjectID != nil {
		member, err := s.projectRepo.GetMember(ctx, *input.ProjectID, input.CreatedBy)
		if err != nil {
			return nil, ErrNotProjectMember
		}
		if member.Role == models.ProjectRoleViewer {
			return nil, ErrViewerCannotEdit
		}
	}

	// Handle nesting
	var depth int
	if input.ParentTaskID != nil {
		parentTask, err := s.repo.GetByID(ctx, *input.ParentTaskID)
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
		num, err := s.repo.GetNextTaskNumber(ctx, *input.ProjectID)
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
			ResourceID:    task.ID.String(),
			ModuleID:      event.ModuleWork,
			Title:         "New task assigned: " + task.Title,
			Body:          "You have been assigned a new task",
			DeepLink:      deepLink,
			TargetUserIDs: []string{input.AssigneeID.String()},
			Timestamp:     now,
		})
	}

	return s.repo.GetByID(ctx, task.ID)
}

// UpdateInput contains the data that can be updated on a task
type UpdateInput struct {
	Title       *string
	Description *string
	StatusID    *uuid.UUID
	Priority    *string
	AssigneeID  *uuid.UUID // uuid.Nil to clear
	DueDate     *time.Time
	SortOrder   *float64
}

// Update updates an existing task
func (s *Service) Update(ctx context.Context, taskID uuid.UUID, input UpdateInput, actorID uuid.UUID) (*models.TaskWithRelations, error) {
	existing, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Check project membership
	if existing.ProjectID != nil {
		member, memberErr := s.projectRepo.GetMember(ctx, *existing.ProjectID, actorID)
		if memberErr != nil {
			return nil, ErrNotProjectMember
		}
		if member.Role == models.ProjectRoleViewer {
			return nil, ErrViewerCannotEdit
		}
	}

	task := &existing.Task

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

	if input.DueDate != nil {
		old := ""
		if task.DueDate != nil {
			old = task.DueDate.Format(time.DateOnly)
		}
		_ = s.logFieldChange(ctx, taskID, actorID, models.TaskActionDueDateChanged, "due_date", old, input.DueDate.Format(time.DateOnly))
		task.DueDate = input.DueDate
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

		if statusChanged && task.AssigneeID != nil && *task.AssigneeID != actorID {
			_ = s.eventEmitter.EmitTaskEvent(ctx, models.EventPayload{
				Type:          event.EventWorkTaskStatusChanged,
				Priority:      models.PriorityNormal,
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

	return s.repo.GetByID(ctx, task.ID)
}

// MoveTask moves a task to a different status column (Kanban move)
func (s *Service) MoveTask(ctx context.Context, taskID uuid.UUID, newStatusID uuid.UUID, newSortOrder float64, actorID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Check membership
	if existing.ProjectID != nil {
		member, memberErr := s.projectRepo.GetMember(ctx, *existing.ProjectID, actorID)
		if memberErr != nil {
			return ErrNotProjectMember
		}
		if member.Role == models.ProjectRoleViewer {
			return ErrViewerCannotEdit
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

	if moveErr := s.repo.MoveTask(ctx, taskID, newStatusID, newSortOrder); moveErr != nil {
		return moveErr
	}

	// Handle completed_at based on closed status
	task := &existing.Task
	if isClosed && task.CompletedAt == nil {
		now := time.Now()
		task.CompletedAt = &now
		task.StatusID = &newStatusID
		task.UpdatedAt = time.Now()
		_ = s.repo.Update(ctx, task)
	} else if !isClosed && task.CompletedAt != nil {
		task.CompletedAt = nil
		task.StatusID = &newStatusID
		task.UpdatedAt = time.Now()
		_ = s.repo.Update(ctx, task)
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

// Delete removes a task
func (s *Service) Delete(ctx context.Context, taskID uuid.UUID, actorID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Check project membership
	if existing.ProjectID != nil {
		member, memberErr := s.projectRepo.GetMember(ctx, *existing.ProjectID, actorID)
		if memberErr != nil {
			return ErrNotProjectMember
		}
		if member.Role == models.ProjectRoleViewer {
			return ErrViewerCannotEdit
		}
	}

	if deleteErr := s.repo.Delete(ctx, taskID); deleteErr != nil {
		return deleteErr
	}

	slog.Info("task deleted",
		"task_id", taskID,
		"title", existing.Title,
	)

	return nil
}

// GetByID retrieves a task by ID
func (s *Service) GetByID(ctx context.Context, taskID uuid.UUID) (*models.TaskWithRelations, error) {
	return s.repo.GetByID(ctx, taskID)
}

// List retrieves tasks with filtering and pagination, scoped to the given tenant.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, filters TaskFilters) ([]models.TaskWithRelations, int, error) {
	return s.repo.List(ctx, tenantID, filters)
}

// DepInput contains the data needed to create a task dependency
type DepInput struct {
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

// DeleteDependency removes a task dependency
func (s *Service) DeleteDependency(ctx context.Context, depID uuid.UUID) error {
	return s.repo.DeleteDependency(ctx, depID)
}

// ListMyTasks returns tasks assigned to or created by the user, scoped to the given tenant.
func (s *Service) ListMyTasks(ctx context.Context, tenantID, userID uuid.UUID, filters TaskFilters) ([]models.TaskWithRelations, int, error) {
	// Override filter to target the user
	filters.AssigneeID = &userID
	return s.repo.List(ctx, tenantID, filters)
}

// CreateFromTemplate copies all tasks from a source project into a target project
func (s *Service) CreateFromTemplate(ctx context.Context, sourceProjectID, targetProjectID uuid.UUID, actorID uuid.UUID) error {
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
		taskNumber, numErr := s.repo.GetNextTaskNumber(ctx, targetProjectID)
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

		newTask := &models.Task{
			ID:           newID,
			ProjectID:    &targetProjectID,
			TaskNumber:   taskNumber,
			Title:        src.Title,
			Description:  src.Description,
			StatusID:     src.StatusID,
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
