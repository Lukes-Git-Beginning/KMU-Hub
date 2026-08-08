package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

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
// timeEntryMockRepo: timeentry.Repository backed by in-memory maps, so the
// real timeentry.Service logic (auto-stop-previous-timer, owner-only
// edit/delete, duration validation) actually executes end to end.
// ============================================================================

type timeEntryMockRepo struct {
	entries          map[uuid.UUID]models.TimeEntry
	activeByUser     map[uuid.UUID]uuid.UUID
	projectEntries   []models.ProjectTimeEntry
	utilBuckets      []models.UtilizationBucket
	lastListPage     int
	lastListPageSize int
}

func newTimeEntryMockRepo() *timeEntryMockRepo {
	return &timeEntryMockRepo{
		entries:      make(map[uuid.UUID]models.TimeEntry),
		activeByUser: make(map[uuid.UUID]uuid.UUID),
	}
}

func (r *timeEntryMockRepo) Create(_ context.Context, e *models.TimeEntry) error {
	r.entries[e.ID] = *e
	if e.EndedAt == nil {
		r.activeByUser[e.UserID] = e.ID
	}
	return nil
}

func (r *timeEntryMockRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.TimeEntryWithUser, error) {
	e, ok := r.entries[id]
	if !ok || e.TenantID != tenantID {
		return nil, timeentry.ErrNotFound
	}
	return &models.TimeEntryWithUser{TimeEntry: e, UserName: "Test User"}, nil
}

func (r *timeEntryMockRepo) Update(_ context.Context, e *models.TimeEntry) error {
	if _, ok := r.entries[e.ID]; !ok {
		return timeentry.ErrNotFound
	}
	r.entries[e.ID] = *e
	return nil
}

func (r *timeEntryMockRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	e, ok := r.entries[id]
	if !ok || e.TenantID != tenantID {
		return timeentry.ErrNotFound
	}
	delete(r.entries, id)
	if r.activeByUser[e.UserID] == id {
		delete(r.activeByUser, e.UserID)
	}
	return nil
}

func (r *timeEntryMockRepo) ListByTask(_ context.Context, taskID, tenantID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error) {
	r.lastListPage = page
	r.lastListPageSize = pageSize
	var out []models.TimeEntryWithUser
	for _, e := range r.entries {
		if e.TaskID == taskID && e.TenantID == tenantID {
			out = append(out, models.TimeEntryWithUser{TimeEntry: e, UserName: "Test User"})
		}
	}
	return out, len(out), nil
}

func (r *timeEntryMockRepo) ListByUser(_ context.Context, userID, tenantID uuid.UUID, _, _ int) ([]models.TimeEntryWithUser, int, error) {
	var out []models.TimeEntryWithUser
	for _, e := range r.entries {
		if e.UserID == userID && e.TenantID == tenantID {
			out = append(out, models.TimeEntryWithUser{TimeEntry: e, UserName: "Test User"})
		}
	}
	return out, len(out), nil
}

func (r *timeEntryMockRepo) ListBillable(_ context.Context, tenantID uuid.UUID) ([]models.BillableTimeEntry, error) {
	var out []models.BillableTimeEntry
	for _, e := range r.entries {
		if e.TenantID == tenantID && e.EndedAt != nil {
			out = append(out, models.BillableTimeEntry{TimeEntry: e, UserName: "Test User", TaskTitle: "Task", ProjectName: "Project"})
		}
	}
	return out, nil
}

func (r *timeEntryMockRepo) ListByProject(_ context.Context, _, _ uuid.UUID) ([]models.ProjectTimeEntry, error) {
	return r.projectEntries, nil
}

func (r *timeEntryMockRepo) AggregateProjectHours(_ context.Context, _, _ uuid.UUID, _ string, _ time.Time) ([]models.UtilizationBucket, error) {
	return r.utilBuckets, nil
}

func (r *timeEntryMockRepo) GetActiveTimer(_ context.Context, userID, tenantID uuid.UUID) (*models.ActiveTimer, error) {
	id, ok := r.activeByUser[userID]
	if !ok {
		return nil, nil
	}
	e, ok := r.entries[id]
	if !ok || e.TenantID != tenantID {
		return nil, nil
	}
	return &models.ActiveTimer{TimeEntry: e, TaskTitle: "Active Task"}, nil
}

func (r *timeEntryMockRepo) StopActiveTimer(_ context.Context, userID, tenantID uuid.UUID) (*models.TimeEntry, error) {
	id, ok := r.activeByUser[userID]
	if !ok {
		return nil, nil
	}
	e, ok := r.entries[id]
	if !ok || e.TenantID != tenantID {
		return nil, nil
	}
	now := time.Now()
	e.EndedAt = &now
	dur := int(now.Sub(e.StartedAt).Seconds())
	e.DurationSeconds = &dur
	r.entries[id] = e
	delete(r.activeByUser, userID)
	return &e, nil
}

func (r *timeEntryMockRepo) GetTaskTimeSummary(_ context.Context, taskID, tenantID uuid.UUID) (*models.TimeEntrySummary, error) {
	summary := &models.TimeEntrySummary{TaskID: taskID}
	for _, e := range r.entries {
		if e.TaskID == taskID && e.TenantID == tenantID {
			summary.EntryCount++
			if e.DurationSeconds != nil {
				summary.TotalDurationSeconds += *e.DurationSeconds
			}
		}
	}
	return summary, nil
}

// customFieldMockRepo: customfield.Repository backed by an in-memory map, so
// tenant scoping and not-found behavior actually execute (customfieldStubRepo
// in work_label_test.go is a pure no-op stub, only usable for validation
// paths that never reach the repo).
type customFieldMockRepo struct {
	defs map[string]*customfield.Definition
}

func newCustomFieldMockRepo() *customFieldMockRepo {
	return &customFieldMockRepo{defs: make(map[string]*customfield.Definition)}
}

func (r *customFieldMockRepo) Create(_ context.Context, tenantID, name, fieldType string, options []string, position int) (*customfield.Definition, error) {
	d := &customfield.Definition{ID: uuid.NewString(), TenantID: tenantID, Name: name, FieldType: fieldType, Options: options, Position: position}
	r.defs[d.ID] = d
	return d, nil
}

func (r *customFieldMockRepo) GetByID(_ context.Context, tenantID, id string) (*customfield.Definition, error) {
	d, ok := r.defs[id]
	if !ok || d.TenantID != tenantID {
		return nil, customfield.ErrNotFound
	}
	return d, nil
}

func (r *customFieldMockRepo) List(_ context.Context, tenantID string) ([]*customfield.Definition, error) {
	var out []*customfield.Definition
	for _, d := range r.defs {
		if d.TenantID == tenantID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (r *customFieldMockRepo) Update(_ context.Context, tenantID, id, name, fieldType string, options []string, position int) (*customfield.Definition, error) {
	d, ok := r.defs[id]
	if !ok || d.TenantID != tenantID {
		return nil, customfield.ErrNotFound
	}
	d.Name = name
	d.FieldType = fieldType
	d.Options = options
	d.Position = position
	return d, nil
}

func (r *customFieldMockRepo) Delete(_ context.Context, tenantID, id string) error {
	d, ok := r.defs[id]
	if !ok || d.TenantID != tenantID {
		return customfield.ErrNotFound
	}
	delete(r.defs, id)
	return nil
}

// ============================================================================
// Test server builders
// ============================================================================

func newWorkTimeEntryTestServer() (*WorkGRPCServer, *timeEntryMockRepo, *projectMockRepo) {
	taskRepo := newWorkTaskMockRepo()
	projRepo := newProjectMockRepo()
	projectSvc := project.NewService(projRepo)
	statusSvc := wstatus.NewService(&statusStubRepo{})
	taskSvc := task.NewService(taskRepo, projRepo)
	commentSvc := comment.NewService(&commentStubRepo{}, taskRepo)
	teRepo := newTimeEntryMockRepo()
	teSvc := timeentry.NewService(teRepo)
	labelSvc := label.NewService(newLabelMockRepo())
	cfSvc := customfield.NewService(&customfieldStubRepo{})

	srv := NewWorkGRPCServer(projectSvc, statusSvc, taskSvc, taskRepo, commentSvc, teSvc, labelSvc, cfSvc)
	return srv, teRepo, projRepo
}

func newWorkCustomFieldTestServer() (*WorkGRPCServer, *customFieldMockRepo) {
	taskRepo := newWorkTaskMockRepo()
	projectSvc := project.NewService(&projectStubRepo{})
	statusSvc := wstatus.NewService(&statusStubRepo{})
	taskSvc := task.NewService(taskRepo, &projectStubRepo{})
	commentSvc := comment.NewService(&commentStubRepo{}, taskRepo)
	teSvc := timeentry.NewService(&timeentryStubRepo{})
	labelSvc := label.NewService(newLabelMockRepo())
	cfRepo := newCustomFieldMockRepo()
	cfSvc := customfield.NewService(cfRepo)

	srv := NewWorkGRPCServer(projectSvc, statusSvc, taskSvc, taskRepo, commentSvc, teSvc, labelSvc, cfSvc)
	return srv, cfRepo
}

func newWorkCommentCRUDTestServer() (*WorkGRPCServer, *commentAuthzMockRepo, *workTaskMockRepo) {
	taskRepo := newWorkTaskMockRepo()
	repo := newCommentAuthzMockRepo()
	projectSvc := project.NewService(&projectStubRepo{})
	statusSvc := wstatus.NewService(&statusStubRepo{})
	taskSvc := task.NewService(taskRepo, &projectStubRepo{})
	commentSvc := comment.NewService(repo, taskRepo)
	teSvc := timeentry.NewService(&timeentryStubRepo{})
	labelSvc := label.NewService(newLabelMockRepo())
	cfSvc := customfield.NewService(&customfieldStubRepo{})

	srv := NewWorkGRPCServer(projectSvc, statusSvc, taskSvc, taskRepo, commentSvc, teSvc, labelSvc, cfSvc)
	return srv, repo, taskRepo
}

func seedWorkTask(repo *workTaskMockRepo, tenantID uuid.UUID) uuid.UUID {
	tid := uuid.New()
	repo.tasks[tid] = &models.TaskWithRelations{Task: models.Task{
		ID: tid, TenantID: tenantID, Title: "Seeded task", Priority: "normal",
		CreatedBy: uuid.New(),
	}}
	return tid
}

// ============================================================================
// CreateTaskComment / ListTaskComments
// ============================================================================

func TestCreateTaskComment_Success(t *testing.T) {
	srv, _, taskRepo := newWorkCommentCRUDTestServer()
	tenantID := uuid.New()
	taskID := seedWorkTask(taskRepo, tenantID)
	authorID := uuid.New()

	ctx := ctxWithWorkTenant(tenantID)
	resp, err := srv.CreateTaskComment(ctx, &workv1.CreateTaskCommentRequest{
		TaskId:   taskID.String(),
		AuthorId: authorID.String(),
		Content:  "Hello task",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Comment)
	assert.Equal(t, "Hello task", resp.Comment.Content)
	assert.Equal(t, authorID.String(), resp.Comment.AuthorId)
}

func TestCreateTaskComment_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateTaskComment(context.Background(), &workv1.CreateTaskCommentRequest{
		TaskId:   uuid.NewString(),
		AuthorId: uuid.NewString(),
		Content:  "x",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateTaskComment_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.CreateTaskComment(ctx, &workv1.CreateTaskCommentRequest{
		TaskId:   "not-a-uuid",
		AuthorId: uuid.NewString(),
		Content:  "x",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTaskComment_InvalidAuthorID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.CreateTaskComment(ctx, &workv1.CreateTaskCommentRequest{
		TaskId:   uuid.NewString(),
		AuthorId: "not-a-uuid",
		Content:  "x",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTaskComment_InvalidQuotedCommentID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	bad := "not-a-uuid"
	_, err := srv.CreateTaskComment(ctx, &workv1.CreateTaskCommentRequest{
		TaskId:          uuid.NewString(),
		AuthorId:        uuid.NewString(),
		Content:         "x",
		QuotedCommentId: &bad,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTaskComment_TaskNotFound(t *testing.T) {
	srv, _, _ := newWorkCommentCRUDTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.CreateTaskComment(ctx, &workv1.CreateTaskCommentRequest{
		TaskId:   uuid.NewString(),
		AuthorId: uuid.NewString(),
		Content:  "x",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreateTaskComment_ContentRequired(t *testing.T) {
	srv, _, taskRepo := newWorkCommentCRUDTestServer()
	tenantID := uuid.New()
	taskID := seedWorkTask(taskRepo, tenantID)

	ctx := ctxWithWorkTenant(tenantID)
	_, err := srv.CreateTaskComment(ctx, &workv1.CreateTaskCommentRequest{
		TaskId:   taskID.String(),
		AuthorId: uuid.NewString(),
		Content:  "   ",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTaskComment_QuotedCommentNotFound(t *testing.T) {
	srv, _, taskRepo := newWorkCommentCRUDTestServer()
	tenantID := uuid.New()
	taskID := seedWorkTask(taskRepo, tenantID)
	quoted := uuid.NewString()

	ctx := ctxWithWorkTenant(tenantID)
	_, err := srv.CreateTaskComment(ctx, &workv1.CreateTaskCommentRequest{
		TaskId:          taskID.String(),
		AuthorId:        uuid.NewString(),
		Content:         "x",
		QuotedCommentId: &quoted,
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreateTaskComment_QuotedCommentWrongTask(t *testing.T) {
	srv, repo, taskRepo := newWorkCommentCRUDTestServer()
	tenantID := uuid.New()
	taskID := seedWorkTask(taskRepo, tenantID)
	otherTaskID := seedWorkTask(taskRepo, tenantID)

	quotedID := uuid.New()
	repo.comments[quotedID] = &models.TaskComment{ID: quotedID, TaskID: otherTaskID, AuthorID: uuid.New(), Content: "quoted"}

	ctx := ctxWithWorkTenant(tenantID)
	quotedStr := quotedID.String()
	_, err := srv.CreateTaskComment(ctx, &workv1.CreateTaskCommentRequest{
		TaskId:          taskID.String(),
		AuthorId:        uuid.NewString(),
		Content:         "x",
		QuotedCommentId: &quotedStr,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListTaskComments_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListTaskComments(context.Background(), &workv1.ListTaskCommentsRequest{TaskId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListTaskComments_DefaultsPageAndPageSize(t *testing.T) {
	srv, repo, taskRepo := newWorkCommentCRUDTestServer()
	tenantID := uuid.New()
	taskID := seedWorkTask(taskRepo, tenantID)
	repo.comments[uuid.New()] = &models.TaskComment{ID: uuid.New(), TaskID: taskID, AuthorID: uuid.New(), Content: "c1"}

	_, err := srv.ListTaskComments(context.Background(), &workv1.ListTaskCommentsRequest{
		TaskId: taskID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.lastPage)
	assert.Equal(t, 20, repo.lastPageSize)
}

func TestListTaskComments_Success(t *testing.T) {
	srv, repo, taskRepo := newWorkCommentCRUDTestServer()
	tenantID := uuid.New()
	taskID := seedWorkTask(taskRepo, tenantID)
	c1 := uuid.New()
	c2 := uuid.New()
	repo.comments[c1] = &models.TaskComment{ID: c1, TaskID: taskID, AuthorID: uuid.New(), Content: "c1"}
	repo.comments[c2] = &models.TaskComment{ID: c2, TaskID: taskID, AuthorID: uuid.New(), Content: "c2"}
	// A comment on a different task must not leak into this list.
	repo.comments[uuid.New()] = &models.TaskComment{ID: uuid.New(), TaskID: uuid.New(), AuthorID: uuid.New(), Content: "other task"}

	resp, err := srv.ListTaskComments(context.Background(), &workv1.ListTaskCommentsRequest{
		TaskId: taskID.String(), Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), resp.Total)
	assert.Len(t, resp.Comments, 2)
}

// ============================================================================
// Timer lifecycle: StartTimer / StopTimer / GetActiveTimer
// ============================================================================

func TestStartTimer_Success(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	taskID := uuid.New()
	userID := uuid.New()

	ctx := ctxWithWorkTenant(tenantID)
	resp, err := srv.StartTimer(ctx, &workv1.StartTimerRequest{TaskId: taskID.String(), UserId: userID.String()})
	require.NoError(t, err)
	require.NotNil(t, resp.Entry)
	assert.Nil(t, resp.StoppedEntry)
	assert.Equal(t, taskID.String(), resp.Entry.TaskId)
	assert.Contains(t, repo.activeByUser, userID)
}

func TestStartTimer_AutoStopsExistingTimer(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	userID := uuid.New()
	oldTaskID := uuid.New()

	// Seed a running timer for the user on a different task.
	oldEntry := models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: oldTaskID, UserID: userID, StartedAt: time.Now().Add(-time.Hour)}
	repo.entries[oldEntry.ID] = oldEntry
	repo.activeByUser[userID] = oldEntry.ID

	ctx := ctxWithWorkTenant(tenantID)
	newTaskID := uuid.New()
	resp, err := srv.StartTimer(ctx, &workv1.StartTimerRequest{TaskId: newTaskID.String(), UserId: userID.String()})
	require.NoError(t, err)
	require.NotNil(t, resp.StoppedEntry)
	assert.Equal(t, oldEntry.ID.String(), resp.StoppedEntry.Id)
	assert.NotNil(t, resp.StoppedEntry.DurationSeconds)
	// The new timer, not the old one, is now active.
	assert.Equal(t, resp.Entry.Id, repo.activeByUser[userID].String())
}

func TestStartTimer_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.StartTimer(context.Background(), &workv1.StartTimerRequest{TaskId: uuid.NewString(), UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestStartTimer_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.StartTimer(ctx, &workv1.StartTimerRequest{TaskId: "bad", UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestStartTimer_InvalidUserID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.StartTimer(ctx, &workv1.StartTimerRequest{TaskId: uuid.NewString(), UserId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestStopTimer_Success(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	userID := uuid.New()
	entry := models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: uuid.New(), UserID: userID, StartedAt: time.Now().Add(-30 * time.Minute)}
	repo.entries[entry.ID] = entry
	repo.activeByUser[userID] = entry.ID

	ctx := ctxWithWorkTenant(tenantID)
	resp, err := srv.StopTimer(ctx, &workv1.StopTimerRequest{UserId: userID.String()})
	require.NoError(t, err)
	require.NotNil(t, resp.Entry.DurationSeconds)
	assert.Greater(t, *resp.Entry.DurationSeconds, int32(0))
	assert.NotContains(t, repo.activeByUser, userID)
}

func TestStopTimer_NoActiveTimer(t *testing.T) {
	srv, _, _ := newWorkTimeEntryTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.StopTimer(ctx, &workv1.StopTimerRequest{UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestStopTimer_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.StopTimer(context.Background(), &workv1.StopTimerRequest{UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestStopTimer_InvalidUserID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.StopTimer(ctx, &workv1.StopTimerRequest{UserId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetActiveTimer_ReturnsRunningTimer(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	userID := uuid.New()
	entry := models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: uuid.New(), UserID: userID, StartedAt: time.Now()}
	repo.entries[entry.ID] = entry
	repo.activeByUser[userID] = entry.ID

	ctx := ctxWithWorkTenant(tenantID)
	resp, err := srv.GetActiveTimer(ctx, &workv1.GetActiveTimerRequest{UserId: userID.String()})
	require.NoError(t, err)
	require.NotNil(t, resp.Timer)
	assert.Equal(t, entry.ID.String(), resp.Timer.Id)
}

func TestGetActiveTimer_NilWhenNone(t *testing.T) {
	srv, _, _ := newWorkTimeEntryTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	resp, err := srv.GetActiveTimer(ctx, &workv1.GetActiveTimerRequest{UserId: uuid.NewString()})
	require.NoError(t, err)
	assert.Nil(t, resp.Timer)
}

func TestGetActiveTimer_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.GetActiveTimer(context.Background(), &workv1.GetActiveTimerRequest{UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// ============================================================================
// AddManualTimeEntry / UpdateTimeEntry / DeleteTimeEntry
// ============================================================================

func TestAddManualTimeEntry_Success(t *testing.T) {
	srv, _, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	ctx := ctxWithWorkTenant(tenantID)

	resp, err := srv.AddManualTimeEntry(ctx, &workv1.AddManualTimeEntryRequest{
		TaskId:          uuid.NewString(),
		UserId:          uuid.NewString(),
		StartedAt:       timestamppb.New(time.Now().Add(-time.Hour)),
		DurationSeconds: 3600,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Entry.DurationSeconds)
	assert.Equal(t, int32(3600), *resp.Entry.DurationSeconds)
	assert.True(t, resp.Entry.IsManual)
}

func TestAddManualTimeEntry_InvalidDuration(t *testing.T) {
	srv, _, _ := newWorkTimeEntryTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.AddManualTimeEntry(ctx, &workv1.AddManualTimeEntryRequest{
		TaskId: uuid.NewString(), UserId: uuid.NewString(),
		StartedAt: timestamppb.New(time.Now()), DurationSeconds: 0,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAddManualTimeEntry_DescriptionTooLong(t *testing.T) {
	srv, _, _ := newWorkTimeEntryTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	desc := make([]byte, 2001)
	for i := range desc {
		desc[i] = 'a'
	}
	descStr := string(desc)
	_, err := srv.AddManualTimeEntry(ctx, &workv1.AddManualTimeEntryRequest{
		TaskId: uuid.NewString(), UserId: uuid.NewString(),
		StartedAt: timestamppb.New(time.Now()), DurationSeconds: 60,
		Description: &descStr,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAddManualTimeEntry_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.AddManualTimeEntry(context.Background(), &workv1.AddManualTimeEntryRequest{
		TaskId: uuid.NewString(), UserId: uuid.NewString(), StartedAt: timestamppb.New(time.Now()), DurationSeconds: 60,
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestAddManualTimeEntry_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.AddManualTimeEntry(ctx, &workv1.AddManualTimeEntryRequest{
		TaskId: "bad", UserId: uuid.NewString(), StartedAt: timestamppb.New(time.Now()), DurationSeconds: 60,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateTimeEntry_Success(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	userID := uuid.New()
	entry := models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: uuid.New(), UserID: userID, StartedAt: time.Now()}
	repo.entries[entry.ID] = entry

	ctx := ctxWithWorkTenant(tenantID)
	newDesc := "updated"
	dur := int32(120)
	resp, err := srv.UpdateTimeEntry(ctx, &workv1.UpdateTimeEntryRequest{
		Id: entry.ID.String(), UserId: userID.String(), Description: &newDesc, DurationSeconds: &dur,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Entry.Description)
	assert.Equal(t, "updated", *resp.Entry.Description)
	assert.Equal(t, int32(120), *resp.Entry.DurationSeconds)
}

func TestUpdateTimeEntry_CannotEditOthers(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	entry := models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: uuid.New(), UserID: uuid.New(), StartedAt: time.Now()}
	repo.entries[entry.ID] = entry

	ctx := ctxWithWorkTenant(tenantID)
	_, err := srv.UpdateTimeEntry(ctx, &workv1.UpdateTimeEntryRequest{Id: entry.ID.String(), UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestUpdateTimeEntry_NotFound(t *testing.T) {
	srv, _, _ := newWorkTimeEntryTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.UpdateTimeEntry(ctx, &workv1.UpdateTimeEntryRequest{Id: uuid.NewString(), UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateTimeEntry_InvalidDuration(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	userID := uuid.New()
	entry := models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: uuid.New(), UserID: userID, StartedAt: time.Now()}
	repo.entries[entry.ID] = entry

	ctx := ctxWithWorkTenant(tenantID)
	dur := int32(-5)
	_, err := srv.UpdateTimeEntry(ctx, &workv1.UpdateTimeEntryRequest{Id: entry.ID.String(), UserId: userID.String(), DurationSeconds: &dur})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateTimeEntry_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UpdateTimeEntry(context.Background(), &workv1.UpdateTimeEntryRequest{Id: uuid.NewString(), UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateTimeEntry_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.UpdateTimeEntry(ctx, &workv1.UpdateTimeEntryRequest{Id: "bad", UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteTimeEntry_Success(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	userID := uuid.New()
	entry := models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: uuid.New(), UserID: userID, StartedAt: time.Now()}
	repo.entries[entry.ID] = entry

	ctx := ctxWithWorkTenant(tenantID)
	_, err := srv.DeleteTimeEntry(ctx, &workv1.DeleteTimeEntryRequest{Id: entry.ID.String(), UserId: userID.String()})
	require.NoError(t, err)
	_, ok := repo.entries[entry.ID]
	assert.False(t, ok)
}

func TestDeleteTimeEntry_CannotDeleteOthers(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	entry := models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: uuid.New(), UserID: uuid.New(), StartedAt: time.Now()}
	repo.entries[entry.ID] = entry

	ctx := ctxWithWorkTenant(tenantID)
	_, err := srv.DeleteTimeEntry(ctx, &workv1.DeleteTimeEntryRequest{Id: entry.ID.String(), UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.PermissionDenied)
	_, ok := repo.entries[entry.ID]
	assert.True(t, ok, "entry must still exist")
}

func TestDeleteTimeEntry_NotFound(t *testing.T) {
	srv, _, _ := newWorkTimeEntryTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.DeleteTimeEntry(ctx, &workv1.DeleteTimeEntryRequest{Id: uuid.NewString(), UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestDeleteTimeEntry_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.DeleteTimeEntry(context.Background(), &workv1.DeleteTimeEntryRequest{Id: uuid.NewString(), UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// ============================================================================
// ListTimeEntries / GetTaskTimeSummary / ListBillableTimeEntries /
// ListProjectTimeEntries / ListProjectTeamUtilization
// ============================================================================

func TestListTimeEntries_DefaultsPageAndPageSize(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	ctx := ctxWithWorkTenant(tenantID)

	_, err := srv.ListTimeEntries(ctx, &workv1.ListTimeEntriesRequest{TaskId: uuid.NewString()})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.lastListPage)
	assert.Equal(t, 20, repo.lastListPageSize)
}

func TestListTimeEntries_Success(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	taskID := uuid.New()
	repo.entries[uuid.New()] = models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: taskID, UserID: uuid.New(), StartedAt: time.Now()}

	ctx := ctxWithWorkTenant(tenantID)
	resp, err := srv.ListTimeEntries(ctx, &workv1.ListTimeEntriesRequest{TaskId: taskID.String(), Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Total)
	assert.Len(t, resp.Entries, 1)
}

func TestListTimeEntries_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.ListTimeEntries(ctx, &workv1.ListTimeEntriesRequest{TaskId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListTimeEntries_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListTimeEntries(context.Background(), &workv1.ListTimeEntriesRequest{TaskId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetTaskTimeSummary_Success(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	taskID := uuid.New()
	dur := 1800
	repo.entries[uuid.New()] = models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: taskID, UserID: uuid.New(), StartedAt: time.Now(), DurationSeconds: &dur}

	ctx := ctxWithWorkTenant(tenantID)
	resp, err := srv.GetTaskTimeSummary(ctx, &workv1.GetTaskTimeSummaryRequest{TaskId: taskID.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(1800), resp.Summary.TotalDurationSeconds)
	assert.Equal(t, int32(1), resp.Summary.EntryCount)
}

func TestGetTaskTimeSummary_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.GetTaskTimeSummary(ctx, &workv1.GetTaskTimeSummaryRequest{TaskId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListBillableTimeEntries_Success(t *testing.T) {
	srv, repo, _ := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	dur := 3600
	ended := time.Now()
	repo.entries[uuid.New()] = models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: uuid.New(), UserID: uuid.New(), StartedAt: ended.Add(-time.Hour), EndedAt: &ended, DurationSeconds: &dur}
	// A running (not-yet-billable) entry for the same tenant must not appear.
	repo.entries[uuid.New()] = models.TimeEntry{ID: uuid.New(), TenantID: tenantID, TaskID: uuid.New(), UserID: uuid.New(), StartedAt: time.Now()}

	ctx := ctxWithWorkTenant(tenantID)
	resp, err := srv.ListBillableTimeEntries(ctx, &workv1.ListBillableTimeEntriesRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Entries, 1)
}

func TestListBillableTimeEntries_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListBillableTimeEntries(context.Background(), &workv1.ListBillableTimeEntriesRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListProjectTimeEntries_Success(t *testing.T) {
	srv, repo, projRepo := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "PRJ")
	repo.projectEntries = []models.ProjectTimeEntry{{TimeEntry: models.TimeEntry{ID: uuid.New()}, UserName: "A", TaskTitle: "T"}}

	ctx := ctxWithWorkTenant(tenantID)
	resp, err := srv.ListProjectTimeEntries(ctx, &workv1.ListProjectTimeEntriesRequest{ProjectId: p.ID.String()})
	require.NoError(t, err)
	assert.Len(t, resp.Entries, 1)
}

func TestListProjectTimeEntries_ProjectNotFound(t *testing.T) {
	srv, _, _ := newWorkTimeEntryTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.ListProjectTimeEntries(ctx, &workv1.ListProjectTimeEntriesRequest{ProjectId: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestListProjectTimeEntries_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.ListProjectTimeEntries(ctx, &workv1.ListProjectTimeEntriesRequest{ProjectId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListProjectTimeEntries_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListProjectTimeEntries(context.Background(), &workv1.ListProjectTimeEntriesRequest{ProjectId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListProjectTeamUtilization_Success(t *testing.T) {
	srv, _, projRepo := newWorkTimeEntryTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "UTIL")

	ctx := ctxWithWorkTenant(tenantID)
	resp, err := srv.ListProjectTeamUtilization(ctx, &workv1.ListProjectTeamUtilizationRequest{ProjectId: p.ID.String()})
	require.NoError(t, err)
	// projRepo.ListMembers on an untouched project returns none; the handler
	// must still succeed with an empty team rather than erroring.
	assert.Empty(t, resp.Team)
}

func TestListProjectTeamUtilization_ProjectNotFound(t *testing.T) {
	srv, _, _ := newWorkTimeEntryTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.ListProjectTeamUtilization(ctx, &workv1.ListProjectTeamUtilizationRequest{ProjectId: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestListProjectTeamUtilization_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	ctx := ctxWithWorkTenant(uuid.New())
	_, err := srv.ListProjectTeamUtilization(ctx, &workv1.ListProjectTeamUtilizationRequest{ProjectId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Labels: Create/Get/List/Update/Delete/SetTaskLabels
// ============================================================================

func TestCreateLabel_Success(t *testing.T) {
	srv, _, labelRepo := newWorkLabelTestServer()
	tenantID := uuid.New()
	ctx := ctxWithLabelTenant(tenantID)

	resp, err := srv.CreateLabel(ctx, &workv1.CreateLabelRequest{Name: "Bug"})
	require.NoError(t, err)
	assert.Equal(t, "#6b7280", resp.Label.Color, "empty color must default")
	assert.Contains(t, labelRepo.labels, resp.Label.Id)
}

func TestCreateLabel_InvalidColor(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.CreateLabel(ctx, &workv1.CreateLabelRequest{Name: "Bug", Color: "not-a-color"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateLabel_NameRequired(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.CreateLabel(ctx, &workv1.CreateLabelRequest{Name: "  "})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateLabel_MissingTenant(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	_, err := srv.CreateLabel(context.Background(), &workv1.CreateLabelRequest{Name: "Bug"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetLabel_NotFound(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.GetLabel(ctx, &workv1.GetLabelRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetLabel_MissingTenant(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	_, err := srv.GetLabel(context.Background(), &workv1.GetLabelRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListLabels_TenantScoped(t *testing.T) {
	srv, _, labelRepo := newWorkLabelTestServer()
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	labelRepo.labels["mine"] = &label.Label{ID: "mine", TenantID: tenantID.String(), Name: "Mine"}
	labelRepo.labels["theirs"] = &label.Label{ID: "theirs", TenantID: otherTenantID.String(), Name: "Theirs"}

	ctx := ctxWithLabelTenant(tenantID)
	resp, err := srv.ListLabels(ctx, &workv1.ListLabelsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Labels, 1)
	assert.Equal(t, "Mine", resp.Labels[0].Name)
}

func TestListLabels_MissingTenant(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	_, err := srv.ListLabels(context.Background(), &workv1.ListLabelsRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateLabel_Success(t *testing.T) {
	srv, _, labelRepo := newWorkLabelTestServer()
	tenantID := uuid.New()
	labelRepo.labels["l1"] = &label.Label{ID: "l1", TenantID: tenantID.String(), Name: "Old", Color: "#111111"}

	ctx := ctxWithLabelTenant(tenantID)
	resp, err := srv.UpdateLabel(ctx, &workv1.UpdateLabelRequest{Id: "l1", Name: "New", Color: "#222222"})
	require.NoError(t, err)
	assert.Equal(t, "New", resp.Label.Name)
	assert.Equal(t, "#222222", resp.Label.Color)
}

func TestUpdateLabel_NotFound(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.UpdateLabel(ctx, &workv1.UpdateLabelRequest{Id: "missing", Name: "New", Color: "#222222"})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateLabel_InvalidColor(t *testing.T) {
	srv, _, labelRepo := newWorkLabelTestServer()
	tenantID := uuid.New()
	labelRepo.labels["l1"] = &label.Label{ID: "l1", TenantID: tenantID.String(), Name: "Old"}
	ctx := ctxWithLabelTenant(tenantID)
	_, err := srv.UpdateLabel(ctx, &workv1.UpdateLabelRequest{Id: "l1", Name: "New", Color: "not-a-color"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteLabel_Success(t *testing.T) {
	srv, _, labelRepo := newWorkLabelTestServer()
	tenantID := uuid.New()
	labelRepo.labels["l1"] = &label.Label{ID: "l1", TenantID: tenantID.String(), Name: "Old"}
	ctx := ctxWithLabelTenant(tenantID)
	_, err := srv.DeleteLabel(ctx, &workv1.DeleteLabelRequest{Id: "l1"})
	require.NoError(t, err)
	assert.NotContains(t, labelRepo.labels, "l1")
}

func TestDeleteLabel_NotFound(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.DeleteLabel(ctx, &workv1.DeleteLabelRequest{Id: "missing"})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSetTaskLabels_Success(t *testing.T) {
	srv, _, labelRepo := newWorkLabelTestServer()
	tenantID := uuid.New()
	l1 := uuid.NewString()
	ctx := ctxWithLabelTenant(tenantID)
	_, err := srv.SetTaskLabels(ctx, &workv1.SetTaskLabelsRequest{TaskId: "task-1", LabelIds: []string{l1}})
	require.NoError(t, err)
	assert.Equal(t, []string{l1}, labelRepo.taskLabels["task-1"])
}

func TestSetTaskLabels_BlankLabelIDRejected(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.SetTaskLabels(ctx, &workv1.SetTaskLabelsRequest{TaskId: "task-1", LabelIds: []string{"  "}})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSetTaskLabels_MissingTenant(t *testing.T) {
	srv, _, _ := newWorkLabelTestServer()
	_, err := srv.SetTaskLabels(context.Background(), &workv1.SetTaskLabelsRequest{TaskId: "task-1"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// ============================================================================
// Custom Field Definitions: Create/Get/List/Update/Delete
// ============================================================================

func TestCreateCustomFieldDefinition_Success(t *testing.T) {
	srv, cfRepo := newWorkCustomFieldTestServer()
	tenantID := uuid.New()
	ctx := ctxWithLabelTenant(tenantID)

	resp, err := srv.CreateCustomFieldDefinition(ctx, &workv1.CreateCustomFieldDefinitionRequest{
		Name: "Budget", FieldType: "number", Position: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, "Budget", resp.Definition.Name)
	assert.Contains(t, cfRepo.defs, resp.Definition.Id)
}

func TestCreateCustomFieldDefinition_NameRequired(t *testing.T) {
	srv, _ := newWorkCustomFieldTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.CreateCustomFieldDefinition(ctx, &workv1.CreateCustomFieldDefinitionRequest{Name: "  ", FieldType: "text"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateCustomFieldDefinition_InvalidFieldType(t *testing.T) {
	srv, _ := newWorkCustomFieldTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.CreateCustomFieldDefinition(ctx, &workv1.CreateCustomFieldDefinitionRequest{Name: "Budget", FieldType: "not-a-type"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateCustomFieldDefinition_MissingTenant(t *testing.T) {
	srv, _ := newWorkCustomFieldTestServer()
	_, err := srv.CreateCustomFieldDefinition(context.Background(), &workv1.CreateCustomFieldDefinitionRequest{Name: "Budget", FieldType: "text"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetCustomFieldDefinition_NotFound(t *testing.T) {
	srv, _ := newWorkCustomFieldTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.GetCustomFieldDefinition(ctx, &workv1.GetCustomFieldDefinitionRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetCustomFieldDefinition_MissingTenant(t *testing.T) {
	srv, _ := newWorkCustomFieldTestServer()
	_, err := srv.GetCustomFieldDefinition(context.Background(), &workv1.GetCustomFieldDefinitionRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListCustomFieldDefinitions_TenantScoped(t *testing.T) {
	srv, cfRepo := newWorkCustomFieldTestServer()
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	cfRepo.defs["mine"] = &customfield.Definition{ID: "mine", TenantID: tenantID.String(), Name: "Mine", FieldType: "text"}
	cfRepo.defs["theirs"] = &customfield.Definition{ID: "theirs", TenantID: otherTenantID.String(), Name: "Theirs", FieldType: "text"}

	ctx := ctxWithLabelTenant(tenantID)
	resp, err := srv.ListCustomFieldDefinitions(ctx, &workv1.ListCustomFieldDefinitionsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Definitions, 1)
	assert.Equal(t, "Mine", resp.Definitions[0].Name)
}

func TestListCustomFieldDefinitions_MissingTenant(t *testing.T) {
	srv, _ := newWorkCustomFieldTestServer()
	_, err := srv.ListCustomFieldDefinitions(context.Background(), &workv1.ListCustomFieldDefinitionsRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateCustomFieldDefinition_Success(t *testing.T) {
	srv, cfRepo := newWorkCustomFieldTestServer()
	tenantID := uuid.New()
	cfRepo.defs["cf1"] = &customfield.Definition{ID: "cf1", TenantID: tenantID.String(), Name: "Old", FieldType: "text"}

	ctx := ctxWithLabelTenant(tenantID)
	resp, err := srv.UpdateCustomFieldDefinition(ctx, &workv1.UpdateCustomFieldDefinitionRequest{
		Id: "cf1", Name: "New", FieldType: "number", Position: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, "New", resp.Definition.Name)
	assert.Equal(t, "number", resp.Definition.FieldType)
}

func TestUpdateCustomFieldDefinition_NotFound(t *testing.T) {
	srv, _ := newWorkCustomFieldTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.UpdateCustomFieldDefinition(ctx, &workv1.UpdateCustomFieldDefinitionRequest{Id: "missing", Name: "New", FieldType: "text"})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateCustomFieldDefinition_InvalidFieldType(t *testing.T) {
	srv, cfRepo := newWorkCustomFieldTestServer()
	tenantID := uuid.New()
	cfRepo.defs["cf1"] = &customfield.Definition{ID: "cf1", TenantID: tenantID.String(), Name: "Old", FieldType: "text"}
	ctx := ctxWithLabelTenant(tenantID)
	_, err := srv.UpdateCustomFieldDefinition(ctx, &workv1.UpdateCustomFieldDefinitionRequest{Id: "cf1", Name: "New", FieldType: "not-a-type"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteCustomFieldDefinition_Success(t *testing.T) {
	srv, cfRepo := newWorkCustomFieldTestServer()
	tenantID := uuid.New()
	cfRepo.defs["cf1"] = &customfield.Definition{ID: "cf1", TenantID: tenantID.String(), Name: "Old", FieldType: "text"}
	ctx := ctxWithLabelTenant(tenantID)
	_, err := srv.DeleteCustomFieldDefinition(ctx, &workv1.DeleteCustomFieldDefinitionRequest{Id: "cf1"})
	require.NoError(t, err)
	assert.NotContains(t, cfRepo.defs, "cf1")
}

func TestDeleteCustomFieldDefinition_NotFound(t *testing.T) {
	srv, _ := newWorkCustomFieldTestServer()
	ctx := ctxWithLabelTenant(uuid.New())
	_, err := srv.DeleteCustomFieldDefinition(ctx, &workv1.DeleteCustomFieldDefinitionRequest{Id: "missing"})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestDeleteCustomFieldDefinition_MissingTenant(t *testing.T) {
	srv, _ := newWorkCustomFieldTestServer()
	_, err := srv.DeleteCustomFieldDefinition(context.Background(), &workv1.DeleteCustomFieldDefinitionRequest{Id: "cf1"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}
