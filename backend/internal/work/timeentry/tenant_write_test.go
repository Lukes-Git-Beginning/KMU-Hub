package timeentry

// wp-work: closes the write-surface gap for time entries the same way
// wp-crm-core did for CRM entities. Unlike task/project, every write path in
// this package already carried an explicit tenant_id predicate before this
// unit — this test is pure coverage, no bug found.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedTimeEntryUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-timeentry-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

func seedTimeEntryProject(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":        tenantID,
		"name":             "WP TimeEntry Project",
		"project_key":      "WT" + uuid.New().String()[:6],
		"next_task_number": 1,
		"created_by":       createdBy,
	})
}

func seedTimeEntryTask(t *testing.T, pool *pgxpool.Pool, tenantID, projectID, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantID,
		"project_id":  projectID,
		"title":       "WP TimeEntry Task",
		"task_number": int(uuid.New().ID())%900000 + 100000,
		"created_by":  createdBy,
	})
}

// TestTimeEntryWrites_LandInCallerTenant covers the core Create/Update/Delete
// write surface, following the same pattern as wp-crm-core/wp-crm-meta.
func TestTimeEntryWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "TimeEntry Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "TimeEntry Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedTimeEntryUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedTimeEntryProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)
	taskOwn := seedTimeEntryTask(t, pool, tenantOwn, projectOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "tasks", taskOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	now := time.Now().UTC()
	entry := &models.TimeEntry{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		TaskID:    taskOwn,
		UserID:    userOwn,
		StartedAt: now,
		IsManual:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, entry); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "time_entries", entry.ID, 0)

	if err := repo.Create(ctxOwn, entry); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "time_entries", entry.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "time_entries", entry.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "time_entries", entry.ID, 0)

	// Update carries an explicit tenant_id predicate and checks RowsAffected
	// — a foreign-ctx call must come back an error, not a silent no-op.
	foreign := *entry
	desc := "Hacked"
	foreign.Description = &desc
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); err == nil {
		t.Fatalf("Update (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetByID(ctxOwn, entry.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Description != nil {
		t.Fatalf("a foreign-tenant write reached the time entry: description=%q", *got.Description)
	}

	ownDesc := "Renamed-" + uuid.New().String()[:8]
	foreign.Description = &ownDesc
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, entry.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Description == nil || *got.Description != ownDesc {
		t.Fatalf("own-tenant write did not land: description=%v", got.Description)
	}

	// Delete carries an explicit tenantID predicate.
	if err := repo.Delete(ctxOther, entry.ID, tenantOwn); err == nil {
		t.Fatalf("Delete (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "time_entries", entry.ID, 1)

	if err := repo.Delete(ctxOwn, entry.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "time_entries", entry.ID, 0)
}
