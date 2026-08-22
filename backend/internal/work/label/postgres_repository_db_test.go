package label_test

// postgres_repository_db_test.go runs the label repository against the real
// schema instead of the mockLabelRepo in label_test.go. Filed for
// cov-gateway-work-labels-custom-fields (BACKLOG.yml): "Ein Label zu
// loeschen, das noch verwendet wird, ist derselbe Fragenkomplex wie bei
// Kontakten. An pg_constraint nachsehen."

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/label"
)

func seedLabelUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-label-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

func seedLabelTask(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantID,
		"title":       "WP Label Task",
		"task_number": int(uuid.New().ID())%900000 + 100000,
		"created_by":  createdBy,
	})
}

// TestLabelDelete_CascadesTaskLabels proves the "also cascades to
// task_labels" doc comment on Repository.Delete against the real
// pg_constraint (migration 000145: work_labels(id) -> task_labels ON DELETE
// CASCADE), not just the mock repo: deleting a label still assigned to a
// task must silently remove the assignment, not fail or leave a row behind.
func TestLabelDelete_CascadesTaskLabels(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Label Cascade Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := seedLabelUser(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "users", userID)
	taskID := seedLabelTask(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	repo := label.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	l, err := repo.Create(ctx, tenantID.String(), "Cascade Probe", "#6b7280")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.SetTaskLabels(ctx, tenantID.String(), taskID.String(), []string{l.ID}); err != nil {
		t.Fatalf("SetTaskLabels: %v", err)
	}

	before, err := repo.GetTaskLabelIDs(ctx, tenantID.String(), taskID.String())
	if err != nil {
		t.Fatalf("GetTaskLabelIDs before delete: %v", err)
	}
	if len(before) != 1 || before[0] != l.ID {
		t.Fatalf("expected label assigned before delete, got %v", before)
	}

	if err := repo.Delete(ctx, tenantID.String(), l.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	after, err := repo.GetTaskLabelIDs(ctx, tenantID.String(), taskID.String())
	if err != nil {
		t.Fatalf("GetTaskLabelIDs after delete: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected task_labels row removed by cascade, got %v", after)
	}
}

// TestLabelRepository_TenantIsolation exercises Create/GetByID/List against
// the real RLS policy on work_labels (migration 000145), not the mock repo:
// a label created under tenant A must be invisible to tenant B.
func TestLabelRepository_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Label RLS Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Label RLS Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := label.NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	l, err := repo.Create(ctxA, tenantA.String(), "RLS Probe", "#3b82f6")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "work_labels", uuid.MustParse(l.ID))

	if _, err := repo.GetByID(ctxB, tenantB.String(), l.ID); err == nil {
		t.Fatal("expected tenant B to not see tenant A's label")
	}

	labelsB, err := repo.List(ctxB, tenantB.String())
	if err != nil {
		t.Fatalf("List as tenant B: %v", err)
	}
	for _, other := range labelsB {
		if other.ID == l.ID {
			t.Fatal("tenant B's list leaked tenant A's label")
		}
	}
}
