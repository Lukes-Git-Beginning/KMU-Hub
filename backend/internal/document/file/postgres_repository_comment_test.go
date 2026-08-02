package file

// DB-level tests for document_file_comments (Migration 000265): tenant
// scoping on list/get/update/delete, and the JOIN-based author_name
// resolution. Mock-repository tests in service_test.go already cover the
// Service contract (author-only edit, author-or-admin delete) against a
// fake; this file proves the SQL layer actually persists and filters by
// tenant_id.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresRepository_CreateAndListComments(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "comment-create-list")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	comment := &models.DocumentFileComment{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		AuthorID: fx.user, Content: "first comment",
	}
	if err := repo.CreateComment(ctx, comment); err != nil {
		t.Fatalf("CreateComment: %v", err)
	}

	comments, err := repo.ListComments(ctx, fx.file, fx.tenant)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("ListComments returned %d entries, want 1", len(comments))
	}
	if comments[0].ID != comment.ID || comments[0].Content != "first comment" || comments[0].AuthorName != "Doc Tester" {
		t.Errorf("ListComments[0] = %+v, want ID=%s Content=%q AuthorName=%q", comments[0], comment.ID, "first comment", "Doc Tester")
	}
}

func TestPostgresRepository_ListComments_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "comment-tenant-isolation")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	if err := repo.CreateComment(ctx, &models.DocumentFileComment{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		AuthorID: fx.user, Content: "tenant-scoped comment",
	}); err != nil {
		t.Fatalf("CreateComment: %v", err)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "document comment test other tenant")

	comments, err := repo.ListComments(ctx, fx.file, otherTenant)
	if err != nil {
		t.Fatalf("ListComments (foreign tenant): %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("ListComments under a foreign tenant_id returned %d entries, want 0", len(comments))
	}
}

func TestPostgresRepository_UpdateComment(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "comment-update")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	comment := &models.DocumentFileComment{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		AuthorID: fx.user, Content: "before edit",
	}
	if err := repo.CreateComment(ctx, comment); err != nil {
		t.Fatalf("CreateComment: %v", err)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "document comment test update wrong tenant")
	if err := repo.UpdateComment(ctx, comment.ID, otherTenant, "hijacked"); err != ErrCommentNotFound {
		t.Fatalf("UpdateComment under a foreign tenant_id = %v, want ErrCommentNotFound", err)
	}

	if err := repo.UpdateComment(ctx, comment.ID, fx.tenant, "after edit"); err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}

	got, err := repo.GetCommentByID(ctx, comment.ID, fx.tenant)
	if err != nil {
		t.Fatalf("GetCommentByID: %v", err)
	}
	if got.Content != "after edit" {
		t.Errorf("GetCommentByID.Content = %q, want %q (foreign-tenant update must not have applied)", got.Content, "after edit")
	}
}

func TestPostgresRepository_DeleteComment(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "comment-delete")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	comment := &models.DocumentFileComment{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		AuthorID: fx.user, Content: "to be deleted",
	}
	if err := repo.CreateComment(ctx, comment); err != nil {
		t.Fatalf("CreateComment: %v", err)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "document comment test delete wrong tenant")
	if err := repo.DeleteComment(ctx, comment.ID, otherTenant); err != ErrCommentNotFound {
		t.Fatalf("DeleteComment under a foreign tenant_id = %v, want ErrCommentNotFound", err)
	}
	if comments, _ := repo.ListComments(ctx, fx.file, fx.tenant); len(comments) != 1 {
		t.Fatalf("comment survived foreign-tenant delete attempt: got %d rows, want 1", len(comments))
	}

	if err := repo.DeleteComment(ctx, comment.ID, fx.tenant); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
	comments, err := repo.ListComments(ctx, fx.file, fx.tenant)
	if err != nil {
		t.Fatalf("ListComments after delete: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("ListComments after delete returned %d entries, want 0", len(comments))
	}
}
