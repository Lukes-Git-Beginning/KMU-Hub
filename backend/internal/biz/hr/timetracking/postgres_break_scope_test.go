package timetracking_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Same system-context reasoning as postgres_tenant_scope_test.go: the policy is
// switched off so a row that still cannot be reached was filtered by the query's
// own tenant_id predicate rather than by RLS.

// TestBreakCreate_WritesTenant is the regression test for the actual outage.
// Migration 000230 added tenant_id NOT NULL to hr_break_entries but the INSERT
// never learned about the column, so every StartBreak against a real database
// failed with a not-null violation — starting a break was dead, not degraded.
func TestBreakCreate_WritesTenant(t *testing.T) {
	f := newScopeFixture(t)
	repo := timetracking.NewPostgresBreakRepo(f.pool)

	entry := &models.HRBreakEntry{
		ID:              uuid.New(),
		TenantID:        testutil.TenantA,
		WorkTimeEntryID: f.entryA,
		StartTime:       time.Now().Truncate(time.Second),
		CreatedAt:       time.Now().Truncate(time.Second),
	}
	if err := repo.Create(f.ctx, entry); err != nil {
		t.Fatalf("create break: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_break_entries", entry.ID) })

	var stored uuid.UUID
	err := f.pool.QueryRow(f.ctx,
		`SELECT tenant_id FROM hr_break_entries WHERE id = $1`, entry.ID).Scan(&stored)
	if err != nil {
		t.Fatalf("read back break: %v", err)
	}
	if stored != testutil.TenantA {
		t.Fatalf("stored tenant %s, want %s", stored, testutil.TenantA)
	}
}

// seedBreak inserts an open break for the given tenant's work time entry.
func seedBreak(t *testing.T, f *scopeFixture, tenantID, entryID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, f.pool, "hr_break_entries", map[string]any{
		"id":                 uuid.New(),
		"tenant_id":          tenantID,
		"work_time_entry_id": entryID,
		"start_time":         time.Now().Add(-30 * time.Minute).Truncate(time.Second),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_break_entries", id) })
	return id
}

func TestBreakGetActive_ForeignTenantNotFound(t *testing.T) {
	f := newScopeFixture(t)
	repo := timetracking.NewPostgresBreakRepo(f.pool)
	breakB := seedBreak(t, f, testutil.TenantB, f.entryB)

	own, err := repo.GetActiveBreak(f.ctx, testutil.TenantB, f.entryB)
	if err != nil {
		t.Fatalf("own active break: %v", err)
	}
	if own == nil || own.ID != breakB {
		t.Fatal("TenantB must see its own active break")
	}

	foreign, err := repo.GetActiveBreak(f.ctx, testutil.TenantA, f.entryB)
	if err != nil {
		t.Fatalf("cross-tenant lookup must not error: %v", err)
	}
	if foreign != nil {
		t.Fatal("TenantA saw TenantB's active break")
	}
}

func TestBreakList_ForeignTenantEmpty(t *testing.T) {
	f := newScopeFixture(t)
	repo := timetracking.NewPostgresBreakRepo(f.pool)
	seedBreak(t, f, testutil.TenantB, f.entryB)

	own, err := repo.ListByWorkTimeEntry(f.ctx, testutil.TenantB, f.entryB)
	if err != nil {
		t.Fatalf("own break list: %v", err)
	}
	if len(own) != 1 {
		t.Fatalf("TenantB sees %d breaks, want 1", len(own))
	}

	foreign, err := repo.ListByWorkTimeEntry(f.ctx, testutil.TenantA, f.entryB)
	if err != nil {
		t.Fatalf("cross-tenant list must not error: %v", err)
	}
	if len(foreign) != 0 {
		t.Fatalf("TenantA listed %d of TenantB's breaks", len(foreign))
	}
}

// TestBreakUpdate_ForeignTenantMissesRow is the write half: ending a break
// matched on the break id alone, so a wrong id closed another tenant's break.
func TestBreakUpdate_ForeignTenantMissesRow(t *testing.T) {
	f := newScopeFixture(t)
	repo := timetracking.NewPostgresBreakRepo(f.pool)
	breakB := seedBreak(t, f, testutil.TenantB, f.entryB)

	end := time.Now().Truncate(time.Second)
	minutes := 30
	if err := repo.Update(f.ctx, &models.HRBreakEntry{
		ID:              breakB,
		TenantID:        testutil.TenantA,
		EndTime:         &end,
		DurationMinutes: &minutes,
	}); err != nil {
		t.Fatalf("cross-tenant update must not error: %v", err)
	}

	var stillOpen bool
	if err := f.pool.QueryRow(f.ctx,
		`SELECT end_time IS NULL FROM hr_break_entries WHERE id = $1`, breakB).Scan(&stillOpen); err != nil {
		t.Fatalf("read back break: %v", err)
	}
	if !stillOpen {
		t.Fatal("TenantA's write reached TenantB's break")
	}
}
