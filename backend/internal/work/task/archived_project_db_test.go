package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/project"
)

// TestCreate_RejectsArchivedProject covers the two creation paths into a
// project: plain Create and the template copy. An archived project takes no new
// task and its task counter stays where it was; an active one still does.
func TestCreate_RejectsArchivedProject(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Archived Project Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	owner := seedWorkUser(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "users", owner)

	active := seedWorkProject(t, pool, tenantID, owner)
	defer testutil.CleanupRow(t, pool, "projects", active)
	archived := testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":        tenantID,
		"name":             "Archived Project",
		"project_key":      "AR" + uuid.New().String()[:6],
		"next_task_number": 1,
		"created_by":       owner,
		"archived_at":      time.Now(),
	})
	defer testutil.CleanupRow(t, pool, "projects", archived)
	for _, id := range []uuid.UUID{active, archived} {
		if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`INSERT INTO project_members (project_id, user_id, tenant_id, role) VALUES ($1, $2, $3, 'owner')`,
			id, owner, tenantID); err != nil {
			t.Fatalf("seed member: %v", err)
		}
	}
	template := seedWorkTask(t, pool, tenantID, active, owner, "template task")
	defer testutil.CleanupRow(t, pool, "tasks", template)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	svc := NewService(NewPostgresRepository(pool), project.NewPostgresRepository(pool))

	counter := func(id uuid.UUID) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(testutil.WithSystemCtx(ctx), `SELECT next_task_number FROM projects WHERE id = $1`, id).Scan(&n); err != nil {
			t.Fatalf("read counter: %v", err)
		}
		return n
	}
	taskCount := func(id uuid.UUID) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(testutil.WithSystemCtx(ctx), `SELECT count(*) FROM tasks WHERE project_id = $1`, id).Scan(&n); err != nil {
			t.Fatalf("count tasks: %v", err)
		}
		return n
	}

	t.Run("create in archived project is refused", func(t *testing.T) {
		if _, err := svc.Create(ctx, CreateInput{TenantID: tenantID, ProjectID: &archived, Title: "late", CreatedBy: owner}); !errors.Is(err, project.ErrProjectArchived) {
			t.Fatalf("want ErrProjectArchived, got %v", err)
		}
		if n := counter(archived); n != 1 {
			t.Fatalf("task counter moved to %d", n)
		}
		if n := taskCount(archived); n != 0 {
			t.Fatalf("%d tasks landed in the archived project", n)
		}
	})

	t.Run("template copy into archived project is refused", func(t *testing.T) {
		if err := svc.CreateFromTemplate(ctx, tenantID, active, archived, owner, nil); !errors.Is(err, project.ErrProjectArchived) {
			t.Fatalf("want ErrProjectArchived, got %v", err)
		}
		if n := counter(archived); n != 1 {
			t.Fatalf("task counter moved to %d", n)
		}
		if n := taskCount(archived); n != 0 {
			t.Fatalf("%d tasks landed in the archived project", n)
		}
	})

	t.Run("create in active project still works", func(t *testing.T) {
		created, err := svc.Create(ctx, CreateInput{TenantID: tenantID, ProjectID: &active, Title: "fresh", CreatedBy: owner})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "tasks", created.ID)
	})
}
