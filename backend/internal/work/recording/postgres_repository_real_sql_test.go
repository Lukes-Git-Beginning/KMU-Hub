package recording

// Exercises the PostgresRepository methods that rls_test.go and
// tenant_write_test.go never touch (see coverage_start in the
// cov-video-recording-service-and-repository unit): the various list/query
// methods sat at 0% real-SQL coverage even though the service-layer mock-repo
// tests exercised the business logic above them. These tests hit a live
// Postgres so JSONB round-trips, JOIN/UNION shapes, and the retention-expiry
// WHERE clause are proven against the real schema, not a Go map.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedRecordingCallSession(t *testing.T, pool *pgxpool.Pool, tenantID, initiatorID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "call_sessions", map[string]any{
		"tenant_id":    tenantID,
		"call_type":    "group",
		"room_name":    fmt.Sprintf("wp-recording-call-%s", uuid.New()),
		"initiator_id": initiatorID,
	})
}

// anySnapshot returns a non-empty consent snapshot: recordings.consent_snapshot
// has a NOT NULL CHECK constraint (migration 000091) and CreateRecording always
// inserts the column explicitly, so an empty/nil snapshot is rejected by
// Postgres itself, not just a service-layer rule.
func anySnapshot(userID uuid.UUID) []ParticipantConsentInfo {
	return []ParticipantConsentInfo{{UserID: userID, DisplayName: "Real-SQL Test User", JoinedAt: time.Now().UTC()}}
}

// TestPostgresRepository_ListByMeetingAndByCall_RealSQL covers the two
// list-by-parent queries, previously 0% real-SQL coverage.
func TestPostgresRepository_ListByMeetingAndByCall_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Recording List Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	user := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", user)
	meetingA := seedRecordingMeeting(t, pool, tenant, user)
	defer testutil.CleanupRow(t, pool, "meetings", meetingA)
	meetingB := seedRecordingMeeting(t, pool, tenant, user)
	defer testutil.CleanupRow(t, pool, "meetings", meetingB)
	call := seedRecordingCallSession(t, pool, tenant, user)
	defer testutil.CleanupRow(t, pool, "call_sessions", call)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	now := time.Now().UTC()
	meetingRec := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meetingA, Status: RecordingStatusActive, ConsentSnapshot: anySnapshot(user), CreatedAt: now}
	if err := repo.CreateRecording(ctx, meetingRec); err != nil {
		t.Fatalf("CreateRecording (meeting): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", meetingRec.ID)

	callRec := &Recording{ID: uuid.New(), TenantID: tenant, CallID: &call, Status: RecordingStatusActive, ConsentSnapshot: anySnapshot(user), CreatedAt: now}
	if err := repo.CreateRecording(ctx, callRec); err != nil {
		t.Fatalf("CreateRecording (call): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", callRec.ID)

	// Distractor tied to a different meeting must not leak into meetingA's list.
	otherRec := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meetingB, Status: RecordingStatusActive, ConsentSnapshot: anySnapshot(user), CreatedAt: now}
	if err := repo.CreateRecording(ctx, otherRec); err != nil {
		t.Fatalf("CreateRecording (other meeting): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", otherRec.ID)

	byMeeting, err := repo.ListRecordingsByMeeting(ctx, meetingA)
	if err != nil {
		t.Fatalf("ListRecordingsByMeeting: %v", err)
	}
	if len(byMeeting) != 1 || byMeeting[0].ID != meetingRec.ID {
		t.Fatalf("ListRecordingsByMeeting = %+v, want exactly [%s]", byMeeting, meetingRec.ID)
	}

	byCall, err := repo.ListRecordingsByCall(ctx, call)
	if err != nil {
		t.Fatalf("ListRecordingsByCall: %v", err)
	}
	if len(byCall) != 1 || byCall[0].ID != callRec.ID {
		t.Fatalf("ListRecordingsByCall = %+v, want exactly [%s]", byCall, callRec.ID)
	}
}

// TestPostgresRepository_GetRecordingByEgressID_RealSQL covers the
// webhook lookup path (LiveKit Egress completion/failure callbacks resolve
// the recording by egress_id, not by our own UUID).
func TestPostgresRepository_GetRecordingByEgressID_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Recording Egress Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	user := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", user)
	meeting := seedRecordingMeeting(t, pool, tenant, user)
	defer testutil.CleanupRow(t, pool, "meetings", meeting)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	egressID := fmt.Sprintf("egress-%s", uuid.New())
	rec := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meeting, Status: RecordingStatusActive, EgressID: &egressID, ConsentSnapshot: anySnapshot(user), CreatedAt: time.Now().UTC()}
	if err := repo.CreateRecording(ctx, rec); err != nil {
		t.Fatalf("CreateRecording: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", rec.ID)

	got, err := repo.GetRecordingByEgressID(ctx, egressID)
	if err != nil {
		t.Fatalf("GetRecordingByEgressID: %v", err)
	}
	if got.ID != rec.ID {
		t.Fatalf("GetRecordingByEgressID returned %s, want %s", got.ID, rec.ID)
	}

	if _, err := repo.GetRecordingByEgressID(ctx, "unknown-egress-id"); err != ErrNotFound {
		t.Fatalf("GetRecordingByEgressID (unknown): got %v, want ErrNotFound", err)
	}
}

// TestPostgresRepository_TagRecordingWithConsents_RealSQL proves the JSONB
// snapshot round-trips through Postgres and that an empty snapshot is
// written as the "[]" sentinel, never NULL (marshalConsentSnapshot's own
// comment warns this matters for old recordings).
func TestPostgresRepository_TagRecordingWithConsents_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Recording Tag Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	user := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", user)
	meeting := seedRecordingMeeting(t, pool, tenant, user)
	defer testutil.CleanupRow(t, pool, "meetings", meeting)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	rec := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meeting, StartedBy: &user, Status: RecordingStatusActive, ConsentSnapshot: anySnapshot(user), CreatedAt: time.Now().UTC()}
	if err := repo.CreateRecording(ctx, rec); err != nil {
		t.Fatalf("CreateRecording: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", rec.ID)

	newSnapshot := []ParticipantConsentInfo{
		{UserID: user, DisplayName: "Late Joiner", JoinedAt: time.Now().UTC()},
	}
	if err := repo.TagRecordingWithConsents(ctx, rec.ID, newSnapshot); err != nil {
		t.Fatalf("TagRecordingWithConsents: %v", err)
	}
	got, err := repo.GetRecording(ctx, rec.ID)
	if err != nil {
		t.Fatalf("GetRecording: %v", err)
	}
	if len(got.ConsentSnapshot) != 1 || got.ConsentSnapshot[0].UserID != user {
		t.Fatalf("ConsentSnapshot after tag = %+v, want one entry for %s", got.ConsentSnapshot, user)
	}

	if err := repo.TagRecordingWithConsents(ctx, rec.ID, nil); err != nil {
		t.Fatalf("TagRecordingWithConsents (empty): %v", err)
	}
	var raw []byte
	if err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		"SELECT consent_snapshot FROM recordings WHERE id = $1", rec.ID,
	).Scan(&raw); err != nil {
		t.Fatalf("read back consent_snapshot: %v", err)
	}
	if raw == nil {
		t.Fatalf("consent_snapshot stored as NULL, want the \"[]\" sentinel")
	}
	if string(raw) != "[]" {
		t.Fatalf("consent_snapshot = %s, want []", raw)
	}
}

// TestPostgresRepository_ConsentQueries_RealSQL covers GetConsents ordering,
// GetConsentsWithUser's LEFT JOIN onto users, and CountPendingConsents.
func TestPostgresRepository_ConsentQueries_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Recording Consent Query Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	organizer := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", organizer)
	userA := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", userA)
	userB := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", userB)
	userC := seedRecordingUser(t, pool, tenant) // never responds -- the "pending" case
	defer testutil.CleanupRow(t, pool, "users", userC)
	meeting := seedRecordingMeeting(t, pool, tenant, organizer)
	defer testutil.CleanupRow(t, pool, "meetings", meeting)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	rec := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meeting, Status: RecordingStatusActive, ConsentSnapshot: anySnapshot(organizer), CreatedAt: time.Now().UTC()}
	if err := repo.CreateRecording(ctx, rec); err != nil {
		t.Fatalf("CreateRecording: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", rec.ID)

	now := time.Now().UTC()
	if err := repo.SetConsent(ctx, &RecordingConsent{RecordingID: rec.ID, UserID: userB, Consented: true, RespondedAt: now.Add(time.Minute)}); err != nil {
		t.Fatalf("SetConsent userB: %v", err)
	}
	if err := repo.SetConsent(ctx, &RecordingConsent{RecordingID: rec.ID, UserID: userA, Consented: false, RespondedAt: now}); err != nil {
		t.Fatalf("SetConsent userA: %v", err)
	}

	consents, err := repo.GetConsents(ctx, rec.ID)
	if err != nil {
		t.Fatalf("GetConsents: %v", err)
	}
	if len(consents) != 2 || consents[0].UserID != userA || consents[1].UserID != userB {
		t.Fatalf("GetConsents order = %+v, want [userA(earlier), userB(later)]", consents)
	}

	withUser, err := repo.GetConsentsWithUser(ctx, rec.ID)
	if err != nil {
		t.Fatalf("GetConsentsWithUser: %v", err)
	}
	if len(withUser) != 2 {
		t.Fatalf("GetConsentsWithUser returned %d rows, want 2", len(withUser))
	}
	for _, c := range withUser {
		// seedRecordingUser never sets first_name/last_name -- the LEFT JOIN's
		// COALESCE must still produce empty strings, not a scan error.
		if c.FirstName != "" || c.LastName != "" {
			t.Fatalf("GetConsentsWithUser name = %q %q, want empty strings from COALESCE", c.FirstName, c.LastName)
		}
	}

	pending, err := repo.CountPendingConsents(ctx, rec.ID, []uuid.UUID{userA, userB, userC})
	if err != nil {
		t.Fatalf("CountPendingConsents: %v", err)
	}
	if pending != 1 {
		t.Fatalf("CountPendingConsents = %d, want 1 (only userC has not responded)", pending)
	}
}

// TestPostgresRepository_ListExpiredRecordings_RealSQL locks the retention
// WHERE clause: only status='completed' AND retention_expires_at in the past
// is eligible for cleanup. An expired-but-still-active recording (e.g. a
// crashed egress that never transitioned) must NOT be swept up as if it were
// a finished recording ready for deletion.
func TestPostgresRepository_ListExpiredRecordings_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Recording Expiry Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	user := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", user)
	meeting := seedRecordingMeeting(t, pool, tenant, user)
	defer testutil.CleanupRow(t, pool, "meetings", meeting)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	expiredCompleted := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meeting, Status: RecordingStatusCompleted, RetentionExpiresAt: &past, ConsentSnapshot: anySnapshot(user), CreatedAt: now}
	notYetExpired := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meeting, Status: RecordingStatusCompleted, RetentionExpiresAt: &future, ConsentSnapshot: anySnapshot(user), CreatedAt: now}
	expiredButActive := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meeting, Status: RecordingStatusActive, RetentionExpiresAt: &past, ConsentSnapshot: anySnapshot(user), CreatedAt: now}

	for _, r := range []*Recording{expiredCompleted, notYetExpired, expiredButActive} {
		if err := repo.CreateRecording(ctx, r); err != nil {
			t.Fatalf("CreateRecording(%s): %v", r.Status, err)
		}
		defer testutil.CleanupRow(t, pool, "recordings", r.ID)
	}

	expired, err := repo.ListExpiredRecordings(ctx, now)
	if err != nil {
		t.Fatalf("ListExpiredRecordings: %v", err)
	}
	if len(expired) != 1 || expired[0].ID != expiredCompleted.ID {
		t.Fatalf("ListExpiredRecordings = %+v, want exactly [%s]", expired, expiredCompleted.ID)
	}
}

// TestPostgresRepository_AccessAndParticipants_RealSQL covers
// ListRecordingsWithAccess (Phase 11 file-manager integration point) and
// GetRecordingParticipants (the ACL check behind GetRecordingDownloadURL):
// a meeting attendee must see/appear for the recording, a stranger must not,
// and a non-completed/processing recording must not surface via
// ListRecordingsWithAccess even for an actual attendee.
func TestPostgresRepository_AccessAndParticipants_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Recording Access Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	organizer := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", organizer)
	attendee := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", attendee)
	stranger := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", stranger)
	meeting := seedRecordingMeeting(t, pool, tenant, organizer)
	defer testutil.CleanupRow(t, pool, "meetings", meeting)

	sysCtx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO meeting_attendees (meeting_id, user_id, tenant_id) VALUES ($1, $2, $3)`, meeting, attendee, tenant,
	); err != nil {
		t.Fatalf("seed meeting_attendees: %v", err)
	}
	defer pool.Exec(sysCtx, "DELETE FROM meeting_attendees WHERE meeting_id=$1 AND user_id=$2", meeting, attendee)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	completedRec := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meeting, Status: RecordingStatusCompleted, ConsentSnapshot: anySnapshot(organizer), CreatedAt: time.Now().UTC()}
	if err := repo.CreateRecording(ctx, completedRec); err != nil {
		t.Fatalf("CreateRecording (completed): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", completedRec.ID)

	activeRec := &Recording{ID: uuid.New(), TenantID: tenant, MeetingID: &meeting, Status: RecordingStatusActive, ConsentSnapshot: anySnapshot(organizer), CreatedAt: time.Now().UTC()}
	if err := repo.CreateRecording(ctx, activeRec); err != nil {
		t.Fatalf("CreateRecording (active): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", activeRec.ID)

	attendeeAccess, err := repo.ListRecordingsWithAccess(ctx, attendee, nil, nil)
	if err != nil {
		t.Fatalf("ListRecordingsWithAccess (attendee): %v", err)
	}
	if len(attendeeAccess) != 1 || attendeeAccess[0].ID != completedRec.ID {
		t.Fatalf("ListRecordingsWithAccess (attendee) = %+v, want exactly the completed recording", attendeeAccess)
	}

	strangerAccess, err := repo.ListRecordingsWithAccess(ctx, stranger, nil, nil)
	if err != nil {
		t.Fatalf("ListRecordingsWithAccess (stranger): %v", err)
	}
	if len(strangerAccess) != 0 {
		t.Fatalf("ListRecordingsWithAccess (stranger) = %+v, want none", strangerAccess)
	}

	participants, err := repo.GetRecordingParticipants(ctx, completedRec.ID)
	if err != nil {
		t.Fatalf("GetRecordingParticipants: %v", err)
	}
	found := false
	for _, p := range participants {
		if p == stranger {
			t.Fatalf("GetRecordingParticipants includes the stranger: %+v", participants)
		}
		if p == attendee {
			found = true
		}
	}
	if !found {
		t.Fatalf("GetRecordingParticipants = %+v, want the attendee included", participants)
	}
}

// TestPostgresRepository_MarkInitiatorConsent_And_PreConsentStatus_RealSQL
// covers the pre-consent stamp used by the service-layer gate in
// StartRecording. Both the tenant and the started_by user are enforced in
// the WHERE clause (not via ctx/RLS alone), so a wrong tenant or a wrong
// user must fail closed with ErrNotFound rather than silently no-op.
func TestPostgresRepository_MarkInitiatorConsent_And_PreConsentStatus_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Pre-Consent Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Pre-Consent Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	user := seedRecordingUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", user)
	otherUser := seedRecordingUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", otherUser)
	meeting := seedRecordingMeeting(t, pool, tenantOwn, user)
	defer testutil.CleanupRow(t, pool, "meetings", meeting)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	rec := &Recording{ID: uuid.New(), TenantID: tenantOwn, MeetingID: &meeting, StartedBy: &user, Status: RecordingStatusActive, ConsentSnapshot: anySnapshot(user), CreatedAt: time.Now().UTC()}
	if err := repo.CreateRecording(ctx, rec); err != nil {
		t.Fatalf("CreateRecording: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", rec.ID)

	if stamped, err := repo.GetPreConsentStatus(ctx, rec.ID, tenantOwn); err != nil || stamped {
		t.Fatalf("GetPreConsentStatus (before mark) = (%v, %v), want (false, nil)", stamped, err)
	}

	if err := repo.MarkInitiatorConsent(ctx, rec.ID, otherUser, tenantOwn); err != ErrNotFound {
		t.Fatalf("MarkInitiatorConsent (wrong user): got %v, want ErrNotFound", err)
	}
	if err := repo.MarkInitiatorConsent(ctx, rec.ID, user, tenantOther); err != ErrNotFound {
		t.Fatalf("MarkInitiatorConsent (wrong tenant): got %v, want ErrNotFound", err)
	}

	if err := repo.MarkInitiatorConsent(ctx, rec.ID, user, tenantOwn); err != nil {
		t.Fatalf("MarkInitiatorConsent (correct): %v", err)
	}

	if stamped, err := repo.GetPreConsentStatus(ctx, rec.ID, tenantOwn); err != nil || !stamped {
		t.Fatalf("GetPreConsentStatus (after mark) = (%v, %v), want (true, nil)", stamped, err)
	}
	if _, err := repo.GetPreConsentStatus(ctx, rec.ID, tenantOther); err != ErrNotFound {
		t.Fatalf("GetPreConsentStatus (wrong tenant): got %v, want ErrNotFound", err)
	}
}
