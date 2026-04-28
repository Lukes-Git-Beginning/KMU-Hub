package task

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock Repository ---

type mockRepo struct {
	tasks        map[uuid.UUID]*models.TaskWithRelations
	rawTasks     map[uuid.UUID]*models.Task
	deps         []models.TaskDependency
	activities   []models.TaskActivity
	nextNumber   int
	hasCycleVal  bool
	statusName   string
	statusClosed bool
	projectTasks []models.Task
	projectDeps  []models.TaskDependency
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		tasks:      make(map[uuid.UUID]*models.TaskWithRelations),
		rawTasks:   make(map[uuid.UUID]*models.Task),
		nextNumber: 1,
	}
}

func (m *mockRepo) Create(_ context.Context, task *models.Task) error {
	tw := &models.TaskWithRelations{Task: *task}
	m.tasks[task.ID] = tw
	m.rawTasks[task.ID] = task
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id uuid.UUID) (*models.TaskWithRelations, error) {
	tw, ok := m.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return tw, nil
}

func (m *mockRepo) List(_ context.Context, _ uuid.UUID, filters TaskFilters) ([]models.TaskWithRelations, int, error) {
	var result []models.TaskWithRelations
	for _, tw := range m.tasks {
		if filters.AssigneeID != nil && (tw.AssigneeID == nil || *tw.AssigneeID != *filters.AssigneeID) {
			continue
		}
		if filters.ProjectID != nil && (tw.ProjectID == nil || *tw.ProjectID != *filters.ProjectID) {
			continue
		}
		result = append(result, *tw)
	}
	return result, len(result), nil
}

func (m *mockRepo) Update(_ context.Context, task *models.Task) error {
	tw, ok := m.tasks[task.ID]
	if !ok {
		return ErrNotFound
	}
	tw.Task = *task
	m.rawTasks[task.ID] = task
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.tasks, id)
	delete(m.rawTasks, id)
	return nil
}

func (m *mockRepo) MoveTask(_ context.Context, taskID uuid.UUID, newStatusID uuid.UUID, newSortOrder float64) error {
	tw, ok := m.tasks[taskID]
	if !ok {
		return ErrNotFound
	}
	tw.StatusID = &newStatusID
	tw.SortOrder = newSortOrder
	return nil
}

func (m *mockRepo) GetNextTaskNumber(_ context.Context, _ uuid.UUID) (int, error) {
	num := m.nextNumber
	m.nextNumber++
	return num, nil
}

func (m *mockRepo) GetSubtasks(_ context.Context, _ uuid.UUID, _ int) ([]models.TaskWithRelations, error) {
	return nil, nil
}

func (m *mockRepo) GetParentChain(_ context.Context, _ uuid.UUID) ([]models.Task, error) {
	return nil, nil
}

func (m *mockRepo) GetDepth(_ context.Context, taskID uuid.UUID) (int, error) {
	tw, ok := m.tasks[taskID]
	if !ok {
		return 0, ErrNotFound
	}
	return tw.Depth, nil
}

func (m *mockRepo) CreateDependency(_ context.Context, dep *models.TaskDependency) error {
	m.deps = append(m.deps, *dep)
	return nil
}

func (m *mockRepo) DeleteDependency(_ context.Context, id uuid.UUID) error {
	for i, d := range m.deps {
		if d.ID == id {
			m.deps = append(m.deps[:i], m.deps[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepo) ListDependencies(_ context.Context, taskID uuid.UUID) ([]models.TaskDependency, error) {
	var result []models.TaskDependency
	for _, d := range m.deps {
		if d.SourceTaskID == taskID || d.TargetTaskID == taskID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockRepo) HasCycle(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.hasCycleVal, nil
}

func (m *mockRepo) CreateActivity(_ context.Context, activity *models.TaskActivity) error {
	m.activities = append(m.activities, *activity)
	return nil
}

func (m *mockRepo) ListActivities(_ context.Context, taskID uuid.UUID, page, pageSize int) ([]models.TaskActivity, int, error) {
	var result []models.TaskActivity
	for _, a := range m.activities {
		if a.TaskID == taskID {
			result = append(result, a)
		}
	}
	return result, len(result), nil
}

func (m *mockRepo) LinkEntity(_ context.Context, link *models.TaskEntityLink) error {
	return nil
}

func (m *mockRepo) UnlinkEntity(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepo) ListEntityLinks(_ context.Context, _ uuid.UUID) ([]models.TaskEntityLink, error) {
	return nil, nil
}

func (m *mockRepo) ListTasksForEntity(_ context.Context, _ string, _ uuid.UUID) ([]models.TaskWithRelations, error) {
	return nil, nil
}

func (m *mockRepo) AttachFile(_ context.Context, _ *models.TaskFile) error {
	return nil
}

func (m *mockRepo) RemoveFile(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepo) ListFiles(_ context.Context, _ uuid.UUID) ([]models.TaskFile, error) {
	return nil, nil
}

func (m *mockRepo) SetCustomFieldValues(_ context.Context, _ uuid.UUID, _ map[uuid.UUID]any) error {
	return nil
}

func (m *mockRepo) GetCustomFieldValues(_ context.Context, _ uuid.UUID) (map[string]any, error) {
	return nil, nil
}

func (m *mockRepo) Search(_ context.Context, _ string, _ TaskSearchFilters) ([]models.TaskWithRelations, int, error) {
	return nil, 0, nil
}

func (m *mockRepo) ListByProject(_ context.Context, _ uuid.UUID) ([]models.Task, error) {
	return m.projectTasks, nil
}

func (m *mockRepo) ListDependenciesByProject(_ context.Context, _ uuid.UUID) ([]models.TaskDependency, error) {
	return m.projectDeps, nil
}

func (m *mockRepo) GetStatusByID(_ context.Context, _ uuid.UUID) (string, bool, error) {
	return m.statusName, m.statusClosed, nil
}

// --- Mock Project Repository ---

type mockProjectRepo struct {
	members map[string]*models.ProjectMember // key: "projectID:userID"
}

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{
		members: make(map[string]*models.ProjectMember),
	}
}

func (m *mockProjectRepo) addMember(projectID, userID uuid.UUID, role string) {
	key := projectID.String() + ":" + userID.String()
	m.members[key] = &models.ProjectMember{
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
		AddedAt:   time.Now(),
	}
}

func (m *mockProjectRepo) GetMember(_ context.Context, projectID, userID uuid.UUID) (*models.ProjectMember, error) {
	key := projectID.String() + ":" + userID.String()
	member, ok := m.members[key]
	if !ok {
		return nil, ErrNotProjectMember
	}
	return member, nil
}

func (m *mockProjectRepo) IsMember(_ context.Context, projectID, userID uuid.UUID) (bool, error) {
	key := projectID.String() + ":" + userID.String()
	_, ok := m.members[key]
	return ok, nil
}

// Unused methods - satisfy interface
func (m *mockProjectRepo) Create(_ context.Context, _ *models.Project) error { return nil }
func (m *mockProjectRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.Project, error) {
	return nil, nil
}
func (m *mockProjectRepo) List(_ context.Context, _ uuid.UUID, _ bool, _ bool) ([]models.ProjectWithDetails, error) {
	return nil, nil
}
func (m *mockProjectRepo) Update(_ context.Context, _ *models.Project) error { return nil }
func (m *mockProjectRepo) Archive(_ context.Context, _ uuid.UUID) error      { return nil }
func (m *mockProjectRepo) GetProjectKey(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}
func (m *mockProjectRepo) KeyExists(_ context.Context, _ string) (bool, error) { return false, nil }
func (m *mockProjectRepo) AddMember(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockProjectRepo) RemoveMember(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockProjectRepo) UpdateMemberRole(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockProjectRepo) ListMembers(_ context.Context, _ uuid.UUID) ([]models.ProjectMember, error) {
	return nil, nil
}
func (m *mockProjectRepo) CountOwners(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *mockProjectRepo) SaveAsTemplate(_ context.Context, _ uuid.UUID, _, _ string, _ uuid.UUID) (*models.Project, error) {
	return nil, nil
}
func (m *mockProjectRepo) GetForTemplate(_ context.Context, _ uuid.UUID) (*models.Project, []models.ProjectStatus, error) {
	return nil, nil, nil
}
func (m *mockProjectRepo) GetUserPreference(_ context.Context, _, _ uuid.UUID) (*models.UserProjectPreference, error) {
	return nil, nil
}
func (m *mockProjectRepo) SetUserPreference(_ context.Context, _ *models.UserProjectPreference) error {
	return nil
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
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)
	emitter := &mockEventEmitter{}
	svc.SetEventEmitter(emitter)

	projectID := uuid.New()
	userID := uuid.New()
	projRepo.addMember(projectID, userID, models.ProjectRoleMember)

	t.Run("create task in project", func(t *testing.T) {
		result, err := svc.Create(ctx, CreateInput{
			ProjectID: &projectID,
			Title:     "Build feature X",
			Priority:  models.TaskPriorityHigh,
			CreatedBy: userID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Title != "Build feature X" {
			t.Errorf("expected title 'Build feature X', got %q", result.Title)
		}
		if result.Priority != models.TaskPriorityHigh {
			t.Errorf("expected priority high, got %q", result.Priority)
		}
		if result.TaskNumber != 1 {
			t.Errorf("expected task number 1, got %d", result.TaskNumber)
		}
		if result.Depth != 0 {
			t.Errorf("expected depth 0, got %d", result.Depth)
		}

		// Verify activity was logged
		if len(repo.activities) != 1 {
			t.Fatalf("expected 1 activity, got %d", len(repo.activities))
		}
		if repo.activities[0].Action != models.TaskActionCreated {
			t.Errorf("expected action 'created', got %q", repo.activities[0].Action)
		}
	})

	t.Run("create standalone task", func(t *testing.T) {
		result, err := svc.Create(ctx, CreateInput{
			Title:     "Personal task",
			Priority:  models.TaskPriorityLow,
			CreatedBy: userID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.ProjectID != nil {
			t.Error("expected nil project_id for standalone task")
		}
		if result.TaskNumber != 0 {
			t.Errorf("expected task number 0 for standalone, got %d", result.TaskNumber)
		}
	})

	t.Run("create with assignee emits event", func(t *testing.T) {
		emitter.events = nil
		assigneeID := uuid.New()
		_, err := svc.Create(ctx, CreateInput{
			ProjectID:  &projectID,
			Title:      "Assigned task",
			Priority:   models.TaskPriorityMedium,
			AssigneeID: &assigneeID,
			CreatedBy:  userID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(emitter.events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(emitter.events))
		}
		if emitter.events[0].Type != "work.task.created" {
			t.Errorf("expected event type work.task.created, got %q", emitter.events[0].Type)
		}
	})

	t.Run("empty title rejected", func(t *testing.T) {
		_, err := svc.Create(ctx, CreateInput{
			Title:     "",
			CreatedBy: userID,
		})
		if err != ErrTitleRequired {
			t.Errorf("expected ErrTitleRequired, got %v", err)
		}
	})

	t.Run("title too long rejected", func(t *testing.T) {
		longTitle := ""
		for i := 0; i < 501; i++ {
			longTitle += "a"
		}
		_, err := svc.Create(ctx, CreateInput{
			Title:     longTitle,
			CreatedBy: userID,
		})
		if err != ErrTitleTooLong {
			t.Errorf("expected ErrTitleTooLong, got %v", err)
		}
	})

	t.Run("invalid priority rejected", func(t *testing.T) {
		_, err := svc.Create(ctx, CreateInput{
			Title:     "Task",
			Priority:  "super_important",
			CreatedBy: userID,
		})
		if err != ErrInvalidPriority {
			t.Errorf("expected ErrInvalidPriority, got %v", err)
		}
	})

	t.Run("default priority is medium", func(t *testing.T) {
		result, err := svc.Create(ctx, CreateInput{
			Title:     "Default priority task",
			CreatedBy: userID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Priority != models.TaskPriorityMedium {
			t.Errorf("expected default priority medium, got %q", result.Priority)
		}
	})
}

func TestService_CreateViewerCannotCreate(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	projectID := uuid.New()
	viewerID := uuid.New()
	projRepo.addMember(projectID, viewerID, models.ProjectRoleViewer)

	_, err := svc.Create(ctx, CreateInput{
		ProjectID: &projectID,
		Title:     "Should fail",
		Priority:  models.TaskPriorityMedium,
		CreatedBy: viewerID,
	})
	if err != ErrViewerCannotEdit {
		t.Errorf("expected ErrViewerCannotEdit, got %v", err)
	}
}

func TestService_CreateSubtaskDepthEnforcement(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	projectID := uuid.New()
	userID := uuid.New()
	projRepo.addMember(projectID, userID, models.ProjectRoleMember)

	// Create parent at depth 0
	parent, err := svc.Create(ctx, CreateInput{
		ProjectID: &projectID,
		Title:     "Level 0",
		CreatedBy: userID,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	// Create nested tasks up to depth 5
	currentParentID := parent.ID
	for depth := 1; depth <= MaxNestingDepth; depth++ {
		child, childErr := svc.Create(ctx, CreateInput{
			ProjectID:    &projectID,
			Title:        "Level " + string(rune('0'+depth)),
			ParentTaskID: &currentParentID,
			CreatedBy:    userID,
		})
		if childErr != nil {
			t.Fatalf("create child at depth %d: %v", depth, childErr)
		}
		if child.Depth != depth {
			t.Errorf("expected depth %d, got %d", depth, child.Depth)
		}
		currentParentID = child.ID
	}

	// Try to create at depth 6 -- should fail
	_, err = svc.Create(ctx, CreateInput{
		ProjectID:    &projectID,
		Title:        "Level 6 - too deep",
		ParentTaskID: &currentParentID,
		CreatedBy:    userID,
	})
	if err != ErrMaxDepthExceeded {
		t.Errorf("expected ErrMaxDepthExceeded, got %v", err)
	}
}

func TestService_CreateSubtaskDifferentProject(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	project1 := uuid.New()
	project2 := uuid.New()
	userID := uuid.New()
	projRepo.addMember(project1, userID, models.ProjectRoleMember)
	projRepo.addMember(project2, userID, models.ProjectRoleMember)

	parent, err := svc.Create(ctx, CreateInput{
		ProjectID: &project1,
		Title:     "Parent in project 1",
		CreatedBy: userID,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	// Try to create subtask in different project
	_, err = svc.Create(ctx, CreateInput{
		ProjectID:    &project2,
		Title:        "Child in project 2",
		ParentTaskID: &parent.ID,
		CreatedBy:    userID,
	})
	if err != ErrParentInDiffProject {
		t.Errorf("expected ErrParentInDiffProject, got %v", err)
	}
}

func TestService_CreateDependency(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	userID := uuid.New()

	// Create two tasks (standalone)
	task1, _ := svc.Create(ctx, CreateInput{Title: "Task 1", CreatedBy: userID})
	task2, _ := svc.Create(ctx, CreateInput{Title: "Task 2", CreatedBy: userID})

	t.Run("valid dependency", func(t *testing.T) {
		dep, err := svc.CreateDependency(ctx, DepInput{
			SourceTaskID:   task1.ID,
			TargetTaskID:   task2.ID,
			DependencyType: models.DepTypeBlocks,
			CreatedBy:      userID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dep.DependencyType != models.DepTypeBlocks {
			t.Errorf("expected type 'blocks', got %q", dep.DependencyType)
		}
	})

	t.Run("self dependency rejected", func(t *testing.T) {
		_, err := svc.CreateDependency(ctx, DepInput{
			SourceTaskID:   task1.ID,
			TargetTaskID:   task1.ID,
			DependencyType: models.DepTypeBlocks,
			CreatedBy:      userID,
		})
		if err != ErrSelfDependency {
			t.Errorf("expected ErrSelfDependency, got %v", err)
		}
	})

	t.Run("duplicate dependency rejected", func(t *testing.T) {
		_, err := svc.CreateDependency(ctx, DepInput{
			SourceTaskID:   task1.ID,
			TargetTaskID:   task2.ID,
			DependencyType: models.DepTypeBlocks,
			CreatedBy:      userID,
		})
		if err != ErrDuplicateDependency {
			t.Errorf("expected ErrDuplicateDependency, got %v", err)
		}
	})

	t.Run("invalid dependency type rejected", func(t *testing.T) {
		_, err := svc.CreateDependency(ctx, DepInput{
			SourceTaskID:   task1.ID,
			TargetTaskID:   task2.ID,
			DependencyType: "invalid_type",
			CreatedBy:      userID,
		})
		if err != ErrInvalidDependencyType {
			t.Errorf("expected ErrInvalidDependencyType, got %v", err)
		}
	})

	t.Run("cycle detected", func(t *testing.T) {
		repo.hasCycleVal = true
		task3, _ := svc.Create(ctx, CreateInput{Title: "Task 3", CreatedBy: userID})

		_, err := svc.CreateDependency(ctx, DepInput{
			SourceTaskID:   task3.ID,
			TargetTaskID:   task1.ID,
			DependencyType: models.DepTypeBlocks,
			CreatedBy:      userID,
		})
		if err != ErrCircularDependency {
			t.Errorf("expected ErrCircularDependency, got %v", err)
		}
		repo.hasCycleVal = false
	})

	t.Run("relates_to skips cycle check", func(t *testing.T) {
		repo.hasCycleVal = true // would fail if cycle check ran
		task4, _ := svc.Create(ctx, CreateInput{Title: "Task 4", CreatedBy: userID})

		_, err := svc.CreateDependency(ctx, DepInput{
			SourceTaskID:   task4.ID,
			TargetTaskID:   task1.ID,
			DependencyType: models.DepTypeRelatesTo,
			CreatedBy:      userID,
		})
		if err != nil {
			t.Errorf("expected no error for relates_to, got %v", err)
		}
		repo.hasCycleVal = false
	})
}

func TestService_MoveTask(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)
	emitter := &mockEventEmitter{}
	svc.SetEventEmitter(emitter)

	projectID := uuid.New()
	userID := uuid.New()
	assigneeID := uuid.New()
	projRepo.addMember(projectID, userID, models.ProjectRoleMember)

	statusID := uuid.New()
	task1, _ := svc.Create(ctx, CreateInput{
		ProjectID:  &projectID,
		Title:      "Movable task",
		StatusID:   &statusID,
		AssigneeID: &assigneeID,
		CreatedBy:  userID,
	})
	// Set the status name on the task for the test
	repo.tasks[task1.ID].StatusName = strPtr("To Do")

	t.Run("move to new status", func(t *testing.T) {
		repo.statusName = "In Progress"
		repo.statusClosed = false
		emitter.events = nil

		newStatusID := uuid.New()
		err := svc.MoveTask(ctx, task1.ID, newStatusID, 1.5, userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify event emitted (assignee != actor)
		if len(emitter.events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(emitter.events))
		}
		if emitter.events[0].Type != "work.task.status_changed" {
			t.Errorf("expected event type work.task.status_changed, got %q", emitter.events[0].Type)
		}
	})

	t.Run("move to closed status sets completed_at", func(t *testing.T) {
		repo.statusName = "Done"
		repo.statusClosed = true

		closedStatusID := uuid.New()
		err := svc.MoveTask(ctx, task1.ID, closedStatusID, 2.0, userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		updated := repo.tasks[task1.ID]
		if updated.CompletedAt == nil {
			t.Error("expected completed_at to be set when moving to closed status")
		}
	})

	t.Run("move from closed to open clears completed_at", func(t *testing.T) {
		repo.statusName = "Reopened"
		repo.statusClosed = false

		openStatusID := uuid.New()
		err := svc.MoveTask(ctx, task1.ID, openStatusID, 3.0, userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		updated := repo.tasks[task1.ID]
		if updated.CompletedAt != nil {
			t.Error("expected completed_at to be cleared when moving to open status")
		}
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)
	emitter := &mockEventEmitter{}
	svc.SetEventEmitter(emitter)

	projectID := uuid.New()
	userID := uuid.New()
	projRepo.addMember(projectID, userID, models.ProjectRoleMember)

	repo.statusName = "In Progress"
	repo.statusClosed = false

	task1, _ := svc.Create(ctx, CreateInput{
		ProjectID: &projectID,
		Title:     "Original title",
		Priority:  models.TaskPriorityMedium,
		CreatedBy: userID,
	})

	t.Run("update title logs activity", func(t *testing.T) {
		activitiesBefore := len(repo.activities)
		newTitle := "Updated title"
		result, err := svc.Update(ctx, task1.ID, UpdateInput{
			Title: &newTitle,
		}, userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Title != "Updated title" {
			t.Errorf("expected title 'Updated title', got %q", result.Title)
		}
		if len(repo.activities) <= activitiesBefore {
			t.Error("expected activity to be logged for title change")
		}
		lastActivity := repo.activities[len(repo.activities)-1]
		if lastActivity.Action != models.TaskActionTitleChanged {
			t.Errorf("expected action title_changed, got %q", lastActivity.Action)
		}
	})

	t.Run("update priority logs activity", func(t *testing.T) {
		activitiesBefore := len(repo.activities)
		newPriority := models.TaskPriorityUrgent
		_, err := svc.Update(ctx, task1.ID, UpdateInput{
			Priority: &newPriority,
		}, userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(repo.activities) <= activitiesBefore {
			t.Error("expected activity logged for priority change")
		}
	})

	t.Run("update assignee emits event", func(t *testing.T) {
		emitter.events = nil
		newAssignee := uuid.New()
		_, err := svc.Update(ctx, task1.ID, UpdateInput{
			AssigneeID: &newAssignee,
		}, userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(emitter.events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(emitter.events))
		}
		if emitter.events[0].Type != "work.task.assigned" {
			t.Errorf("expected event type work.task.assigned, got %q", emitter.events[0].Type)
		}
	})

	t.Run("update status emits event", func(t *testing.T) {
		// Set an assignee on the task
		assignee := uuid.New()
		repo.tasks[task1.ID].AssigneeID = &assignee

		emitter.events = nil
		newStatus := uuid.New()
		_, err := svc.Update(ctx, task1.ID, UpdateInput{
			StatusID: &newStatus,
		}, userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(emitter.events) != 1 {
			t.Fatalf("expected 1 event for status change, got %d", len(emitter.events))
		}
		if emitter.events[0].Type != "work.task.status_changed" {
			t.Errorf("expected event type work.task.status_changed, got %q", emitter.events[0].Type)
		}
	})

	t.Run("invalid priority rejected", func(t *testing.T) {
		badPriority := "nonexistent"
		_, err := svc.Update(ctx, task1.ID, UpdateInput{
			Priority: &badPriority,
		}, userID)
		if err != ErrInvalidPriority {
			t.Errorf("expected ErrInvalidPriority, got %v", err)
		}
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	projectID := uuid.New()
	userID := uuid.New()
	projRepo.addMember(projectID, userID, models.ProjectRoleMember)

	task1, _ := svc.Create(ctx, CreateInput{
		ProjectID: &projectID,
		Title:     "To delete",
		CreatedBy: userID,
	})

	err := svc.Delete(ctx, task1.ID, userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify task is gone
	_, err = svc.GetByID(ctx, task1.ID)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestService_DeleteViewerCannotDelete(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	projectID := uuid.New()
	ownerID := uuid.New()
	viewerID := uuid.New()
	projRepo.addMember(projectID, ownerID, models.ProjectRoleOwner)
	projRepo.addMember(projectID, viewerID, models.ProjectRoleViewer)

	task1, _ := svc.Create(ctx, CreateInput{
		ProjectID: &projectID,
		Title:     "Protected task",
		CreatedBy: ownerID,
	})

	err := svc.Delete(ctx, task1.ID, viewerID)
	if err != ErrViewerCannotEdit {
		t.Errorf("expected ErrViewerCannotEdit, got %v", err)
	}
}

func TestService_ListMyTasks(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	userID := uuid.New()
	otherID := uuid.New()
	projectID := uuid.New()
	projRepo.addMember(projectID, userID, models.ProjectRoleMember)
	projRepo.addMember(projectID, otherID, models.ProjectRoleMember)

	// Create task assigned to user
	_, _ = svc.Create(ctx, CreateInput{
		ProjectID:  &projectID,
		Title:      "My task",
		AssigneeID: &userID,
		CreatedBy:  otherID,
	})

	// Create task assigned to someone else
	_, _ = svc.Create(ctx, CreateInput{
		ProjectID:  &projectID,
		Title:      "Other task",
		AssigneeID: &otherID,
		CreatedBy:  otherID,
	})

	tasks, total, err := svc.ListMyTasks(ctx, uuid.Nil, userID, TaskFilters{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 task, got %d", total)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in results, got %d", len(tasks))
	}
	if tasks[0].Title != "My task" {
		t.Errorf("expected 'My task', got %q", tasks[0].Title)
	}
}

func TestService_CreateFromTemplate(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	sourceProjectID := uuid.New()
	targetProjectID := uuid.New()
	actorID := uuid.New()

	parentID := uuid.New()
	childID := uuid.New()

	// Set up source tasks
	repo.projectTasks = []models.Task{
		{
			ID:        parentID,
			ProjectID: &sourceProjectID,
			Title:     "Template Parent",
			Priority:  models.TaskPriorityMedium,
			Depth:     0,
			SortOrder: 1.0,
		},
		{
			ID:           childID,
			ProjectID:    &sourceProjectID,
			Title:        "Template Child",
			Priority:     models.TaskPriorityLow,
			ParentTaskID: &parentID,
			Depth:        1,
			SortOrder:    1.0,
		},
	}

	// Set up source dependencies
	repo.projectDeps = []models.TaskDependency{
		{
			ID:             uuid.New(),
			SourceTaskID:   parentID,
			TargetTaskID:   childID,
			DependencyType: models.DepTypeBlocks,
		},
	}

	err := svc.CreateFromTemplate(ctx, sourceProjectID, targetProjectID, actorID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify 2 tasks created
	createdCount := 0
	for _, tw := range repo.tasks {
		if tw.ProjectID != nil && *tw.ProjectID == targetProjectID {
			createdCount++
		}
	}
	if createdCount != 2 {
		t.Errorf("expected 2 tasks created in target project, got %d", createdCount)
	}

	// Verify dependencies were copied
	if len(repo.deps) != 1 {
		t.Errorf("expected 1 dependency copied, got %d", len(repo.deps))
	}

	// Verify parent_task_id was remapped (child should reference new parent)
	var childTask *models.TaskWithRelations
	for _, tw := range repo.tasks {
		if tw.Title == "Template Child" && tw.ProjectID != nil && *tw.ProjectID == targetProjectID {
			childTask = tw
			break
		}
	}
	if childTask == nil {
		t.Fatal("child task not found in target project")
	}
	if childTask.ParentTaskID == nil {
		t.Error("expected child to have parent_task_id set")
	} else if *childTask.ParentTaskID == parentID {
		t.Error("parent_task_id should be remapped to new parent, not original")
	}
}

func TestService_NotProjectMember(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	projectID := uuid.New()
	nonMemberID := uuid.New()

	_, err := svc.Create(ctx, CreateInput{
		ProjectID: &projectID,
		Title:     "Should fail",
		CreatedBy: nonMemberID,
	})
	if err != ErrNotProjectMember {
		t.Errorf("expected ErrNotProjectMember, got %v", err)
	}
}

func TestService_UpdateNonExistent(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	title := "new"
	_, err := svc.Update(ctx, uuid.New(), UpdateInput{Title: &title}, uuid.New())
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_UpdateDueDate(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	userID := uuid.New()
	task1, _ := svc.Create(ctx, CreateInput{
		Title:     "Task with due date",
		CreatedBy: userID,
	})

	activitiesBefore := len(repo.activities)
	dueDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	_, err := svc.Update(ctx, task1.ID, UpdateInput{
		DueDate: &dueDate,
	}, userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.activities) <= activitiesBefore {
		t.Error("expected activity logged for due date change")
	}
	lastActivity := repo.activities[len(repo.activities)-1]
	if lastActivity.Action != models.TaskActionDueDateChanged {
		t.Errorf("expected action due_date_changed, got %q", lastActivity.Action)
	}
	// Verify new value contains the date
	var newVal string
	_ = json.Unmarshal(*lastActivity.NewValue, &newVal)
	if newVal != "2026-03-15" {
		t.Errorf("expected new value '2026-03-15', got %q", newVal)
	}
}

// --- Helper ---

func strPtr(s string) *string {
	return &s
}
