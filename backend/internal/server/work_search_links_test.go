package server

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/models"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

var errStubWorkTaskRepoFailure = errors.New("stub work task repo: forced failure")

// ============================================================================
// LinkEntityToTask / UnlinkEntityFromTask / ListTaskEntityLinks / ListEntityTasks
// ============================================================================

func TestLinkEntityToTask_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.LinkEntityToTask(context.Background(), &workv1.LinkEntityToTaskRequest{
		TaskId: uuid.NewString(), EntityId: uuid.NewString(), CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestLinkEntityToTask_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.LinkEntityToTask(ctxWithWorkTenant(uuid.New()), &workv1.LinkEntityToTaskRequest{
		TaskId: "not-a-uuid", EntityId: uuid.NewString(), CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLinkEntityToTask_InvalidEntityID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.LinkEntityToTask(ctxWithWorkTenant(uuid.New()), &workv1.LinkEntityToTaskRequest{
		TaskId: uuid.NewString(), EntityId: "not-a-uuid", CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLinkEntityToTask_InvalidCreatedBy(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.LinkEntityToTask(ctxWithWorkTenant(uuid.New()), &workv1.LinkEntityToTaskRequest{
		TaskId: uuid.NewString(), EntityId: uuid.NewString(), CreatedBy: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLinkEntityToTask_HappyPath(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	taskID := uuid.New()
	entityID := uuid.New()
	createdBy := uuid.New()

	resp, err := srv.LinkEntityToTask(ctxWithWorkTenant(tenantID), &workv1.LinkEntityToTaskRequest{
		TaskId: taskID.String(), EntityType: "contact", EntityId: entityID.String(), CreatedBy: createdBy.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Link)
	assert.Equal(t, "contact", resp.Link.EntityType)
	assert.Equal(t, entityID.String(), resp.Link.EntityId)
	require.Contains(t, taskRepo.entityLinks, uuid.MustParse(resp.Link.Id))
}

func TestLinkEntityToTask_RepoErrorMapped(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskRepo.forceErr = errStubWorkTaskRepoFailure

	_, err := srv.LinkEntityToTask(ctxWithWorkTenant(uuid.New()), &workv1.LinkEntityToTaskRequest{
		TaskId: uuid.NewString(), EntityId: uuid.NewString(), CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.Internal)
}

func TestUnlinkEntityFromTask_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UnlinkEntityFromTask(context.Background(), &workv1.UnlinkEntityFromTaskRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUnlinkEntityFromTask_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UnlinkEntityFromTask(ctxWithWorkTenant(uuid.New()), &workv1.UnlinkEntityFromTaskRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUnlinkEntityFromTask_NotFound(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	_, err := srv.UnlinkEntityFromTask(ctxWithWorkTenant(uuid.New()), &workv1.UnlinkEntityFromTaskRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUnlinkEntityFromTask_HappyPath(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	linkID := uuid.New()
	taskRepo.entityLinks = map[uuid.UUID]models.TaskEntityLink{
		linkID: {ID: linkID, TaskID: uuid.New(), EntityType: "contact", EntityID: uuid.New()},
	}

	_, err := srv.UnlinkEntityFromTask(ctxWithWorkTenant(uuid.New()), &workv1.UnlinkEntityFromTaskRequest{Id: linkID.String()})
	require.NoError(t, err)
	assert.NotContains(t, taskRepo.entityLinks, linkID)
}

func TestListTaskEntityLinks_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListTaskEntityLinks(context.Background(), &workv1.ListTaskEntityLinksRequest{TaskId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListTaskEntityLinks_RepoErrorIsInternal(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskRepo.forceErr = errStubWorkTaskRepoFailure

	_, err := srv.ListTaskEntityLinks(context.Background(), &workv1.ListTaskEntityLinksRequest{TaskId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Internal)
}

func TestListTaskEntityLinks_HappyPath(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskID := uuid.New()
	linkID := uuid.New()
	taskRepo.entityLinks = map[uuid.UUID]models.TaskEntityLink{
		linkID: {ID: linkID, TaskID: taskID, EntityType: "deal", EntityID: uuid.New()},
	}

	resp, err := srv.ListTaskEntityLinks(context.Background(), &workv1.ListTaskEntityLinksRequest{TaskId: taskID.String()})
	require.NoError(t, err)
	require.Len(t, resp.Links, 1)
	assert.Equal(t, "deal", resp.Links[0].EntityType)
}

func TestListEntityTasks_InvalidEntityID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListEntityTasks(context.Background(), &workv1.ListEntityTasksRequest{EntityId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListEntityTasks_RepoErrorIsInternal(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskRepo.forceErr = errStubWorkTaskRepoFailure

	_, err := srv.ListEntityTasks(context.Background(), &workv1.ListEntityTasksRequest{EntityId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Internal)
}

func TestListEntityTasks_HappyPath(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	entityID := uuid.New()
	taskID := uuid.New()
	taskRepo.tasks[taskID] = &models.TaskWithRelations{Task: models.Task{ID: taskID, Title: "Linked", Priority: models.TaskPriorityNormal, CreatedBy: uuid.New()}}
	linkID := uuid.New()
	taskRepo.entityLinks = map[uuid.UUID]models.TaskEntityLink{
		linkID: {ID: linkID, TaskID: taskID, EntityType: "company", EntityID: entityID},
	}

	resp, err := srv.ListEntityTasks(context.Background(), &workv1.ListEntityTasksRequest{EntityType: "company", EntityId: entityID.String()})
	require.NoError(t, err)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, int32(1), resp.Total)
	assert.Equal(t, taskID.String(), resp.Tasks[0].Id)
}

// ============================================================================
// ListTaskActivities
// ============================================================================

func TestListTaskActivities_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListTaskActivities(context.Background(), &workv1.ListTaskActivitiesRequest{TaskId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListTaskActivities_RepoErrorIsInternal(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskRepo.forceErr = errStubWorkTaskRepoFailure

	_, err := srv.ListTaskActivities(context.Background(), &workv1.ListTaskActivitiesRequest{TaskId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Internal)
}

func TestListTaskActivities_HappyPathDefaultsPageAndSize(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskID := uuid.New()
	taskRepo.activities = []models.TaskActivity{
		{ID: uuid.New(), TaskID: taskID, ActorID: uuid.New(), Action: "created"},
	}

	resp, err := srv.ListTaskActivities(context.Background(), &workv1.ListTaskActivitiesRequest{TaskId: taskID.String()})
	require.NoError(t, err)
	require.Len(t, resp.Activities, 1)
	assert.Equal(t, "created", resp.Activities[0].Action)
	assert.Equal(t, int32(1), resp.Total)
}

// ============================================================================
// AttachFileToTask / RemoveTaskFile / ListTaskFiles
// ============================================================================

func TestAttachFileToTask_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.AttachFileToTask(context.Background(), &workv1.AttachFileToTaskRequest{TaskId: "not-a-uuid", UploadedBy: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAttachFileToTask_InvalidUploadedBy(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.AttachFileToTask(context.Background(), &workv1.AttachFileToTaskRequest{TaskId: uuid.NewString(), UploadedBy: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAttachFileToTask_RepoErrorMapped(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskRepo.forceErr = errStubWorkTaskRepoFailure

	_, err := srv.AttachFileToTask(context.Background(), &workv1.AttachFileToTaskRequest{
		TaskId: uuid.NewString(), UploadedBy: uuid.NewString(), Filename: "a.png",
	})
	requireGRPCCode(t, err, codes.Internal)
}

func TestAttachFileToTask_HappyPath(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskID := uuid.New()
	uploadedBy := uuid.New()

	resp, err := srv.AttachFileToTask(context.Background(), &workv1.AttachFileToTaskRequest{
		TaskId: taskID.String(), UploadedBy: uploadedBy.String(), Filename: "report.pdf", MimeType: "application/pdf", FileSize: 1024,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.File)
	assert.Equal(t, "report.pdf", resp.File.Filename)
	assert.Nil(t, resp.File.ThumbnailKey)
	require.Contains(t, taskRepo.files, uuid.MustParse(resp.File.Id))
}

func TestRemoveTaskFile_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.RemoveTaskFile(context.Background(), &workv1.RemoveTaskFileRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestRemoveTaskFile_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.RemoveTaskFile(ctxWithWorkTenant(uuid.New()), &workv1.RemoveTaskFileRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRemoveTaskFile_NotFound(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	_, err := srv.RemoveTaskFile(ctxWithWorkTenant(uuid.New()), &workv1.RemoveTaskFileRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestRemoveTaskFile_HappyPath(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	fileID := uuid.New()
	taskRepo.files = map[uuid.UUID]models.TaskFile{fileID: {ID: fileID, TaskID: uuid.New(), Filename: "x.png"}}

	_, err := srv.RemoveTaskFile(ctxWithWorkTenant(uuid.New()), &workv1.RemoveTaskFileRequest{Id: fileID.String()})
	require.NoError(t, err)
	assert.NotContains(t, taskRepo.files, fileID)
}

func TestListTaskFiles_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListTaskFiles(context.Background(), &workv1.ListTaskFilesRequest{TaskId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListTaskFiles_RepoErrorIsInternal(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskRepo.forceErr = errStubWorkTaskRepoFailure

	_, err := srv.ListTaskFiles(context.Background(), &workv1.ListTaskFilesRequest{TaskId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Internal)
}

func TestListTaskFiles_HappyPath(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskID := uuid.New()
	fileID := uuid.New()
	taskRepo.files = map[uuid.UUID]models.TaskFile{fileID: {ID: fileID, TaskID: taskID, Filename: "spec.pdf"}}

	resp, err := srv.ListTaskFiles(context.Background(), &workv1.ListTaskFilesRequest{TaskId: taskID.String()})
	require.NoError(t, err)
	require.Len(t, resp.Files, 1)
	assert.Equal(t, "spec.pdf", resp.Files[0].Filename)
}

// ============================================================================
// SetTaskCustomFieldValues / GetTaskCustomFieldValues
// ============================================================================

func TestSetTaskCustomFieldValues_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SetTaskCustomFieldValues(context.Background(), &workv1.SetTaskCustomFieldValuesRequest{TaskId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestSetTaskCustomFieldValues_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SetTaskCustomFieldValues(ctxWithWorkTenant(uuid.New()), &workv1.SetTaskCustomFieldValuesRequest{TaskId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetTaskCustomFieldValues_InvalidFieldID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SetTaskCustomFieldValues(ctxWithWorkTenant(uuid.New()), &workv1.SetTaskCustomFieldValuesRequest{
		TaskId: uuid.NewString(),
		Values: []*workv1.CustomFieldValueInput{{FieldId: "not-a-uuid", Value: `"x"`}},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetTaskCustomFieldValues_RepoErrorMapped(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskRepo.forceErr = errStubWorkTaskRepoFailure

	_, err := srv.SetTaskCustomFieldValues(ctxWithWorkTenant(uuid.New()), &workv1.SetTaskCustomFieldValuesRequest{TaskId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Internal)
}

func TestSetTaskCustomFieldValues_HappyPathRoundTripsJSONValue(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	taskID := uuid.New()
	fieldID := uuid.New()

	resp, err := srv.SetTaskCustomFieldValues(ctxWithWorkTenant(uuid.New()), &workv1.SetTaskCustomFieldValuesRequest{
		TaskId: taskID.String(),
		Values: []*workv1.CustomFieldValueInput{{FieldId: fieldID.String(), Value: `42`}},
	})
	require.NoError(t, err)
	require.Contains(t, resp.CustomFields, fieldID.String())
	assert.Equal(t, "42", resp.CustomFields[fieldID.String()])
}

func TestGetTaskCustomFieldValues_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.GetTaskCustomFieldValues(context.Background(), &workv1.GetTaskCustomFieldValuesRequest{TaskId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetTaskCustomFieldValues_RepoErrorIsInternal(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskRepo.forceErr = errStubWorkTaskRepoFailure

	_, err := srv.GetTaskCustomFieldValues(context.Background(), &workv1.GetTaskCustomFieldValuesRequest{TaskId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Internal)
}

func TestGetTaskCustomFieldValues_EmptyReturnsNoError(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	resp, err := srv.GetTaskCustomFieldValues(context.Background(), &workv1.GetTaskCustomFieldValuesRequest{TaskId: uuid.NewString()})
	require.NoError(t, err)
	assert.Empty(t, resp.CustomFields)
}

// ============================================================================
// GetUserProjectPreference / SetUserProjectPreference
// ============================================================================

func TestGetUserProjectPreference_InvalidUserID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.GetUserProjectPreference(context.Background(), &workv1.GetUserProjectPreferenceRequest{UserId: "not-a-uuid", ProjectId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetUserProjectPreference_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.GetUserProjectPreference(context.Background(), &workv1.GetUserProjectPreferenceRequest{UserId: uuid.NewString(), ProjectId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetUserProjectPreference_UnknownReturnsDefaults(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	userID := uuid.New()
	projectID := uuid.New()

	resp, err := srv.GetUserProjectPreference(context.Background(), &workv1.GetUserProjectPreferenceRequest{
		UserId: userID.String(), ProjectId: projectID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Preference)
	assert.Equal(t, "list", resp.Preference.ViewType)
	assert.Equal(t, "status", resp.Preference.ListGroupBy)
	assert.False(t, resp.Preference.ListSortDesc)
}

func TestGetUserProjectPreference_ReturnsStoredValue(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	userID := uuid.New()
	projectID := uuid.New()
	projRepo.prefs[userID.String()+"|"+projectID.String()] = &models.UserProjectPreference{
		UserID: userID, ProjectID: projectID, ViewType: "kanban", ListGroupBy: "assignee", ListSortBy: "due_date", ListSortDesc: true,
	}

	resp, err := srv.GetUserProjectPreference(context.Background(), &workv1.GetUserProjectPreferenceRequest{
		UserId: userID.String(), ProjectId: projectID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, "kanban", resp.Preference.ViewType)
	assert.True(t, resp.Preference.ListSortDesc)
}

func TestSetUserProjectPreference_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SetUserProjectPreference(context.Background(), &workv1.SetUserProjectPreferenceRequest{UserId: uuid.NewString(), ProjectId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestSetUserProjectPreference_InvalidUserID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SetUserProjectPreference(ctxWithWorkTenant(uuid.New()), &workv1.SetUserProjectPreferenceRequest{UserId: "not-a-uuid", ProjectId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetUserProjectPreference_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SetUserProjectPreference(ctxWithWorkTenant(uuid.New()), &workv1.SetUserProjectPreferenceRequest{UserId: uuid.NewString(), ProjectId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetUserProjectPreference_CreatesNewWithDefaultsAndOverride(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	viewType := "kanban"

	resp, err := srv.SetUserProjectPreference(ctxWithWorkTenant(tenantID), &workv1.SetUserProjectPreferenceRequest{
		UserId: userID.String(), ProjectId: projectID.String(), ViewType: &viewType,
	})
	require.NoError(t, err)
	assert.Equal(t, "kanban", resp.Preference.ViewType)
	assert.Equal(t, "status", resp.Preference.ListGroupBy, "unset fields keep the service default")

	stored := projRepo.prefs[userID.String()+"|"+projectID.String()]
	require.NotNil(t, stored)
	assert.Equal(t, tenantID, stored.TenantID)
}

func TestSetUserProjectPreference_BackfillsMissingTenantOnExisting(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	projRepo.prefs[userID.String()+"|"+projectID.String()] = &models.UserProjectPreference{
		UserID: userID, ProjectID: projectID, ViewType: "list", ListGroupBy: "status", ListSortBy: "sort_order",
	}

	_, err := srv.SetUserProjectPreference(ctxWithWorkTenant(tenantID), &workv1.SetUserProjectPreferenceRequest{
		UserId: userID.String(), ProjectId: projectID.String(),
	})
	require.NoError(t, err)

	stored := projRepo.prefs[userID.String()+"|"+projectID.String()]
	assert.Equal(t, tenantID, stored.TenantID)
}

// ============================================================================
// SearchTasks
// ============================================================================

func TestSearchTasks_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SearchTasks(context.Background(), &workv1.SearchTasksRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestSearchTasks_InvalidProjectIDInList(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SearchTasks(ctxWithWorkTenant(uuid.New()), &workv1.SearchTasksRequest{ProjectIds: []string{"not-a-uuid"}})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSearchTasks_InvalidAssigneeIDInList(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SearchTasks(ctxWithWorkTenant(uuid.New()), &workv1.SearchTasksRequest{AssigneeIds: []string{"not-a-uuid"}})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSearchTasks_RepoErrorIsInternal(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	taskRepo.forceErr = errStubWorkTaskRepoFailure

	_, err := srv.SearchTasks(ctxWithWorkTenant(uuid.New()), &workv1.SearchTasksRequest{})
	requireGRPCCode(t, err, codes.Internal)
}

func TestSearchTasks_FiltersByProjectAndAssignee(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	projectID := uuid.New()
	assigneeID := uuid.New()
	otherProject := uuid.New()

	match := uuid.New()
	taskRepo.tasks[match] = &models.TaskWithRelations{Task: models.Task{
		ID: match, ProjectID: &projectID, AssigneeID: &assigneeID, Title: "Match", Priority: models.TaskPriorityNormal, CreatedBy: uuid.New(),
	}}
	wrongProject := uuid.New()
	taskRepo.tasks[wrongProject] = &models.TaskWithRelations{Task: models.Task{
		ID: wrongProject, ProjectID: &otherProject, AssigneeID: &assigneeID, Title: "WrongProject", Priority: models.TaskPriorityNormal, CreatedBy: uuid.New(),
	}}
	noAssignee := uuid.New()
	taskRepo.tasks[noAssignee] = &models.TaskWithRelations{Task: models.Task{
		ID: noAssignee, ProjectID: &projectID, Title: "NoAssignee", Priority: models.TaskPriorityNormal, CreatedBy: uuid.New(),
	}}

	resp, err := srv.SearchTasks(ctxWithWorkTenant(tenantID), &workv1.SearchTasksRequest{
		ProjectIds:  []string{projectID.String()},
		AssigneeIds: []string{assigneeID.String()},
	})
	require.NoError(t, err)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, int32(1), resp.Total)
	assert.Equal(t, match.String(), resp.Tasks[0].Id)
}

func TestSearchTasks_NoFiltersReturnsAll(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	id := uuid.New()
	taskRepo.tasks[id] = &models.TaskWithRelations{Task: models.Task{ID: id, Title: "Only", Priority: models.TaskPriorityNormal, CreatedBy: uuid.New()}}

	resp, err := srv.SearchTasks(ctxWithWorkTenant(tenantID), &workv1.SearchTasksRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Tasks, 1)
}
