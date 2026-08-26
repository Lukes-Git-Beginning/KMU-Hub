package customfield_test

// Exercises PostgresRepository against the real schema (migration 000146):
// tenant scoping on every read/write path, the (tenant_id, name) unique
// constraint, RLS-smoke across a foreign tenant, and what happens to
// task_custom_field_values when a definition with existing values is
// deleted (ON DELETE CASCADE, migration 000320) — see BACKLOG.yml unit
// cov-work-customfield-and-presence-zero-coverage.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/customfield"
)

func seedCFUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("cf-work-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

func seedCFTask(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantID,
		"title":       "CF Task " + uuid.New().String()[:8],
		"task_number": int(uuid.New().ID())%900000 + 100000,
		"created_by":  createdBy,
	})
}

func TestPostgresCreate_AndGetByID_RoundTripsOptions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CF Create Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := customfield.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	def, err := repo.Create(ctx, tenantID.String(), "Priority Tier", "select", []string{"Gold", "Silver", "Bronze"}, 3)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if def.ID == "" {
		t.Fatal("expected generated id")
	}
	if def.Position != 3 {
		t.Fatalf("expected position 3, got %d", def.Position)
	}
	if len(def.Options) != 3 || def.Options[0] != "Gold" {
		t.Fatalf("expected options round-tripped in order, got %v", def.Options)
	}

	fetched, err := repo.GetByID(ctx, tenantID.String(), def.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fetched.Name != "Priority Tier" || fetched.FieldType != "select" {
		t.Fatalf("unexpected fetched definition: %+v", fetched)
	}
}

func TestPostgresCreate_NilOptions_StoresEmptySlice(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CF Nil Options Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := customfield.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	def, err := repo.Create(ctx, tenantID.String(), "Notes", "text", nil, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if def.Options == nil || len(def.Options) != 0 {
		t.Fatalf("expected empty (non-nil) options slice, got %v", def.Options)
	}
}

func TestPostgresCreate_DuplicateNameSameTenant_Rejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CF Duplicate Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := customfield.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	if _, err := repo.Create(ctx, tenantID.String(), "Severity", "text", nil, 0); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := repo.Create(ctx, tenantID.String(), "Severity", "number", nil, 1)
	if err != customfield.ErrDuplicateName {
		t.Fatalf("expected ErrDuplicateName, got %v", err)
	}
}

func TestPostgresCreate_SameNameDifferentTenants_BothSucceed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "CF Same Name Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "CF Same Name Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := customfield.NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	if _, err := repo.Create(ctxA, tenantA.String(), "Severity", "text", nil, 0); err != nil {
		t.Fatalf("Create tenant A: %v", err)
	}
	if _, err := repo.Create(ctxB, tenantB.String(), "Severity", "text", nil, 0); err != nil {
		t.Fatalf("Create tenant B, expected the unique constraint to be tenant-scoped: %v", err)
	}
}

func TestPostgresGetByID_ForeignTenant_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CF RLS Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CF RLS Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := customfield.NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	def, err := repo.Create(ctxOwn, tenantOwn.String(), "Confidential Field", "text", nil, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Under kmuhub_app (NOSUPERUSER NOBYPASSRLS), a foreign tenant's session
	// must not see the row at all, regardless of the tenantID argument passed
	// to GetByID — RLS filters on the session's own app.tenant_id.
	if _, err := repo.GetByID(ctxOther, tenantOwn.String(), def.ID); err != customfield.ErrNotFound {
		t.Fatalf("expected ErrNotFound reading another tenant's definition under RLS, got %v", err)
	}
}

func TestPostgresList_TenantScoped_OrderedByPositionThenName(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CF List Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CF List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := customfield.NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	if _, err := repo.Create(ctxOwn, tenantOwn.String(), "Zeta Field", "text", nil, 1); err != nil {
		t.Fatalf("Create Zeta: %v", err)
	}
	if _, err := repo.Create(ctxOwn, tenantOwn.String(), "Alpha Field", "text", nil, 1); err != nil {
		t.Fatalf("Create Alpha: %v", err)
	}
	if _, err := repo.Create(ctxOwn, tenantOwn.String(), "Prefix Field", "text", nil, 0); err != nil {
		t.Fatalf("Create Prefix: %v", err)
	}
	if _, err := repo.Create(ctxOther, tenantOther.String(), "Foreign Field", "text", nil, 0); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}

	defs, err := repo.List(ctxOwn, tenantOwn.String())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(defs) != 3 {
		t.Fatalf("expected exactly the 3 own-tenant definitions, got %d", len(defs))
	}
	// ORDER BY position ASC, name ASC: position 0 first, then position 1
	// group alphabetically (Alpha before Zeta).
	got := []string{defs[0].Name, defs[1].Name, defs[2].Name}
	want := []string{"Prefix Field", "Alpha Field", "Zeta Field"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected order %v, got %v", want, got)
		}
	}
}

func TestPostgresList_NoDefinitions_ReturnsEmptyNotNilError(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CF Empty List Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := customfield.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	defs, err := repo.List(ctx, tenantID.String())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("expected no definitions, got %d", len(defs))
	}
}

func TestPostgresUpdate_ChangesFields(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CF Update Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := customfield.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	def, err := repo.Create(ctx, tenantID.String(), "Old Name", "text", nil, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := repo.Update(ctx, tenantID.String(), def.ID, "New Name", "select", []string{"A", "B"}, 5)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New Name" || updated.FieldType != "select" || updated.Position != 5 {
		t.Fatalf("update did not apply, got %+v", updated)
	}
	if len(updated.Options) != 2 {
		t.Fatalf("expected 2 options after update, got %v", updated.Options)
	}
}

func TestPostgresUpdate_ForeignTenant_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CF Update RLS Own")
	testutil.EnsureTenant(t, pool, tenantOther, "CF Update RLS Other")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := customfield.NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	def, err := repo.Create(ctxOwn, tenantOwn.String(), "Untouchable", "text", nil, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := repo.Update(ctxOther, tenantOwn.String(), def.ID, "Hijacked", "text", nil, 9); err != customfield.ErrNotFound {
		t.Fatalf("expected ErrNotFound updating another tenant's row under RLS, got %v", err)
	}

	// Confirm the row was not mutated.
	unchanged, err := repo.GetByID(ctxOwn, tenantOwn.String(), def.ID)
	if err != nil {
		t.Fatalf("GetByID after failed cross-tenant update: %v", err)
	}
	if unchanged.Name != "Untouchable" {
		t.Fatalf("cross-tenant update must not have applied, got name %q", unchanged.Name)
	}
}

func TestPostgresUpdate_ToDuplicateName_Rejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CF Update Duplicate Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := customfield.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	if _, err := repo.Create(ctx, tenantID.String(), "Taken", "text", nil, 0); err != nil {
		t.Fatalf("Create Taken: %v", err)
	}
	other, err := repo.Create(ctx, tenantID.String(), "Renameable", "text", nil, 1)
	if err != nil {
		t.Fatalf("Create Renameable: %v", err)
	}

	if _, err := repo.Update(ctx, tenantID.String(), other.ID, "Taken", "text", nil, 1); err != customfield.ErrDuplicateName {
		t.Fatalf("expected ErrDuplicateName, got %v", err)
	}
}

func TestPostgresDelete_RemovesDefinition(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CF Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := customfield.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	def, err := repo.Create(ctx, tenantID.String(), "Disposable", "text", nil, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, tenantID.String(), def.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, tenantID.String(), def.ID); err != customfield.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestPostgresDelete_ForeignTenant_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CF Delete RLS Own")
	testutil.EnsureTenant(t, pool, tenantOther, "CF Delete RLS Other")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := customfield.NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	def, err := repo.Create(ctxOwn, tenantOwn.String(), "Survivor", "text", nil, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctxOther, tenantOwn.String(), def.ID); err != customfield.ErrNotFound {
		t.Fatalf("expected ErrNotFound deleting another tenant's row under RLS, got %v", err)
	}

	if _, err := repo.GetByID(ctxOwn, tenantOwn.String(), def.ID); err != nil {
		t.Fatalf("row must have survived the cross-tenant delete attempt, got %v", err)
	}
}

// TestPostgresDelete_WithExistingTaskValues_CascadesValues documents the
// scope question raised in the unit: deleting a definition that still has
// task_custom_field_values rows pointing at it does not error and does not
// orphan those rows — migration 000320 repointed the FK with
// ON DELETE CASCADE, so the value rows disappear along with the definition.
func TestPostgresDelete_WithExistingTaskValues_CascadesValues(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CF Cascade Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := customfield.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	def, err := repo.Create(ctx, tenantID.String(), "Cascade Field", "text", nil, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	userID := seedCFUser(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "users", userID)
	taskID := seedCFTask(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	defUUID, err := uuid.Parse(def.ID)
	if err != nil {
		t.Fatalf("parse definition id: %v", err)
	}
	sysCtx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO task_custom_field_values (task_id, tenant_id, field_id, value)
		 VALUES ($1, $2, $3, '"in progress"'::jsonb)`,
		taskID, tenantID, defUUID,
	); err != nil {
		t.Fatalf("seed task_custom_field_values: %v", err)
	}

	if err := repo.Delete(ctx, tenantID.String(), def.ID); err != nil {
		t.Fatalf("Delete definition with existing values: %v", err)
	}

	var remaining int
	if err := pool.QueryRow(sysCtx,
		`SELECT count(*) FROM task_custom_field_values WHERE field_id = $1`, defUUID,
	).Scan(&remaining); err != nil {
		t.Fatalf("count remaining values: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected ON DELETE CASCADE to remove the orphaned value row, %d remain", remaining)
	}
}
