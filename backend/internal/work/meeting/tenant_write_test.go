package meeting

// wp-work: closes the write-surface gap for meetings the same way wp-crm-core
// did for CRM entities — the existing tenant_isolation_phase2_test.go seeds
// exclusively via testutil.SeedRow and never calls the real
// CreateMeeting/UpdateMeeting/DeleteMeeting methods. Unlike calendar, every
// write path here already carried an explicit tenant_id predicate AND checked
// RowsAffected before this unit — this test is pure coverage, no bug found.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedMeetingUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-meeting-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

// TestMeetingWrites_LandInCallerTenant covers the core
// CreateMeeting/UpdateMeeting/DeleteMeeting write surface, following the same
// pattern as wp-crm-core/wp-crm-meta.
func TestMeetingWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Meeting Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Meeting Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedMeetingUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	now := time.Now().UTC()
	m := &Meeting{
		ID:             uuid.New(),
		TenantID:       tenantOwn,
		Title:          "Write-Test-" + uuid.New().String()[:8],
		OrganizerID:    userOwn,
		Status:         "scheduled",
		ScheduledStart: now,
		ScheduledEnd:   now.Add(time.Hour),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.CreateMeeting(ctxOther, m); err == nil {
		t.Fatalf("CreateMeeting (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "meetings", m.ID, 0)

	if err := repo.CreateMeeting(ctxOwn, m); err != nil {
		t.Fatalf("CreateMeeting (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", m.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "meetings", m.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "meetings", m.ID, 0)

	// UpdateMeeting carries an explicit tenant_id predicate and checks
	// RowsAffected — a foreign-ctx call must come back an error, not a silent
	// no-op.
	foreign := *m
	foreign.Title = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateMeeting(ctxOther, &foreign); err == nil {
		t.Fatalf("UpdateMeeting (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetMeeting(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetMeeting: %v", err)
	}
	if got.Title != m.Title {
		t.Fatalf("a foreign-tenant write reached the meeting: title=%q", got.Title)
	}

	foreign.Title = "Renamed-" + uuid.New().String()[:8]
	if err := repo.UpdateMeeting(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateMeeting (own ctx): %v", err)
	}
	got, err = repo.GetMeeting(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetMeeting (own ctx): %v", err)
	}
	if got.Title != foreign.Title {
		t.Fatalf("own-tenant write did not land: title=%v", got.Title)
	}

	// DeleteMeeting carries an explicit tenant_id predicate and checks
	// RowsAffected.
	if err := repo.DeleteMeeting(ctxOther, m.ID, tenantOwn); err == nil {
		t.Fatalf("DeleteMeeting (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "meetings", m.ID, 1)

	if err := repo.DeleteMeeting(ctxOwn, m.ID, tenantOwn); err != nil {
		t.Fatalf("DeleteMeeting (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "meetings", m.ID, 0)
}
