package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/comment"
	"github.com/kmuhub/kmuhub/internal/work/customfield"
	"github.com/kmuhub/kmuhub/internal/work/label"
	"github.com/kmuhub/kmuhub/internal/work/project"
	wstatus "github.com/kmuhub/kmuhub/internal/work/status"
	"github.com/kmuhub/kmuhub/internal/work/task"
	"github.com/kmuhub/kmuhub/internal/work/timeentry"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// ============================================================================
// Minimal stub repositories for packages without exported mocks
// ============================================================================

// projectStubRepo implements project.Repository — all methods are no-ops.
type projectStubRepo struct{}

func (r *projectStubRepo) Create(_ context.Context, _ *models.Project) error { return nil }
func (r *projectStubRepo) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.Project, error) {
	return nil, project.ErrNotFound
}
func (r *projectStubRepo) List(_ context.Context, _, _ uuid.UUID, _ bool, _ bool) ([]models.ProjectWithDetails, error) {
	return nil, nil
}
func (r *projectStubRepo) Update(_ context.Context, _ *models.Project) error { return nil }
func (r *projectStubRepo) Archive(_ context.Context, _, _ uuid.UUID) error   { return nil }
func (r *projectStubRepo) GetProjectKey(_ context.Context, _, _ uuid.UUID) (string, error) {
	return "", project.ErrNotFound
}
func (r *projectStubRepo) KeyExists(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}
func (r *projectStubRepo) AddMember(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }
func (r *projectStubRepo) RemoveMember(_ context.Context, _, _ uuid.UUID) error        { return nil }
func (r *projectStubRepo) UpdateMemberRole(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (r *projectStubRepo) GetMember(_ context.Context, _, _ uuid.UUID) (*models.ProjectMember, error) {
	return nil, project.ErrNotFound
}
func (r *projectStubRepo) ListMembers(_ context.Context, _ uuid.UUID) ([]models.ProjectMember, error) {
	return nil, nil
}
func (r *projectStubRepo) IsMember(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil }
func (r *projectStubRepo) CountOwners(_ context.Context, _ uuid.UUID) (int, error)  { return 1, nil }
func (r *projectStubRepo) SaveAsTemplate(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ string, _ uuid.UUID) (*models.Project, error) {
	return nil, project.ErrNotFound
}
func (r *projectStubRepo) GetForTemplate(_ context.Context, _, _ uuid.UUID) (*models.Project, []models.ProjectStatus, error) {
	return nil, nil, project.ErrTemplateNotFound
}
func (r *projectStubRepo) GetUserPreference(_ context.Context, _, _ uuid.UUID) (*models.UserProjectPreference, error) {
	return nil, nil
}
func (r *projectStubRepo) SetUserPreference(_ context.Context, _ *models.UserProjectPreference) error {
	return nil
}

// statusStubRepo implements wstatus.Repository — all methods are no-ops.
type statusStubRepo struct{}

func (r *statusStubRepo) Create(_ context.Context, _ *models.ProjectStatus) error { return nil }
func (r *statusStubRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.ProjectStatus, error) {
	return nil, wstatus.ErrNotFound
}
func (r *statusStubRepo) ListByProject(_ context.Context, _ uuid.UUID) ([]models.ProjectStatus, error) {
	return nil, nil
}
func (r *statusStubRepo) Update(_ context.Context, _ *models.ProjectStatus) error { return nil }
func (r *statusStubRepo) Delete(_ context.Context, _ uuid.UUID) error             { return nil }
func (r *statusStubRepo) Reorder(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error { return nil }
func (r *statusStubRepo) CountTasksWithStatus(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (r *statusStubRepo) HasDefault(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil }
func (r *statusStubRepo) GetNextSortOrder(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (r *statusStubRepo) NameExistsInProject(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
	return false, nil
}
func (r *statusStubRepo) CopyStatusesForProject(_ context.Context, _, _ uuid.UUID) error { return nil }

// commentStubRepo implements comment.Repository for testing.
type commentStubRepo struct{}

func (r *commentStubRepo) Create(_ context.Context, _ *models.TaskComment) error { return nil }
func (r *commentStubRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.TaskComment, error) {
	return nil, comment.ErrNotFound
}
func (r *commentStubRepo) List(_ context.Context, _ uuid.UUID, _, _ int) ([]comment.TaskCommentWithAuthor, int, error) {
	return nil, 0, nil
}
func (r *commentStubRepo) Update(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *commentStubRepo) Delete(_ context.Context, _ uuid.UUID) error           { return nil }

// timeentryStubRepo implements timeentry.Repository for testing.
type timeentryStubRepo struct{}

func (r *timeentryStubRepo) Create(_ context.Context, _ *models.TimeEntry) error { return nil }
func (r *timeentryStubRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*models.TimeEntryWithUser, error) {
	return nil, timeentry.ErrNotFound
}
func (r *timeentryStubRepo) Update(_ context.Context, _ *models.TimeEntry) error { return nil }
func (r *timeentryStubRepo) Delete(_ context.Context, _, _ uuid.UUID) error      { return nil }
func (r *timeentryStubRepo) ListByTask(_ context.Context, _, _ uuid.UUID, _, _ int) ([]models.TimeEntryWithUser, int, error) {
	return nil, 0, nil
}
func (r *timeentryStubRepo) ListByUser(_ context.Context, _, _ uuid.UUID, _, _ int) ([]models.TimeEntryWithUser, int, error) {
	return nil, 0, nil
}
func (r *timeentryStubRepo) GetActiveTimer(_ context.Context, _, _ uuid.UUID) (*models.ActiveTimer, error) {
	return nil, nil
}
func (r *timeentryStubRepo) StopActiveTimer(_ context.Context, _, _ uuid.UUID) (*models.TimeEntry, error) {
	return nil, nil
}
func (r *timeentryStubRepo) GetTaskTimeSummary(_ context.Context, _, _ uuid.UUID) (*models.TimeEntrySummary, error) {
	return &models.TimeEntrySummary{}, nil
}

// customfieldStubRepo implements customfield.Repository for testing.
type customfieldStubRepo struct{}

func (r *customfieldStubRepo) Create(_ context.Context, _, _, _ string, _ []string, _ int) (*customfield.Definition, error) {
	return nil, nil
}
func (r *customfieldStubRepo) GetByID(_ context.Context, _, _ string) (*customfield.Definition, error) {
	return nil, customfield.ErrNotFound
}
func (r *customfieldStubRepo) List(_ context.Context, _ string) ([]*customfield.Definition, error) {
	return nil, nil
}
func (r *customfieldStubRepo) Update(_ context.Context, _, _, _, _ string, _ []string, _ int) (*customfield.Definition, error) {
	return nil, customfield.ErrNotFound
}
func (r *customfieldStubRepo) Delete(_ context.Context, _, _ string) error { return nil }

// ============================================================================
// labelMockRepo: label.Repository with call tracking
// ============================================================================

type labelMockRepo struct {
	labels     map[string]*label.Label // id → label
	taskLabels map[string][]string     // taskID → []labelID
	callCount  int                     // counts GetLabelsByTaskIDs calls
}

func newLabelMockRepo() *labelMockRepo {
	return &labelMockRepo{
		labels:     make(map[string]*label.Label),
		taskLabels: make(map[string][]string),
	}
}

func (m *labelMockRepo) Create(_ context.Context, tenantID, name, color string) (*label.Label, error) {
	l := &label.Label{ID: uuid.NewString(), TenantID: tenantID, Name: name, Color: color}
	m.labels[l.ID] = l
	return l, nil
}

func (m *labelMockRepo) GetByID(_ context.Context, tenantID, id string) (*label.Label, error) {
	l, ok := m.labels[id]
	if !ok || l.TenantID != tenantID {
		return nil, label.ErrNotFound
	}
	return l, nil
}

func (m *labelMockRepo) List(_ context.Context, tenantID string) ([]*label.Label, error) {
	var out []*label.Label
	for _, l := range m.labels {
		if l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *labelMockRepo) Update(_ context.Context, tenantID, id, name, color string) (*label.Label, error) {
	l, ok := m.labels[id]
	if !ok || l.TenantID != tenantID {
		return nil, label.ErrNotFound
	}
	l.Name = name
	l.Color = color
	return l, nil
}

func (m *labelMockRepo) Delete(_ context.Context, tenantID, id string) error {
	l, ok := m.labels[id]
	if !ok || l.TenantID != tenantID {
		return label.ErrNotFound
	}
	delete(m.labels, id)
	return nil
}

func (m *labelMockRepo) GetLabelsByTaskIDs(_ context.Context, tenantID string, taskIDs []string) (map[string][]*label.Label, error) {
	m.callCount++
	result := make(map[string][]*label.Label)
	for _, taskID := range taskIDs {
		ids := m.taskLabels[taskID]
		for _, lid := range ids {
			if l, ok := m.labels[lid]; ok && l.TenantID == tenantID {
				result[taskID] = append(result[taskID], l)
			}
		}
	}
	return result, nil
}

func (m *labelMockRepo) SetTaskLabels(_ context.Context, _, taskID string, labelIDs []string) error {
	m.taskLabels[taskID] = labelIDs
	return nil
}

func (m *labelMockRepo) GetTaskLabelIDs(_ context.Context, _, taskID string) ([]string, error) {
	return m.taskLabels[taskID], nil
}

// ============================================================================
// workTaskMockRepo: task.Repository with filter capture
// ============================================================================

type workTaskMockRepo struct {
	tasks          map[uuid.UUID]*models.TaskWithRelations
	lastListFilter task.TaskFilters
}

func newWorkTaskMockRepo() *workTaskMockRepo {
	return &workTaskMockRepo{tasks: make(map[uuid.UUID]*models.TaskWithRelations)}
}

func (r *workTaskMockRepo) Create(_ context.Context, t *models.Task) error {
	r.tasks[t.ID] = &models.TaskWithRelations{Task: *t}
	return nil
}
func (r *workTaskMockRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*models.TaskWithRelations, error) {
	tw, ok := r.tasks[id]
	if !ok {
		return nil, task.ErrNotFound
	}
	return tw, nil
}
func (r *workTaskMockRepo) List(_ context.Context, _ uuid.UUID, filters task.TaskFilters) ([]models.TaskWithRelations, int, error) {
	r.lastListFilter = filters
	var out []models.TaskWithRelations
	for _, tw := range r.tasks {
		out = append(out, *tw)
	}
	return out, len(out), nil
}
func (r *workTaskMockRepo) Update(_ context.Context, _ *models.Task) error { return nil }
func (r *workTaskMockRepo) Delete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *workTaskMockRepo) MoveTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ float64) error {
	return nil
}
func (r *workTaskMockRepo) GetNextTaskNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return 1, nil
}
func (r *workTaskMockRepo) GetSubtasks(_ context.Context, _ uuid.UUID, _ int) ([]models.TaskWithRelations, error) {
	return nil, nil
}
func (r *workTaskMockRepo) GetParentChain(_ context.Context, _ uuid.UUID) ([]models.Task, error) {
	return nil, nil
}
func (r *workTaskMockRepo) GetDepth(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (r *workTaskMockRepo) CreateDependency(_ context.Context, _ *models.TaskDependency) error {
	return nil
}
func (r *workTaskMockRepo) DeleteDependency(_ context.Context, _ uuid.UUID) error { return nil }
func (r *workTaskMockRepo) ListDependencies(_ context.Context, _ uuid.UUID) ([]models.TaskDependency, error) {
	return nil, nil
}
func (r *workTaskMockRepo) HasCycle(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (r *workTaskMockRepo) CreateActivity(_ context.Context, _ *models.TaskActivity) error {
	return nil
}
func (r *workTaskMockRepo) ListActivities(_ context.Context, _ uuid.UUID, _, _ int) ([]models.TaskActivity, int, error) {
	return nil, 0, nil
}
func (r *workTaskMockRepo) LinkEntity(_ context.Context, _ *models.TaskEntityLink) error { return nil }
func (r *workTaskMockRepo) UnlinkEntity(_ context.Context, _ uuid.UUID) error            { return nil }
func (r *workTaskMockRepo) ListEntityLinks(_ context.Context, _ uuid.UUID) ([]models.TaskEntityLink, error) {
	return nil, nil
}
func (r *workTaskMockRepo) ListTasksForEntity(_ context.Context, _ string, _ uuid.UUID) ([]models.TaskWithRelations, error) {
	return nil, nil
}
func (r *workTaskMockRepo) AttachFile(_ context.Context, _ *models.TaskFile) error { return nil }
func (r *workTaskMockRepo) RemoveFile(_ context.Context, _ uuid.UUID) error        { return nil }
func (r *workTaskMockRepo) ListFiles(_ context.Context, _ uuid.UUID) ([]models.TaskFile, error) {
	return nil, nil
}
func (r *workTaskMockRepo) SetCustomFieldValues(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ map[uuid.UUID]any) error {
	return nil
}
func (r *workTaskMockRepo) GetCustomFieldValues(_ context.Context, _ uuid.UUID) (map[string]any, error) {
	return nil, nil
}
func (r *workTaskMockRepo) Search(_ context.Context, _ uuid.UUID, _ string, _ task.TaskSearchFilters) ([]models.TaskWithRelations, int, error) {
	return nil, 0, nil
}
func (r *workTaskMockRepo) ListByProject(_ context.Context, _ uuid.UUID) ([]models.Task, error) {
	return nil, nil
}
func (r *workTaskMockRepo) ListDependenciesByProject(_ context.Context, _ uuid.UUID) ([]models.TaskDependency, error) {
	return nil, nil
}
func (r *workTaskMockRepo) GetStatusByID(_ context.Context, _ uuid.UUID) (string, bool, error) {
	return "Open", false, nil
}

// ============================================================================
// Test helpers
// ============================================================================

func newWorkLabelTestServer() (*WorkGRPCServer, *workTaskMockRepo, *labelMockRepo) {
	taskRepo := newWorkTaskMockRepo()
	labelRepo := newLabelMockRepo()

	projectSvc := project.NewService(&projectStubRepo{})
	statusSvc := wstatus.NewService(&statusStubRepo{})
	taskSvc := task.NewService(taskRepo, &projectStubRepo{})
	commentSvc := comment.NewService(&commentStubRepo{}, taskRepo)
	teSvc := timeentry.NewService(&timeentryStubRepo{})
	labelSvc := label.NewService(labelRepo)
	cfSvc := customfield.NewService(&customfieldStubRepo{})

	srv := NewWorkGRPCServer(projectSvc, statusSvc, taskSvc, taskRepo, commentSvc, teSvc, labelSvc, cfSvc)
	return srv, taskRepo, labelRepo
}

func ctxWithLabelTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

// ============================================================================
// Tests
// ============================================================================

// TestListTasks_PopulatesLabelIds verifies that ListTasks calls GetLabelsByTaskIDs
// exactly once and populates label_ids in the returned protos.
func TestListTasks_PopulatesLabelIds(t *testing.T) {
	srv, taskRepo, labelRepo := newWorkLabelTestServer()
	tenantID := uuid.New()

	// Seed a label
	lbl := &label.Label{ID: uuid.NewString(), TenantID: tenantID.String(), Name: "Bug", Color: "#ef4444"}
	labelRepo.labels[lbl.ID] = lbl

	// Seed a task
	tid := uuid.New()
	tw := &models.TaskWithRelations{Task: models.Task{
		ID: tid, TenantID: tenantID, Title: "Test task", Priority: "medium",
		CreatedBy: uuid.New(),
	}}
	taskRepo.tasks[tid] = tw

	// Associate label with task
	labelRepo.taskLabels[tid.String()] = []string{lbl.ID}

	ctx := ctxWithLabelTenant(tenantID)
	resp, err := srv.ListTasks(ctx, &workv1.ListTasksRequest{
		TenantId: tenantID.String(),
	})
	require.NoError(t, err)
	require.Len(t, resp.Tasks, 1)

	// label_ids must be populated
	assert.Equal(t, []string{lbl.ID}, resp.Tasks[0].LabelIds, "label_ids should be populated")

	// GetLabelsByTaskIDs must be called exactly once
	assert.Equal(t, 1, labelRepo.callCount, "GetLabelsByTaskIDs should be called exactly once")
}

// TestListTasks_FilterLabelIds_PassedToFilters verifies that filter_label_ids in the
// request is parsed and forwarded to TaskFilters.LabelIDs.
func TestListTasks_FilterLabelIds_PassedToFilters(t *testing.T) {
	srv, taskRepo, _ := newWorkLabelTestServer()
	tenantID := uuid.New()
	lid := uuid.New()

	ctx := ctxWithLabelTenant(tenantID)
	_, err := srv.ListTasks(ctx, &workv1.ListTasksRequest{
		TenantId:       tenantID.String(),
		FilterLabelIds: []string{lid.String()},
	})
	require.NoError(t, err)

	// Verify LabelIDs was set on the filters used by the repo
	require.Len(t, taskRepo.lastListFilter.LabelIDs, 1)
	assert.Equal(t, lid, taskRepo.lastListFilter.LabelIDs[0])
}

// TestListTasks_FilterLabelIds_InvalidUUID returns InvalidArgument.
func TestListTasks_FilterLabelIds_InvalidUUID(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	tenantID := uuid.New()
	ctx := ctxWithLabelTenant(tenantID)

	_, err := srv.ListTasks(ctx, &workv1.ListTasksRequest{
		TenantId:       tenantID.String(),
		FilterLabelIds: []string{"not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestGetTask_PopulatesLabelIds verifies GetTask calls GetLabelsByTaskIDs and
// populates label_ids in the returned proto.
func TestGetTask_PopulatesLabelIds(t *testing.T) {
	srv, taskRepo, labelRepo := newWorkLabelTestServer()
	tenantID := uuid.New()

	// Seed a label
	lbl := &label.Label{ID: uuid.NewString(), TenantID: tenantID.String(), Name: "Feature", Color: "#3b82f6"}
	labelRepo.labels[lbl.ID] = lbl

	// Seed a task
	tid := uuid.New()
	tw := &models.TaskWithRelations{Task: models.Task{
		ID: tid, TenantID: tenantID, Title: "Feature task", Priority: "low",
		CreatedBy: uuid.New(),
	}}
	taskRepo.tasks[tid] = tw
	labelRepo.taskLabels[tid.String()] = []string{lbl.ID}

	ctx := ctxWithLabelTenant(tenantID)
	resp, err := srv.GetTask(ctx, &workv1.GetTaskRequest{Id: tid.String()})
	require.NoError(t, err)
	assert.Equal(t, []string{lbl.ID}, resp.Task.LabelIds, "label_ids should be populated")

	// GetLabelsByTaskIDs must be called exactly once
	assert.Equal(t, 1, labelRepo.callCount, "GetLabelsByTaskIDs should be called exactly once")
}

// TestGetTask_NoLabels returns empty label_ids when no labels are assigned.
func TestGetTask_NoLabels(t *testing.T) {
	srv, taskRepo, _ := newWorkLabelTestServer()
	tenantID := uuid.New()

	tid := uuid.New()
	tw := &models.TaskWithRelations{Task: models.Task{
		ID: tid, TenantID: tenantID, Title: "Unlabeled", Priority: "high",
		CreatedBy: uuid.New(),
	}}
	taskRepo.tasks[tid] = tw

	ctx := ctxWithLabelTenant(tenantID)
	resp, err := srv.GetTask(ctx, &workv1.GetTaskRequest{Id: tid.String()})
	require.NoError(t, err)
	assert.Empty(t, resp.Task.LabelIds, "label_ids should be empty when no labels assigned")
}
