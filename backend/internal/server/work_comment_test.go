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

// commentAuthzMockRepo is a comment.Repository backed by an in-memory map. Unlike
// commentStubRepo (work_label_test.go), GetByID returns a real, seeded comment so
// the author/admin authorization branches in comment.Service actually execute.
type commentAuthzMockRepo struct {
	comments     map[uuid.UUID]*models.TaskComment
	lastPage     int
	lastPageSize int
}

func newCommentAuthzMockRepo() *commentAuthzMockRepo {
	return &commentAuthzMockRepo{comments: make(map[uuid.UUID]*models.TaskComment)}
}

func (r *commentAuthzMockRepo) Create(_ context.Context, c *models.TaskComment) error {
	r.comments[c.ID] = c
	return nil
}

func (r *commentAuthzMockRepo) GetByID(_ context.Context, id uuid.UUID) (*models.TaskComment, error) {
	c, ok := r.comments[id]
	if !ok {
		return nil, comment.ErrNotFound
	}
	return c, nil
}

func (r *commentAuthzMockRepo) List(_ context.Context, taskID uuid.UUID, page, pageSize int) ([]comment.TaskCommentWithAuthor, int, error) {
	r.lastPage = page
	r.lastPageSize = pageSize
	var out []comment.TaskCommentWithAuthor
	for _, c := range r.comments {
		if c.TaskID == taskID {
			out = append(out, comment.TaskCommentWithAuthor{TaskComment: *c})
		}
	}
	return out, len(out), nil
}

func (r *commentAuthzMockRepo) Update(_ context.Context, id uuid.UUID, content string) error {
	c, ok := r.comments[id]
	if !ok {
		return comment.ErrNotFound
	}
	c.Content = content
	return nil
}

func (r *commentAuthzMockRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.comments, id)
	return nil
}

func newWorkCommentTestServer(repo comment.Repository) *WorkGRPCServer {
	taskRepo := newWorkTaskMockRepo()
	projectSvc := project.NewService(&projectStubRepo{})
	statusSvc := wstatus.NewService(&statusStubRepo{})
	taskSvc := task.NewService(taskRepo, &projectStubRepo{})
	commentSvc := comment.NewService(repo, taskRepo)
	teSvc := timeentry.NewService(&timeentryStubRepo{})
	labelSvc := label.NewService(newLabelMockRepo())
	cfSvc := customfield.NewService(&customfieldStubRepo{})

	return NewWorkGRPCServer(projectSvc, statusSvc, taskSvc, taskRepo, commentSvc, teSvc, labelSvc, cfSvc)
}

// ctxWithActor mirrors how middleware.TenantInboundUnaryInterceptor populates
// UserIDKey from the gateway-propagated x-user-id metadata on the real path.
func ctxWithActor(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, userID.String())
}

// Regression coverage for fix-g-work-task-comment-authz: UpdateTaskComment and
// DeleteTaskComment used to call comment.Service with a hardcoded uuid.Nil actor
// (breaking every author update) and isAdmin=true (letting any user delete any
// comment). These tests exercise the real gRPC handlers, not just the service.

func TestUpdateTaskComment_AuthorCanEdit(t *testing.T) {
	repo := newCommentAuthzMockRepo()
	authorID := uuid.New()
	commentID := uuid.New()
	repo.comments[commentID] = &models.TaskComment{ID: commentID, AuthorID: authorID, Content: "original"}

	srv := newWorkCommentTestServer(repo)
	resp, err := srv.UpdateTaskComment(ctxWithActor(authorID), &workv1.UpdateTaskCommentRequest{
		Id:      commentID.String(),
		Content: "updated",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "updated", repo.comments[commentID].Content)
}

func TestUpdateTaskComment_NonAuthorDenied(t *testing.T) {
	repo := newCommentAuthzMockRepo()
	authorID := uuid.New()
	otherID := uuid.New()
	commentID := uuid.New()
	repo.comments[commentID] = &models.TaskComment{ID: commentID, AuthorID: authorID, Content: "original"}

	srv := newWorkCommentTestServer(repo)
	_, err := srv.UpdateTaskComment(ctxWithActor(otherID), &workv1.UpdateTaskCommentRequest{
		Id:      commentID.String(),
		Content: "hijacked",
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
	assert.Equal(t, "original", repo.comments[commentID].Content, "content must not change")
}

func TestUpdateTaskComment_MissingActorContext(t *testing.T) {
	repo := newCommentAuthzMockRepo()
	commentID := uuid.New()
	repo.comments[commentID] = &models.TaskComment{ID: commentID, AuthorID: uuid.New(), Content: "original"}

	srv := newWorkCommentTestServer(repo)
	_, err := srv.UpdateTaskComment(context.Background(), &workv1.UpdateTaskCommentRequest{
		Id:      commentID.String(),
		Content: "updated",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestDeleteTaskComment_AuthorCanDelete(t *testing.T) {
	repo := newCommentAuthzMockRepo()
	authorID := uuid.New()
	commentID := uuid.New()
	repo.comments[commentID] = &models.TaskComment{ID: commentID, AuthorID: authorID}

	srv := newWorkCommentTestServer(repo)
	_, err := srv.DeleteTaskComment(ctxWithActor(authorID), &workv1.DeleteTaskCommentRequest{
		Id: commentID.String(),
	})
	require.NoError(t, err)
	_, getErr := repo.GetByID(context.Background(), commentID)
	assert.ErrorIs(t, getErr, comment.ErrNotFound)
}

func TestDeleteTaskComment_NonAuthorNonAdminDenied(t *testing.T) {
	repo := newCommentAuthzMockRepo()
	authorID := uuid.New()
	otherID := uuid.New()
	commentID := uuid.New()
	repo.comments[commentID] = &models.TaskComment{ID: commentID, AuthorID: authorID}

	srv := newWorkCommentTestServer(repo)
	_, err := srv.DeleteTaskComment(ctxWithActor(otherID), &workv1.DeleteTaskCommentRequest{
		Id:      commentID.String(),
		IsAdmin: false,
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
	_, getErr := repo.GetByID(context.Background(), commentID)
	require.NoError(t, getErr, "comment must still exist")
}

func TestDeleteTaskComment_AdminCanDeleteOthers(t *testing.T) {
	repo := newCommentAuthzMockRepo()
	authorID := uuid.New()
	adminID := uuid.New()
	commentID := uuid.New()
	repo.comments[commentID] = &models.TaskComment{ID: commentID, AuthorID: authorID}

	srv := newWorkCommentTestServer(repo)
	_, err := srv.DeleteTaskComment(ctxWithActor(adminID), &workv1.DeleteTaskCommentRequest{
		Id:      commentID.String(),
		IsAdmin: true,
	})
	require.NoError(t, err)
	_, getErr := repo.GetByID(context.Background(), commentID)
	assert.ErrorIs(t, getErr, comment.ErrNotFound)
}
