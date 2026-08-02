package calendar

// wp-work: closes the write-surface gap for calendars the same way wp-crm-core
// did for CRM entities — the existing tenant_isolation_phase2_test.go seeds
// exclusively via testutil.SeedRow and never calls the real Create/Update/Delete
// methods.
//
// Fund: Update/Delete swallowed a cross-tenant no-op as success — Exec's err is
// nil when RLS silently filters the WHERE clause to zero rows, unlike every
// sibling repo in this backlog (project/timeentry/resource/meeting all check
// RowsAffected). Fixed in postgres_repository.go alongside this test, which
// falsifies it: reverting the RowsAffected check makes Update/Delete
// (foreign ctx) return nil instead of an error.

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

func seedCalendarUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-calendar-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

// TestCalendarWrites_LandInCallerTenant covers the core Create/Update/Delete
// write surface, following the same pattern as wp-crm-core/wp-crm-meta.
func TestCalendarWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Calendar Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Calendar Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedCalendarUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	now := time.Now().UTC()
	cal := &models.Calendar{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		Name:         "Write-Test-" + uuid.New().String()[:8],
		CalendarType: models.CalendarTypePersonal,
		Color:        "#4285F4",
		OwnerID:      userOwn,
		Timezone:     "Europe/Berlin",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, cal); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "calendars", cal.ID, 0)

	if err := repo.Create(ctxOwn, cal); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "calendars", cal.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "calendars", cal.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "calendars", cal.ID, 0)

	// Update: a foreign-ctx call must come back an error, not a silent no-op.
	foreign := *cal
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); err == nil {
		t.Fatalf("Update (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetByID(ctxOwn, cal.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != cal.Name {
		t.Fatalf("a foreign-tenant write reached the calendar: name=%q", got.Name)
	}

	foreign.Name = "Renamed-" + uuid.New().String()[:8]
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, cal.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != foreign.Name {
		t.Fatalf("own-tenant write did not land: name=%v", got.Name)
	}

	// Delete: a foreign-ctx call must come back an error, not a silent no-op.
	if err := repo.Delete(ctxOther, cal.ID, tenantOwn); err == nil {
		t.Fatalf("Delete (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "calendars", cal.ID, 1)

	if err := repo.Delete(ctxOwn, cal.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "calendars", cal.ID, 0)
}
