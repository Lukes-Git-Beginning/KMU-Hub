package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/comment"
	"github.com/kmuhub/kmuhub/internal/work/project"
	wstatus "github.com/kmuhub/kmuhub/internal/work/status"
	"github.com/kmuhub/kmuhub/internal/work/task"
	"github.com/kmuhub/kmuhub/internal/work/timeentry"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// ============================================================================
// Stub repositories — minimal implementations of the work-package Repository
// interfaces, just enough to exercise the empty-result List/Search paths
// fixed by fix-work-grpc-nil-slice-wire-shape (Block B follow-up of
// fix-crm-list-nil-slice-wire-shape). Unused methods return zero values.
// ============================================================================

type stubWorkProjectRepo struct{}

func (stubWorkProjectRepo) Create(context.Context, *models.Project) error { return nil }

func (stubWorkProjectRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Project, error) {
	return &models.Project{ID: id, TenantID: tenantID}, nil
}

func (stubWorkProjectRepo) List(context.Context, uuid.UUID, uuid.UUID, bool, bool) ([]models.ProjectWithDetails, error) {
	return nil, nil
}

func (stubWorkProjectRepo) Update(context.Context, *models.Project) error       { return nil }
func (stubWorkProjectRepo) Archive(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (stubWorkProjectRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error  { return nil }

func (stubWorkProjectRepo) GetProjectKey(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", nil
}

func (stubWorkProjectRepo) KeyExists(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}

func (stubWorkProjectRepo) AddMember(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}
func (stubWorkProjectRepo) RemoveMember(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (stubWorkProjectRepo) UpdateMemberRole(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (stubWorkProjectRepo) GetMember(context.Context, uuid.UUID, uuid.UUID) (*models.ProjectMember, error) {
	return nil, nil
}

func (stubWorkProjectRepo) ListMembers(context.Context, uuid.UUID) ([]models.ProjectMember, error) {
	return nil, nil
}

func (stubWorkProjectRepo) IsMember(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

func (stubWorkProjectRepo) CountOwners(context.Context, uuid.UUID) (int, error) { return 0, nil }

func (stubWorkProjectRepo) SaveAsTemplate(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID) (*models.Project, error) {
	return nil, nil
}

func (stubWorkProjectRepo) GetForTemplate(context.Context, uuid.UUID, uuid.UUID) (*models.Project, []models.ProjectStatus, error) {
	return nil, nil, nil
}

func (stubWorkProjectRepo) GetUserPreference(context.Context, uuid.UUID, uuid.UUID) (*models.UserProjectPreference, error) {
	return nil, nil
}

func (stubWorkProjectRepo) SetUserPreference(context.Context, *models.UserProjectPreference) error {
	return nil
}

func newWorkServerWithProjectRepo(repo project.Repository) *WorkGRPCServer {
	return &WorkGRPCServer{projectService: project.NewService(repo)}
}

type stubWorkStatusRepo struct{}

func (stubWorkStatusRepo) Create(context.Context, *models.ProjectStatus) error { return nil }

func (stubWorkStatusRepo) GetByID(context.Context, uuid.UUID) (*models.ProjectStatus, error) {
	return nil, nil
}

func (stubWorkStatusRepo) ListByProject(context.Context, uuid.UUID) ([]models.ProjectStatus, error) {
	return nil, nil
}

func (stubWorkStatusRepo) Update(context.Context, *models.ProjectStatus) error   { return nil }
func (stubWorkStatusRepo) Delete(context.Context, uuid.UUID) error               { return nil }
func (stubWorkStatusRepo) Reorder(context.Context, uuid.UUID, []uuid.UUID) error { return nil }

func (stubWorkStatusRepo) CountTasksWithStatus(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (stubWorkStatusRepo) HasDefault(context.Context, uuid.UUID) (bool, error) { return false, nil }

func (stubWorkStatusRepo) GetNextSortOrder(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (stubWorkStatusRepo) NameExistsInProject(context.Context, uuid.UUID, string, *uuid.UUID) (bool, error) {
	return false, nil
}

func (stubWorkStatusRepo) CopyStatusesForProject(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func newWorkServerWithStatusRepo(repo wstatus.Repository) *WorkGRPCServer {
	return &WorkGRPCServer{statusService: wstatus.NewService(repo)}
}

type stubWorkTaskRepo struct{}

func (stubWorkTaskRepo) Create(context.Context, *models.Task) error { return nil }

func (stubWorkTaskRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*models.TaskWithRelations, error) {
	return nil, nil
}

func (stubWorkTaskRepo) List(context.Context, uuid.UUID, task.TaskFilters) ([]models.TaskWithRelations, int, error) {
	return nil, 0, nil
}

func (stubWorkTaskRepo) Update(context.Context, *models.Task) error         { return nil }
func (stubWorkTaskRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (stubWorkTaskRepo) MoveTask(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, float64, *time.Time) error {
	return nil
}

func (stubWorkTaskRepo) GetNextTaskNumber(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	return 0, nil
}

func (stubWorkTaskRepo) GetSubtasks(context.Context, uuid.UUID, int) ([]models.TaskWithRelations, error) {
	return nil, nil
}

func (stubWorkTaskRepo) GetParentChain(context.Context, uuid.UUID) ([]models.Task, error) {
	return nil, nil
}

func (stubWorkTaskRepo) GetDepth(context.Context, uuid.UUID) (int, error) { return 0, nil }

func (stubWorkTaskRepo) CreateDependency(context.Context, *models.TaskDependency) error {
	return nil
}
func (stubWorkTaskRepo) DeleteDependency(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (stubWorkTaskRepo) ListDependencies(context.Context, uuid.UUID) ([]models.TaskDependency, error) {
	return nil, nil
}

func (stubWorkTaskRepo) HasCycle(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func (stubWorkTaskRepo) CreateActivity(context.Context, *models.TaskActivity) error { return nil }

func (stubWorkTaskRepo) ListActivities(context.Context, uuid.UUID, int, int) ([]models.TaskActivity, int, error) {
	return nil, 0, nil
}

func (stubWorkTaskRepo) LinkEntity(context.Context, *models.TaskEntityLink) error { return nil }
func (stubWorkTaskRepo) UnlinkEntity(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (stubWorkTaskRepo) ListEntityLinks(context.Context, uuid.UUID) ([]models.TaskEntityLink, error) {
	return nil, nil
}

func (stubWorkTaskRepo) ListTasksForEntity(context.Context, string, uuid.UUID) ([]models.TaskWithRelations, error) {
	return nil, nil
}

func (stubWorkTaskRepo) AttachFile(context.Context, *models.TaskFile) error     { return nil }
func (stubWorkTaskRepo) RemoveFile(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (stubWorkTaskRepo) ListFiles(context.Context, uuid.UUID) ([]models.TaskFile, error) {
	return nil, nil
}

func (stubWorkTaskRepo) SetCustomFieldValues(context.Context, uuid.UUID, uuid.UUID, map[uuid.UUID]any) error {
	return nil
}

func (stubWorkTaskRepo) GetCustomFieldValues(context.Context, uuid.UUID) (map[string]any, error) {
	return nil, nil
}

func (stubWorkTaskRepo) Search(context.Context, uuid.UUID, string, task.TaskSearchFilters) ([]models.TaskWithRelations, int, error) {
	return nil, 0, nil
}

func (stubWorkTaskRepo) ListByProject(context.Context, uuid.UUID) ([]models.Task, error) {
	return nil, nil
}

func (stubWorkTaskRepo) ListDependenciesByProject(context.Context, uuid.UUID) ([]models.TaskDependency, error) {
	return nil, nil
}

func (stubWorkTaskRepo) GetStatusByID(context.Context, uuid.UUID) (string, bool, error) {
	return "", false, nil
}

func (stubWorkTaskRepo) GetUserDisplayName(context.Context, uuid.UUID) (string, error) {
	return "", nil
}

func newWorkServerWithTaskRepo(repo task.Repository) *WorkGRPCServer {
	return &WorkGRPCServer{taskRepo: repo}
}

func newWorkServerWithTaskService(repo task.Repository) *WorkGRPCServer {
	return &WorkGRPCServer{taskService: task.NewService(repo, nil)}
}

type stubWorkCommentRepo struct{}

func (stubWorkCommentRepo) Create(context.Context, *models.TaskComment) error { return nil }

func (stubWorkCommentRepo) GetByID(context.Context, uuid.UUID) (*models.TaskComment, error) {
	return nil, nil
}

func (stubWorkCommentRepo) List(context.Context, uuid.UUID, int, int) ([]comment.TaskCommentWithAuthor, int, error) {
	return nil, 0, nil
}

func (stubWorkCommentRepo) Update(context.Context, uuid.UUID, string) error { return nil }
func (stubWorkCommentRepo) Delete(context.Context, uuid.UUID) error         { return nil }

func newWorkServerWithCommentRepo(repo comment.Repository) *WorkGRPCServer {
	return &WorkGRPCServer{commentService: comment.NewService(repo, nil)}
}

type stubWorkTimeEntryRepo struct{}

func (stubWorkTimeEntryRepo) Create(context.Context, *models.TimeEntry) error { return nil }

func (stubWorkTimeEntryRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*models.TimeEntryWithUser, error) {
	return nil, nil
}

func (stubWorkTimeEntryRepo) Update(context.Context, *models.TimeEntry) error    { return nil }
func (stubWorkTimeEntryRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (stubWorkTimeEntryRepo) ListByTask(context.Context, uuid.UUID, uuid.UUID, int, int) ([]models.TimeEntryWithUser, int, error) {
	return nil, 0, nil
}

func (stubWorkTimeEntryRepo) ListByUser(context.Context, uuid.UUID, uuid.UUID, int, int) ([]models.TimeEntryWithUser, int, error) {
	return nil, 0, nil
}

func (stubWorkTimeEntryRepo) ListBillable(context.Context, uuid.UUID) ([]models.BillableTimeEntry, error) {
	return nil, nil
}

func (stubWorkTimeEntryRepo) ListByProject(context.Context, uuid.UUID, uuid.UUID) ([]models.ProjectTimeEntry, error) {
	return nil, nil
}

func (stubWorkTimeEntryRepo) AggregateProjectHours(context.Context, uuid.UUID, uuid.UUID, string, time.Time) ([]models.UtilizationBucket, error) {
	return nil, nil
}

func (stubWorkTimeEntryRepo) GetActiveTimer(context.Context, uuid.UUID, uuid.UUID) (*models.ActiveTimer, error) {
	return nil, nil
}

func (stubWorkTimeEntryRepo) StopActiveTimer(context.Context, uuid.UUID, uuid.UUID) (*models.TimeEntry, error) {
	return nil, nil
}

func (stubWorkTimeEntryRepo) GetTaskTimeSummary(context.Context, uuid.UUID, uuid.UUID) (*models.TimeEntrySummary, error) {
	return nil, nil
}

func newWorkServerWithTimeEntryRepo(repo timeentry.Repository) *WorkGRPCServer {
	return &WorkGRPCServer{timeEntryService: timeentry.NewService(repo)}
}

// ============================================================================
// Empty-result wire-shape tests. Each one proves the fixed handler returns a
// non-nil, zero-length slice (serializes to `[]`) instead of the pre-fix nil
// (serializes to `null`).
// ============================================================================

func TestListProjects_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithProjectRepo(stubWorkProjectRepo{})
	resp, err := srv.ListProjects(ctxWithTenant(uuid.New()), &workv1.ListProjectsRequest{})
	requireGRPCOK(t, err)
	if resp.Projects == nil {
		t.Error("Projects should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(resp.Projects))
	}
}

func TestListProjectMembers_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithProjectRepo(stubWorkProjectRepo{})
	resp, err := srv.ListProjectMembers(ctxWithTenant(uuid.New()), &workv1.ListProjectMembersRequest{
		ProjectId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Members == nil {
		t.Error("Members should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Members) != 0 {
		t.Errorf("expected 0 members, got %d", len(resp.Members))
	}
}

func TestListProjectStatuses_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithStatusRepo(stubWorkStatusRepo{})
	resp, err := srv.ListProjectStatuses(ctxWithTenant(uuid.New()), &workv1.ListProjectStatusesRequest{
		ProjectId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Statuses == nil {
		t.Error("Statuses should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Statuses) != 0 {
		t.Errorf("expected 0 statuses, got %d", len(resp.Statuses))
	}
}

// TestReorderProjectStatuses_EmptyIsNilNotEmptySlice is the Randfund noted in
// the unit scope: a project with zero statuses and an empty status_ids list
// passes Reorder's per-ID validation trivially (nothing to check), so the
// handler reaches the ListByProject re-read that must come back as [].
func TestReorderProjectStatuses_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithStatusRepo(stubWorkStatusRepo{})
	resp, err := srv.ReorderProjectStatuses(ctxWithTenant(uuid.New()), &workv1.ReorderProjectStatusesRequest{
		ProjectId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Statuses == nil {
		t.Error("Statuses should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Statuses) != 0 {
		t.Errorf("expected 0 statuses, got %d", len(resp.Statuses))
	}
}

func TestListTasks_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithTaskService(stubWorkTaskRepo{})
	resp, err := srv.ListTasks(ctxWithTenant(uuid.New()), &workv1.ListTasksRequest{})
	requireGRPCOK(t, err)
	if resp.Tasks == nil {
		t.Error("Tasks should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(resp.Tasks))
	}
}

func TestListSubtasks_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithTaskRepo(stubWorkTaskRepo{})
	resp, err := srv.ListSubtasks(ctxWithTenant(uuid.New()), &workv1.ListSubtasksRequest{
		ParentTaskId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Tasks == nil {
		t.Error("Tasks should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(resp.Tasks))
	}
}

func TestListTaskDependencies_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithTaskRepo(stubWorkTaskRepo{})
	resp, err := srv.ListTaskDependencies(ctxWithTenant(uuid.New()), &workv1.ListTaskDependenciesRequest{
		TaskId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Dependencies == nil {
		t.Error("Dependencies should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(resp.Dependencies))
	}
}

func TestListTaskComments_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithCommentRepo(stubWorkCommentRepo{})
	resp, err := srv.ListTaskComments(ctxWithTenant(uuid.New()), &workv1.ListTaskCommentsRequest{
		TaskId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Comments == nil {
		t.Error("Comments should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(resp.Comments))
	}
}

func TestListTaskEntityLinks_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithTaskRepo(stubWorkTaskRepo{})
	resp, err := srv.ListTaskEntityLinks(ctxWithTenant(uuid.New()), &workv1.ListTaskEntityLinksRequest{
		TaskId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Links == nil {
		t.Error("Links should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Links) != 0 {
		t.Errorf("expected 0 links, got %d", len(resp.Links))
	}
}

func TestListEntityTasks_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithTaskRepo(stubWorkTaskRepo{})
	resp, err := srv.ListEntityTasks(ctxWithTenant(uuid.New()), &workv1.ListEntityTasksRequest{
		EntityType: "invoice",
		EntityId:   uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Tasks == nil {
		t.Error("Tasks should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(resp.Tasks))
	}
}

func TestListTaskActivities_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithTaskRepo(stubWorkTaskRepo{})
	resp, err := srv.ListTaskActivities(ctxWithTenant(uuid.New()), &workv1.ListTaskActivitiesRequest{
		TaskId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Activities == nil {
		t.Error("Activities should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Activities) != 0 {
		t.Errorf("expected 0 activities, got %d", len(resp.Activities))
	}
}

func TestListTaskFiles_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithTaskRepo(stubWorkTaskRepo{})
	resp, err := srv.ListTaskFiles(ctxWithTenant(uuid.New()), &workv1.ListTaskFilesRequest{
		TaskId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Files == nil {
		t.Error("Files should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(resp.Files))
	}
}

// TestSearchTasks_EmptyIsNilNotEmptySlice is the second Randfund noted in the
// unit scope: SearchTasks shares the same bare "var protos []*workv1.TaskProto"
// pattern as ListTasks/ListSubtasks/ListEntityTasks.
func TestSearchTasks_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithTaskRepo(stubWorkTaskRepo{})
	resp, err := srv.SearchTasks(ctxWithTenant(uuid.New()), &workv1.SearchTasksRequest{
		Query: "nothing matches",
	})
	requireGRPCOK(t, err)
	if resp.Tasks == nil {
		t.Error("Tasks should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(resp.Tasks))
	}
}

func TestListTimeEntries_EmptyIsNilNotEmptySlice(t *testing.T) {
	srv := newWorkServerWithTimeEntryRepo(stubWorkTimeEntryRepo{})
	resp, err := srv.ListTimeEntries(ctxWithTenant(uuid.New()), &workv1.ListTimeEntriesRequest{
		TaskId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Entries == nil {
		t.Error("Entries should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(resp.Entries))
	}
}
