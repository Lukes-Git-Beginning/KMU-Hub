package server

import (
	"context"
	"testing"

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
