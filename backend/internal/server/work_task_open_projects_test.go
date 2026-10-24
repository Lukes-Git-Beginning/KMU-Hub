package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// nonMemberProjectRepo answers like the real repository for a caller who is
// not in project_members: GetMember -> project.ErrNotProjectMember.
type nonMemberProjectRepo struct{ projectStubRepo }

func (*nonMemberProjectRepo) GetMember(context.Context, uuid.UUID, uuid.UUID) (*models.ProjectMember, error) {
	return nil, project.ErrNotProjectMember
}

func (*nonMemberProjectRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Project, error) {
	return &models.Project{ID: id, TenantID: tenantID}, nil
}

// The project calls pass projectsOpenToTenant as isAdmin; the task writes must
// follow the same switch, or a user sees a project and cannot work in it.
func TestWorkGRPC_NonMemberWorksOnTasksOfAnOpenProject(t *testing.T) {
	taskRepo := newWorkTaskMockRepo()
	projRepo := &nonMemberProjectRepo{}
	taskSvc := task.NewService(taskRepo, projRepo)
	srv := NewWorkGRPCServer(
		project.NewService(projRepo), wstatus.NewService(&statusStubRepo{}), taskSvc, taskRepo,
		comment.NewService(newCommentAuthzMockRepo(), taskRepo), timeentry.NewService(&timeentryStubRepo{}),
		label.NewService(newLabelMockRepo()), customfield.NewService(&customfieldStubRepo{}),
	)

	userID, tenantID, projectID, taskID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ctx := context.WithValue(ctxWithActor(userID), middleware.TenantIDKey, tenantID.String())
	taskRepo.tasks[taskID] = &models.TaskWithRelations{Task: models.Task{ID: taskID, ProjectID: &projectID, Title: "open"}}

	if _, err := srv.DeleteTask(ctx, &workv1.DeleteTaskRequest{Id: taskID.String()}); err != nil {
		t.Fatalf("DeleteTask as non-member over the gRPC handler: %v", err)
	}

	// Switch off again: the same call must come back PermissionDenied.
	taskSvc.SetOpenProjects(false)
	_, err := srv.DeleteTask(ctx, &workv1.DeleteTaskRequest{Id: taskID.String()})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("closed projects: want PermissionDenied, got %v", err)
	}
}

// clear_due_date travels over the gRPC handler into the service; absent keeps the date.
func TestWorkGRPC_UpdateTaskClearDueDate(t *testing.T) {
	taskRepo := newWorkTaskMockRepo()
	projRepo := &nonMemberProjectRepo{}
	taskSvc := task.NewService(taskRepo, projRepo)
	taskSvc.SetOpenProjects(true)
	srv := NewWorkGRPCServer(
		project.NewService(projRepo), wstatus.NewService(&statusStubRepo{}), taskSvc, taskRepo,
		comment.NewService(newCommentAuthzMockRepo(), taskRepo), timeentry.NewService(&timeentryStubRepo{}),
		label.NewService(newLabelMockRepo()), customfield.NewService(&customfieldStubRepo{}),
	)

	userID, tenantID, projectID, taskID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ctx := context.WithValue(ctxWithActor(userID), middleware.TenantIDKey, tenantID.String())
	due := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	taskRepo.tasks[taskID] = &models.TaskWithRelations{Task: models.Task{ID: taskID, ProjectID: &projectID, Title: "due", DueDate: &due}}

	title := "renamed"
	if _, err := srv.UpdateTask(ctx, &workv1.UpdateTaskRequest{Id: taskID.String(), UpdatedBy: userID.String(), Title: &title}); err != nil {
		t.Fatalf("UpdateTask without clear: %v", err)
	}
	if taskRepo.tasks[taskID].DueDate == nil {
		t.Fatal("an update without clear_due_date must keep the date")
	}

	if _, err := srv.UpdateTask(ctx, &workv1.UpdateTaskRequest{Id: taskID.String(), UpdatedBy: userID.String(), ClearDueDate: true}); err != nil {
		t.Fatalf("UpdateTask with clear: %v", err)
	}
	if taskRepo.tasks[taskID].DueDate != nil {
		t.Fatalf("clear_due_date must remove the date, got %v", taskRepo.tasks[taskID].DueDate)
	}
}

// is_closed travels over the gRPC handler into the service (task lands in the resolved status),
// a project without a matching status maps to FailedPrecondition, and the task wire type carries
// the closed flag of its status.
func TestWorkGRPC_UpdateTaskIsClosed(t *testing.T) {
	taskRepo := newWorkTaskMockRepo()
	projRepo := &nonMemberProjectRepo{}
	taskSvc := task.NewService(taskRepo, projRepo)
	taskSvc.SetOpenProjects(true)
	srv := NewWorkGRPCServer(
		project.NewService(projRepo), wstatus.NewService(&statusStubRepo{}), taskSvc, taskRepo,
		comment.NewService(newCommentAuthzMockRepo(), taskRepo), timeentry.NewService(&timeentryStubRepo{}),
		label.NewService(newLabelMockRepo()), customfield.NewService(&customfieldStubRepo{}),
	)

	userID, tenantID, projectID, taskID, closedStatus := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ctx := context.WithValue(ctxWithActor(userID), middleware.TenantIDKey, tenantID.String())
	taskRepo.tasks[taskID] = &models.TaskWithRelations{Task: models.Task{ID: taskID, ProjectID: &projectID, Title: "tick"}}

	taskRepo.firstStatusErr = task.ErrNoStatusForState
	yes := true
	_, err := srv.UpdateTask(ctx, &workv1.UpdateTaskRequest{Id: taskID.String(), UpdatedBy: userID.String(), IsClosed: &yes})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("project without a closed status want FailedPrecondition, got %v", err)
	}

	taskRepo.firstStatusErr = nil
	taskRepo.firstStatus = closedStatus
	if _, err := srv.UpdateTask(ctx, &workv1.UpdateTaskRequest{Id: taskID.String(), UpdatedBy: userID.String(), IsClosed: &yes}); err != nil {
		t.Fatalf("UpdateTask is_closed: %v", err)
	}
	if got := taskRepo.tasks[taskID].StatusID; got == nil || *got != closedStatus {
		t.Fatalf("is_closed=true must move the task to the resolved status, got %v", got)
	}

	if !taskToProto(&models.TaskWithRelations{StatusIsClosed: true}).IsClosed {
		t.Error("TaskProto.is_closed must carry the status flag")
	}
}
