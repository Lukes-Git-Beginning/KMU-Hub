package recording

// wp-work: closes the write-surface gap for recordings the same way
// wp-crm-core did for CRM entities. Unlike calendar/meeting/resource, the
// repo methods here never took a tenantID parameter at all — tenant scoping
// is enforced purely via RLS on ctx (see rls_test.go, which seeds through
// testutil.SeedRow and never exercises the real Create/Update/Delete path).
//
// Fund: UpdateRecording/DeleteRecording swallowed a cross-tenant no-op as
// success — Exec's err is nil when RLS silently filters the WHERE clause to
// zero rows. Fixed in postgres_repository.go alongside this test, which
// falsifies it: reverting the RowsAffected check makes Update/Delete
// (foreign ctx) return nil instead of an error.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedRecordingUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-recording-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

func seedRecordingMeeting(t *testing.T, pool *pgxpool.Pool, tenantID, organizerID uuid.UUID) uuid.UUID {
	t.Helper()
	now := time.Now().UTC()
	return testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id":       tenantID,
		"title":           "WP Recording Meeting",
		"organizer_id":    organizerID,
		"status":          "scheduled",
		"scheduled_start": now,
		"scheduled_end":   now.Add(time.Hour),
	})
}

// TestRecordingWrites_LandInCallerTenant covers the core
// CreateRecording/UpdateRecording/DeleteRecording write surface, following
// the same pattern as wp-crm-core/wp-crm-meta.
func TestRecordingWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Recording Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Recording Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedRecordingUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	meetingOwn := seedRecordingMeeting(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "meetings", meetingOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	now := time.Now().UTC()
	rec := &Recording{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		MeetingID: &meetingOwn,
		StartedBy: &userOwn,
		ConsentSnapshot: []ParticipantConsentInfo{
			{UserID: userOwn, DisplayName: "WP Recording User", JoinedAt: now},
		},
		Status:    RecordingStatusActive,
		CreatedAt: now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.CreateRecording(ctxOther, rec); err == nil {
		t.Fatalf("CreateRecording (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "recordings", rec.ID, 0)

	if err := repo.CreateRecording(ctxOwn, rec); err != nil {
		t.Fatalf("CreateRecording (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", rec.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "recordings", rec.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "recordings", rec.ID, 0)

	// UpdateRecording relies on RLS alone (no tenantID parameter) — a
	// foreign-ctx call must come back an error, not a silent no-op.
	foreign := *rec
	foreign.Status = RecordingStatusFailed
	if err := repo.UpdateRecording(ctxOther, &foreign); err == nil {
		t.Fatalf("UpdateRecording (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetRecording(ctxOwn, rec.ID)
	if err != nil {
		t.Fatalf("GetRecording: %v", err)
	}
	if got.Status != RecordingStatusActive {
		t.Fatalf("a foreign-tenant write reached the recording: status=%q", got.Status)
	}

	foreign.Status = RecordingStatusProcessing
	if err := repo.UpdateRecording(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateRecording (own ctx): %v", err)
	}
	got, err = repo.GetRecording(ctxOwn, rec.ID)
	if err != nil {
		t.Fatalf("GetRecording (own ctx): %v", err)
	}
	if got.Status != RecordingStatusProcessing {
		t.Fatalf("own-tenant write did not land: status=%v", got.Status)
	}

	// DeleteRecording relies on RLS alone (no tenantID parameter).
	if err := repo.DeleteRecording(ctxOther, rec.ID); err == nil {
		t.Fatalf("DeleteRecording (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "recordings", rec.ID, 1)

	if err := repo.DeleteRecording(ctxOwn, rec.ID); err != nil {
		t.Fatalf("DeleteRecording (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "recordings", rec.ID, 0)
}
