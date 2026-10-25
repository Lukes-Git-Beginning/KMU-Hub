package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/project"
)

// TestTaskFileWrites_NeedProjectEditRights_RealSQL: attaching and removing a task file follow
// the rule Update uses. A viewer of the task's project and a non-member (closed mode) are
// refused and leave the file list untouched, an editor may attach and remove, a project-less
// task is open to every caller of the tenant, and another tenant's file reads as not found.
func TestTaskFileWrites_NeedProjectEditRights_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task File Rights Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Task File Rights Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	editor := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", editor)
	viewer := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", viewer)
	outsider := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", outsider)
	otherUser := seedWorkUser(t, pool, tenantOther)
	defer testutil.CleanupRow(t, pool, "users", otherUser)

	proj := seedWorkProject(t, pool, tenantOwn, editor)
	defer testutil.CleanupRow(t, pool, "projects", proj)
	if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO project_members (project_id, user_id, tenant_id, role) VALUES ($1, $2, $3, 'member'), ($1, $4, $3, 'viewer')
		 ON CONFLICT (project_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		proj, editor, tenantOwn, viewer); err != nil {
		t.Fatalf("seed members: %v", err)
	}

	inProject := seedWorkTask(t, pool, tenantOwn, proj, editor, "in project")
	defer testutil.CleanupRow(t, pool, "tasks", inProject)
	loose := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id": tenantOwn, "title": "loose", "task_number": 0, "created_by": editor,
	})
	defer testutil.CleanupRow(t, pool, "tasks", loose)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, project.NewPostgresRepository(pool))
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	newFile := func(taskID, by uuid.UUID) *models.TaskFile {
		return &models.TaskFile{
			ID: uuid.New(), TaskID: taskID, Filename: "a.pdf", MimeType: "application/pdf", FileSize: 1,
			StorageKey: chatfile.UnattachedUploadPrefix(tenantOwn) + uuid.NewString(), UploadedBy: by, CreatedAt: time.Now(),
		}
	}
	count := func(taskID uuid.UUID) int {
		t.Helper()
		files, err := repo.ListFiles(ctx, taskID)
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
		return len(files)
	}

	// Attach: viewer and outsider are refused, nothing is written.
	if err := svc.AttachFile(ctx, tenantOwn, viewer, newFile(inProject, viewer)); !errors.Is(err, ErrViewerCannotEdit) {
		t.Fatalf("viewer attach: want ErrViewerCannotEdit, got %v", err)
	}
	if err := svc.AttachFile(ctx, tenantOwn, outsider, newFile(inProject, outsider)); !errors.Is(err, ErrNotProjectMember) {
		t.Fatalf("outsider attach: want ErrNotProjectMember, got %v", err)
	}
	if n := count(inProject); n != 0 {
		t.Fatalf("refused attaches must not write, task has %d files", n)
	}

	// Attach: the editor may.
	kept := newFile(inProject, editor)
	if err := svc.AttachFile(ctx, tenantOwn, editor, kept); err != nil {
		t.Fatalf("editor attach: %v", err)
	}

	// Remove: viewer and outsider are refused, the file stays.
	if err := svc.RemoveFile(ctx, tenantOwn, viewer, kept.ID); !errors.Is(err, ErrViewerCannotEdit) {
		t.Fatalf("viewer remove: want ErrViewerCannotEdit, got %v", err)
	}
	if err := svc.RemoveFile(ctx, tenantOwn, outsider, kept.ID); !errors.Is(err, ErrNotProjectMember) {
		t.Fatalf("outsider remove: want ErrNotProjectMember, got %v", err)
	}
	if n := count(inProject); n != 1 {
		t.Fatalf("refused removes must keep the file, task has %d files", n)
	}

	// Another tenant cannot see the file, let alone delete it.
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	if err := svc.RemoveFile(ctxOther, tenantOther, otherUser, kept.ID); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("foreign tenant remove: want ErrFileNotFound, got %v", err)
	}

	// Remove: the editor may.
	if err := svc.RemoveFile(ctx, tenantOwn, editor, kept.ID); err != nil {
		t.Fatalf("editor remove: %v", err)
	}
	if n := count(inProject); n != 0 {
		t.Fatalf("editor remove must delete the file, task has %d files", n)
	}

	// A project-less task behaves like Update: any caller of the tenant may write.
	free := newFile(loose, outsider)
	if err := svc.AttachFile(ctx, tenantOwn, outsider, free); err != nil {
		t.Fatalf("attach to a project-less task: %v", err)
	}
	if err := svc.RemoveFile(ctx, tenantOwn, viewer, free.ID); err != nil {
		t.Fatalf("remove from a project-less task: %v", err)
	}
	if n := count(loose); n != 0 {
		t.Fatalf("project-less remove must delete the file, task has %d files", n)
	}
}
