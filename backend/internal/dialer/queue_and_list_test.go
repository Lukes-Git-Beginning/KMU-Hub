package dialer

// Covers the largest previously untested block of postgres_repository.go:
// campaign/contact List+ListContacts pagination, the AddContacts/
// GetNextPendingContact queue logic, GetCampaignStats/UpdateCampaignCounts
// aggregation, the full CallRepository session lifecycle (including the
// atomic UpdateSessionWithEventAndContact transaction), the today-count and
// recent-calls reads, and OutcomeRepository.List/EnsureDefaults plus
// AgentStatusRepository's two read helpers. tenant_write_test.go already
// covers Campaign/CampaignContact/Outcome CRUD writes and GetAgentStats'
// active-campaign scoping — not repeated here.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// ---------------------------------------------------------------------------
// Fixture helpers
// ---------------------------------------------------------------------------

// seedDialerUser inserts a user with a globally unique email derived from
// emailPrefix — a static email string would collide across repeated local
// runs whose fixtures outlive the pool.Close() that skips CleanupRow.
func seedDialerUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, emailPrefix, first, last string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenantID,
		"email":         emailPrefix + "-" + uuid.New().String() + "@test.local",
		"password_hash": "$2a$10$placeholder",
		"first_name":    first,
		"last_name":     last,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}

func seedDialerCampaign(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID, name, status string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantID,
		"name":       name,
		"status":     status,
		"created_by": createdBy,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_campaigns", id) })
	return id
}

func seedDialerContact(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID, first, last string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantID,
		"first_name": first,
		"last_name":  last,
		"created_by": createdBy,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", id) })
	return id
}

func seedDialerCampaignContact(t *testing.T, pool *pgxpool.Pool, tenantID, campID, contactID uuid.UUID, position int, status string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   tenantID,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    position,
		"status":      status,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_campaign_contacts", id) })
	return id
}

func seedDialerOutcome(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, label string, isPositive, isAppointment bool) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "dialer_call_outcomes", map[string]any{
		"id":             uuid.New(),
		"tenant_id":      tenantID,
		"label":          label,
		"color":          "#123456",
		"is_positive":    isPositive,
		"is_appointment": isAppointment,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_call_outcomes", id) })
	return id
}

func seedDialerCallSession(t *testing.T, pool *pgxpool.Pool, tenantID, ccID, agentID uuid.UUID, outcomeID *uuid.UUID, durationSecs *int) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"id":                  uuid.New(),
		"tenant_id":           tenantID,
		"campaign_contact_id": ccID,
		"agent_id":            agentID,
	}
	if outcomeID != nil {
		cols["outcome_id"] = *outcomeID
	}
	if durationSecs != nil {
		cols["duration_seconds"] = *durationSecs
	}
	id := testutil.SeedRow(t, pool, "dialer_call_sessions", cols)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_call_sessions", id) })
	return id
}

func strPtr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// CampaignRepository.List / ListContacts — pagination + status filter +
// tenant-scoped total
// ---------------------------------------------------------------------------

func TestCampaignRepository_List_FiltersPaginatesAndScopesTotal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer List Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Dialer List Other Tenant")

	userID := seedDialerUser(t, pool, tenantOwn, "dialer-list", "List", "Owner")
	seedDialerCampaign(t, pool, tenantOwn, userID, "Draft Campaign", CampaignStatusDraft)
	seedDialerCampaign(t, pool, tenantOwn, userID, "Active Campaign 1", CampaignStatusActive)
	seedDialerCampaign(t, pool, tenantOwn, userID, "Active Campaign 2", CampaignStatusActive)

	otherUserID := seedDialerUser(t, pool, tenantOther, "dialer-list-other", "Other", "Owner")
	seedDialerCampaign(t, pool, tenantOther, otherUserID, "Foreign Campaign", CampaignStatusActive)

	repo := NewPostgresCampaignRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	all, total, err := repo.List(ctxOwn, tenantOwn, nil, 1, 10)
	if err != nil {
		t.Fatalf("List (no filter): %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Fatalf("List (no filter): expected total=3 len=3, got total=%d len=%d", total, len(all))
	}

	active, total, err := repo.List(ctxOwn, tenantOwn, strPtr(CampaignStatusActive), 1, 10)
	if err != nil {
		t.Fatalf("List (status filter): %v", err)
	}
	if total != 2 || len(active) != 2 {
		t.Fatalf("List (status filter): expected total=2 len=2, got total=%d len=%d", total, len(active))
	}

	// Pagination: page size smaller than total must not shrink the total count.
	page1, total, err := repo.List(ctxOwn, tenantOwn, nil, 1, 2)
	if err != nil {
		t.Fatalf("List (page 1): %v", err)
	}
	if total != 3 || len(page1) != 2 {
		t.Fatalf("List (page 1): expected total=3 len=2, got total=%d len=%d", total, len(page1))
	}
	page2, total, err := repo.List(ctxOwn, tenantOwn, nil, 2, 2)
	if err != nil {
		t.Fatalf("List (page 2): %v", err)
	}
	if total != 3 || len(page2) != 1 {
		t.Fatalf("List (page 2): expected total=3 len=1, got total=%d len=%d", total, len(page2))
	}

	// Cross-tenant: the total must carry the same tenant condition as the rows.
	foreign, foreignTotal, err := repo.List(ctxOther, tenantOther, nil, 1, 10)
	if err != nil {
		t.Fatalf("List (foreign tenant): %v", err)
	}
	if foreignTotal != 1 || len(foreign) != 1 {
		t.Fatalf("List (foreign tenant): expected total=1 len=1, got total=%d len=%d", foreignTotal, len(foreign))
	}

	// Error path: page=0 drives a negative OFFSET, which Postgres rejects.
	if _, _, err := repo.List(ctxOwn, tenantOwn, nil, 0, 10); err == nil {
		t.Fatalf("List (page=0): expected an error from a negative OFFSET, got nil")
	}
}

func TestCampaignRepository_ListContacts_FiltersPaginatesAndScopesTotal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer ListContacts Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Dialer ListContacts Other Tenant")

	userID := seedDialerUser(t, pool, tenantOwn, "dialer-lc", "LC", "Owner")
	campID := seedDialerCampaign(t, pool, tenantOwn, userID, "LC Campaign", CampaignStatusActive)

	c1 := seedDialerContact(t, pool, tenantOwn, userID, "Pending", "One")
	c2 := seedDialerContact(t, pool, tenantOwn, userID, "Pending", "Two")
	c3 := seedDialerContact(t, pool, tenantOwn, userID, "Completed", "One")
	seedDialerCampaignContact(t, pool, tenantOwn, campID, c1, 1, ContactStatusPending)
	seedDialerCampaignContact(t, pool, tenantOwn, campID, c2, 2, ContactStatusPending)
	seedDialerCampaignContact(t, pool, tenantOwn, campID, c3, 3, ContactStatusCompleted)

	repo := NewPostgresCampaignRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	all, total, err := repo.ListContacts(ctxOwn, tenantOwn, campID, nil, 1, 10)
	if err != nil {
		t.Fatalf("ListContacts (no filter): %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Fatalf("ListContacts (no filter): expected total=3 len=3, got total=%d len=%d", total, len(all))
	}
	if all[0].Position != 1 || all[1].Position != 2 || all[2].Position != 3 {
		t.Fatalf("ListContacts: expected ascending position order, got %d,%d,%d", all[0].Position, all[1].Position, all[2].Position)
	}

	pending, total, err := repo.ListContacts(ctxOwn, tenantOwn, campID, strPtr(ContactStatusPending), 1, 10)
	if err != nil {
		t.Fatalf("ListContacts (status filter): %v", err)
	}
	if total != 2 || len(pending) != 2 {
		t.Fatalf("ListContacts (status filter): expected total=2 len=2, got total=%d len=%d", total, len(pending))
	}

	// Cross-tenant: a foreign tenantID against a real campaignID must see nothing.
	foreignRows, foreignTotal, err := repo.ListContacts(ctxOther, tenantOther, campID, nil, 1, 10)
	if err != nil {
		t.Fatalf("ListContacts (foreign tenant): %v", err)
	}
	if foreignTotal != 0 || len(foreignRows) != 0 {
		t.Fatalf("ListContacts (foreign tenant): expected total=0 len=0, got total=%d len=%d", foreignTotal, len(foreignRows))
	}

	// Error path: page=0 drives a negative OFFSET.
	if _, _, err := repo.ListContacts(ctxOwn, tenantOwn, campID, nil, 0, 10); err == nil {
		t.Fatalf("ListContacts (page=0): expected an error from a negative OFFSET, got nil")
	}
}

// ---------------------------------------------------------------------------
// AddContacts / GetNextPendingContact — queue write and claim logic
// ---------------------------------------------------------------------------

func TestCampaignRepository_AddContacts_SkipsDuplicatesAndRejectsForeignTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer AddContacts Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Dialer AddContacts Other Tenant")

	userID := seedDialerUser(t, pool, tenantOwn, "dialer-add", "Add", "Owner")
	campID := seedDialerCampaign(t, pool, tenantOwn, userID, "Add Campaign", CampaignStatusDraft)
	c1 := seedDialerContact(t, pool, tenantOwn, userID, "Add", "One")
	c2 := seedDialerContact(t, pool, tenantOwn, userID, "Add", "Two")

	repo := NewPostgresCampaignRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	added, skipped, err := repo.AddContacts(ctxOwn, tenantOwn, campID, []CampaignContact{
		{ID: uuid.New(), ContactID: c1, Position: 1},
		{ID: uuid.New(), ContactID: c2, Position: 2},
	})
	if err != nil {
		t.Fatalf("AddContacts (first batch): %v", err)
	}
	if added != 2 || skipped != 0 {
		t.Fatalf("AddContacts (first batch): expected added=2 skipped=0, got added=%d skipped=%d", added, skipped)
	}
	_, total, err := repo.ListContacts(ctxOwn, tenantOwn, campID, nil, 1, 10)
	if err != nil || total != 2 {
		t.Fatalf("ListContacts after first batch: total=%d err=%v", total, err)
	}

	// Re-adding c1 is a duplicate on (campaign_id, contact_id) and must be
	// skipped via ON CONFLICT DO NOTHING, not erred.
	c3 := seedDialerContact(t, pool, tenantOwn, userID, "Add", "Three")
	added, skipped, err = repo.AddContacts(ctxOwn, tenantOwn, campID, []CampaignContact{
		{ID: uuid.New(), ContactID: c1, Position: 3},
		{ID: uuid.New(), ContactID: c3, Position: 4},
	})
	if err != nil {
		t.Fatalf("AddContacts (duplicate batch): %v", err)
	}
	if added != 1 || skipped != 1 {
		t.Fatalf("AddContacts (duplicate batch): expected added=1 skipped=1, got added=%d skipped=%d", added, skipped)
	}
	_, total, _ = repo.ListContacts(ctxOwn, tenantOwn, campID, nil, 1, 10)
	if total != 3 {
		t.Fatalf("ListContacts after duplicate batch: expected total=3, got %d", total)
	}

	// Empty input is a documented no-op.
	added, skipped, err = repo.AddContacts(ctxOwn, tenantOwn, campID, nil)
	if err != nil || added != 0 || skipped != 0 {
		t.Fatalf("AddContacts (empty input): expected 0,0,nil, got %d,%d,%v", added, skipped, err)
	}

	// Error path: passing a tenantID that does not own campID makes the
	// tenant_id sub-select resolve to NULL, which the NOT NULL column rejects —
	// proving the defense-in-depth parameter is load-bearing, not decorative.
	c4 := seedDialerContact(t, pool, tenantOwn, userID, "Add", "Four")

	// AddContacts writes campaign_contacts rows outside of SeedRow, so this
	// sweep-by-campaign cleanup — registered last, after every contact
	// fixture above, so it runs FIRST under t.Cleanup's LIFO order — clears
	// every join row from all three AddContacts batches before the
	// per-contact cleanups run. Without it, a contact cleanup can fail on
	// the still-referencing contact_id FK and silently orphan rows
	// (CleanupRow only t.Logf's failures, matching every fixture helper in
	// this package), which is exactly what left two orphaned users rows in
	// the local dev DB before this fix.
	t.Cleanup(func() {
		sysCtx := testutil.WithSystemCtx(context.Background())
		if _, err := pool.Exec(sysCtx, "DELETE FROM dialer_campaign_contacts WHERE campaign_id = $1", campID); err != nil {
			t.Logf("cleanup dialer_campaign_contacts campaign_id=%s: %v", campID, err)
		}
	})

	_, _, err = repo.AddContacts(ctxOwn, tenantOther, campID, []CampaignContact{
		{ID: uuid.New(), ContactID: c4, Position: 5},
	})
	if err == nil {
		t.Fatalf("AddContacts (foreign tenantID param): expected a not-null violation, got nil")
	}
	_, total, _ = repo.ListContacts(ctxOwn, tenantOwn, campID, nil, 1, 10)
	if total != 3 {
		t.Fatalf("AddContacts (foreign tenantID param): row leaked in despite the error, total=%d", total)
	}
}

func TestCampaignRepository_GetNextPendingContact_ClaimsInPositionOrder(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer Queue Claim Tenant")

	userID := seedDialerUser(t, pool, tenantOwn, "dialer-claim", "Claim", "Owner")
	campID := seedDialerCampaign(t, pool, tenantOwn, userID, "Claim Campaign", CampaignStatusActive)

	c1 := seedDialerContact(t, pool, tenantOwn, userID, "Claim", "One")
	c2 := seedDialerContact(t, pool, tenantOwn, userID, "Claim", "Two")
	// Insert out of position order to prove the claim is ORDER BY position, not insertion order.
	cc2 := seedDialerCampaignContact(t, pool, tenantOwn, campID, c2, 2, ContactStatusPending)
	cc1 := seedDialerCampaignContact(t, pool, tenantOwn, campID, c1, 1, ContactStatusPending)

	repo := NewPostgresCampaignRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	first, err := repo.GetNextPendingContact(ctxOwn, campID)
	if err != nil {
		t.Fatalf("GetNextPendingContact (1st claim): %v", err)
	}
	if first.ID != cc1 {
		t.Fatalf("GetNextPendingContact (1st claim): expected position-1 row %s, got %s", cc1, first.ID)
	}
	if first.Status != ContactStatusInProgress {
		t.Fatalf("GetNextPendingContact (1st claim): expected status=in_progress, got %q", first.Status)
	}

	second, err := repo.GetNextPendingContact(ctxOwn, campID)
	if err != nil {
		t.Fatalf("GetNextPendingContact (2nd claim): %v", err)
	}
	if second.ID != cc2 {
		t.Fatalf("GetNextPendingContact (2nd claim): expected position-2 row %s, got %s", cc2, second.ID)
	}

	// Error path: the queue is now empty.
	if _, err := repo.GetNextPendingContact(ctxOwn, campID); err != ErrNoContactsAvailable {
		t.Fatalf("GetNextPendingContact (empty queue): expected ErrNoContactsAvailable, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetCampaignStats / UpdateCampaignCounts — aggregation
// ---------------------------------------------------------------------------

func TestCampaignRepository_GetCampaignStats_Aggregates(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer Stats Tenant")

	userID := seedDialerUser(t, pool, tenantOwn, "dialer-stats", "Stats", "Agent")
	campID := seedDialerCampaign(t, pool, tenantOwn, userID, "Stats Campaign", CampaignStatusActive)

	cPending := seedDialerContact(t, pool, tenantOwn, userID, "Stats", "Pending")
	cCompleted := seedDialerContact(t, pool, tenantOwn, userID, "Stats", "Completed")
	cSkipped := seedDialerContact(t, pool, tenantOwn, userID, "Stats", "Skipped")
	cCallback := seedDialerContact(t, pool, tenantOwn, userID, "Stats", "Callback")
	seedDialerCampaignContact(t, pool, tenantOwn, campID, cPending, 1, ContactStatusPending)
	ccCompleted := seedDialerCampaignContact(t, pool, tenantOwn, campID, cCompleted, 2, ContactStatusCompleted)
	seedDialerCampaignContact(t, pool, tenantOwn, campID, cSkipped, 3, ContactStatusSkipped)
	seedDialerCampaignContact(t, pool, tenantOwn, campID, cCallback, 4, ContactStatusCallback)

	outcomePos := seedDialerOutcome(t, pool, tenantOwn, "Reached", true, false)
	outcomeNeg := seedDialerOutcome(t, pool, tenantOwn, "Not Reached", false, false)
	dur1, dur2 := 100, 200
	seedDialerCallSession(t, pool, tenantOwn, ccCompleted, userID, &outcomePos, &dur1)
	seedDialerCallSession(t, pool, tenantOwn, ccCompleted, userID, &outcomeNeg, &dur2)

	repo := NewPostgresCampaignRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	stats, err := repo.GetCampaignStats(ctxOwn, tenantOwn, campID)
	if err != nil {
		t.Fatalf("GetCampaignStats: %v", err)
	}
	if stats.TotalContacts != 4 || stats.PendingContacts != 1 || stats.CompletedContacts != 1 ||
		stats.SkippedContacts != 1 || stats.CallbackContacts != 1 {
		t.Fatalf("GetCampaignStats contact breakdown mismatch: %+v", stats)
	}
	if stats.TotalCalls != 2 || stats.PositiveCalls != 1 {
		t.Fatalf("GetCampaignStats call breakdown mismatch: total=%d positive=%d", stats.TotalCalls, stats.PositiveCalls)
	}
	if stats.AvgDurationSecs != 150 {
		t.Fatalf("GetCampaignStats: expected avg duration 150, got %v", stats.AvgDurationSecs)
	}
	if len(stats.OutcomeBreakdown) != 2 {
		t.Fatalf("GetCampaignStats: expected 2 outcome buckets, got %d", len(stats.OutcomeBreakdown))
	}

	// UpdateCampaignCounts must recompute the denormalized counters from the
	// same live rows GetCampaignStats just aggregated.
	if err := repo.UpdateCampaignCounts(ctxOwn, campID); err != nil {
		t.Fatalf("UpdateCampaignCounts: %v", err)
	}
	camp, err := repo.GetByIDForTenant(ctxOwn, campID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByIDForTenant after UpdateCampaignCounts: %v", err)
	}
	if camp.ContactCount != 4 || camp.CompletedCount != 1 {
		t.Fatalf("UpdateCampaignCounts: expected contact_count=4 completed_count=1, got %d/%d", camp.ContactCount, camp.CompletedCount)
	}
}

// ---------------------------------------------------------------------------
// CallRepository — session lifecycle, atomic transaction, events, aggregates
// ---------------------------------------------------------------------------

func dialerCallFixture(t *testing.T, pool *pgxpool.Pool) (tenantID, campID, ccID, agentID uuid.UUID, ctxOwn context.Context) {
	t.Helper()
	tenantID = uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dialer Call Fixture Tenant")
	agentID = seedDialerUser(t, pool, tenantID, "dialer-call-"+tenantID.String(), "Call", "Agent")
	campID = seedDialerCampaign(t, pool, tenantID, agentID, "Call Fixture Campaign", CampaignStatusActive)
	contactID := seedDialerContact(t, pool, tenantID, agentID, "Call", "Contact")
	ccID = seedDialerCampaignContact(t, pool, tenantID, campID, contactID, 1, ContactStatusInProgress)
	ctxOwn = testutil.WithTenantCtx(context.Background(), tenantID)
	return
}

func TestCallRepository_SessionLifecycle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn, campID, ccID, agentID, ctxOwn := dialerCallFixture(t, pool)
	_ = campID

	repo := NewPostgresCallRepository(pool)
	now := time.Now().UTC()
	session := &CallSession{
		ID: uuid.New(), TenantID: tenantOwn, CampaignContactID: ccID, AgentID: agentID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSession(ctxOwn, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_call_sessions", session.ID) })

	got, err := repo.GetSessionByID(ctxOwn, session.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetSessionByID: %v", err)
	}
	if got.TenantID != tenantOwn || got.CampaignContactID != ccID {
		t.Fatalf("GetSessionByID: unexpected row %+v", got)
	}

	// Error path: wrong tenantID must not resolve the row.
	if _, err := repo.GetSessionByID(ctxOwn, session.ID, uuid.New()); err != ErrCallSessionNotFound {
		t.Fatalf("GetSessionByID (foreign tenantID): expected ErrCallSessionNotFound, got %v", err)
	}

	notes := "updated notes"
	got.Notes = &notes
	got.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateSession(ctxOwn, got); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}
	reloaded, _ := repo.GetSessionByID(ctxOwn, session.ID, tenantOwn)
	if reloaded.Notes == nil || *reloaded.Notes != notes {
		t.Fatalf("UpdateSession did not persist notes: %+v", reloaded)
	}

	// Error path: updating a non-existent session ID.
	ghost := *got
	ghost.ID = uuid.New()
	if err := repo.UpdateSession(ctxOwn, &ghost); err != ErrCallSessionNotFound {
		t.Fatalf("UpdateSession (ghost id): expected ErrCallSessionNotFound, got %v", err)
	}
}

func TestCallRepository_UpdateSessionWithEventAndContact_Atomic(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn, _, ccID, agentID, ctxOwn := dialerCallFixture(t, pool)

	repo := NewPostgresCallRepository(pool)
	now := time.Now().UTC()
	session := &CallSession{
		ID: uuid.New(), TenantID: tenantOwn, CampaignContactID: ccID, AgentID: agentID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSession(ctxOwn, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_call_sessions", session.ID) })

	outcomeID := seedDialerOutcome(t, pool, tenantOwn, "Atomic Reached", true, false)
	dur := 42
	session.OutcomeID = &outcomeID
	session.DurationSeconds = &dur
	session.UpdatedAt = time.Now().UTC()
	event := &CallEvent{
		ID: uuid.New(), DialerCallSessionID: session.ID, EventType: CallEventOutcomeLogged,
		Payload: map[string]any{"k": "v"}, OccurredAt: time.Now().UTC(),
	}

	if err := repo.UpdateSessionWithEventAndContact(ctxOwn, session, event, ccID, ContactStatusCompleted, &outcomeID, nil); err != nil {
		t.Fatalf("UpdateSessionWithEventAndContact (success): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_call_events", event.ID) })

	reloaded, _ := repo.GetSessionByID(ctxOwn, session.ID, tenantOwn)
	if reloaded.OutcomeID == nil || *reloaded.OutcomeID != outcomeID {
		t.Fatalf("UpdateSessionWithEventAndContact: session outcome not persisted: %+v", reloaded)
	}
	events, err := repo.ListEventsBySession(ctxOwn, session.ID)
	if err != nil || len(events) != 1 {
		t.Fatalf("ListEventsBySession: expected 1 event, got %d (err=%v)", len(events), err)
	}

	campRepo := NewPostgresCampaignRepository(pool)
	cc, err := campRepo.GetCampaignContactByID(ctxOwn, ccID, tenantOwn)
	if err != nil {
		t.Fatalf("GetCampaignContactByID: %v", err)
	}
	if cc.Status != ContactStatusCompleted {
		t.Fatalf("UpdateSessionWithEventAndContact: contact status not updated, got %q", cc.Status)
	}

	// Failure path: the event's DialerCallSessionID does not exist, so the
	// event insert's tenant_id sub-select is NULL and violates the NOT NULL
	// column — proving the session update rolls back rather than partially
	// committing (a bare BEGIN/COMMIT-per-statement bug would leave the
	// session's notes persisted here).
	preFailureNotes := "before rollback probe"
	session.Notes = &preFailureNotes
	session.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateSession(ctxOwn, session); err != nil {
		t.Fatalf("UpdateSession (seed notes before rollback probe): %v", err)
	}

	rollbackNotes := "should never be visible"
	session.Notes = &rollbackNotes
	session.UpdatedAt = time.Now().UTC()
	badEvent := &CallEvent{
		ID: uuid.New(), DialerCallSessionID: uuid.New(), EventType: CallEventFailed,
		Payload: map[string]any{}, OccurredAt: time.Now().UTC(),
	}
	err = repo.UpdateSessionWithEventAndContact(ctxOwn, session, badEvent, ccID, ContactStatusCompleted, &outcomeID, nil)
	if err == nil {
		t.Fatalf("UpdateSessionWithEventAndContact (mismatched event session): expected an error, got nil")
	}
	reloaded, _ = repo.GetSessionByID(ctxOwn, session.ID, tenantOwn)
	if reloaded.Notes == nil || *reloaded.Notes != preFailureNotes {
		t.Fatalf("UpdateSessionWithEventAndContact: session update was not rolled back, notes=%v", reloaded.Notes)
	}
	events, _ = repo.ListEventsBySession(ctxOwn, session.ID)
	if len(events) != 1 {
		t.Fatalf("UpdateSessionWithEventAndContact: rolled-back event still visible, count=%d", len(events))
	}
}

func TestCallRepository_AppendEvent_And_ListEventsBySession(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn, _, ccID, agentID, ctxOwn := dialerCallFixture(t, pool)

	repo := NewPostgresCallRepository(pool)
	now := time.Now().UTC()
	session := &CallSession{
		ID: uuid.New(), TenantID: tenantOwn, CampaignContactID: ccID, AgentID: agentID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSession(ctxOwn, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_call_sessions", session.ID) })

	e1 := &CallEvent{ID: uuid.New(), DialerCallSessionID: session.ID, EventType: CallEventInitiated, Payload: map[string]any{}, OccurredAt: now}
	e2 := &CallEvent{ID: uuid.New(), DialerCallSessionID: session.ID, EventType: CallEventAnswered, Payload: map[string]any{"x": 1.0}, OccurredAt: now.Add(time.Second)}
	if err := repo.AppendEvent(ctxOwn, e1); err != nil {
		t.Fatalf("AppendEvent (1): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_call_events", e1.ID) })
	if err := repo.AppendEvent(ctxOwn, e2); err != nil {
		t.Fatalf("AppendEvent (2): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_call_events", e2.ID) })

	events, err := repo.ListEventsBySession(ctxOwn, session.ID)
	if err != nil {
		t.Fatalf("ListEventsBySession: %v", err)
	}
	if len(events) != 2 || events[0].EventType != CallEventInitiated || events[1].EventType != CallEventAnswered {
		t.Fatalf("ListEventsBySession: expected chronological [initiated, answered], got %+v", events)
	}
	if events[1].Payload["x"] != 1.0 {
		t.Fatalf("ListEventsBySession: payload roundtrip mismatch: %+v", events[1].Payload)
	}

	// Error path: a session ID that does not exist leaves the tenant_id
	// sub-select NULL, which the NOT NULL column rejects.
	orphan := &CallEvent{ID: uuid.New(), DialerCallSessionID: uuid.New(), EventType: CallEventFailed, Payload: map[string]any{}, OccurredAt: now}
	if err := repo.AppendEvent(ctxOwn, orphan); err == nil {
		t.Fatalf("AppendEvent (orphan session): expected a not-null violation, got nil")
	}
}

func TestCallRepository_TodayCountsAndAverages(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn, campID, _, agent1, ctxOwn := dialerCallFixture(t, pool)
	agent2 := seedDialerUser(t, pool, tenantOwn, "dialer-agent2", "Agent", "Two")

	contactA := seedDialerContact(t, pool, tenantOwn, agent1, "Today", "A")
	contactB := seedDialerContact(t, pool, tenantOwn, agent1, "Today", "B")
	ccA := seedDialerCampaignContact(t, pool, tenantOwn, campID, contactA, 2, ContactStatusCompleted)
	ccB := seedDialerCampaignContact(t, pool, tenantOwn, campID, contactB, 3, ContactStatusCompleted)

	appointmentOutcome := seedDialerOutcome(t, pool, tenantOwn, "Today Appointment", true, true)
	plainOutcome := seedDialerOutcome(t, pool, tenantOwn, "Today Plain", true, false)
	dur1, dur2 := 60, 120

	seedDialerCallSession(t, pool, tenantOwn, ccA, agent1, &appointmentOutcome, &dur1)
	seedDialerCallSession(t, pool, tenantOwn, ccB, agent1, &plainOutcome, &dur2)
	seedDialerCallSession(t, pool, tenantOwn, ccA, agent2, &plainOutcome, &dur2)

	repo := NewPostgresCallRepository(pool)

	tenantCount, err := repo.GetTenantCallsTodayCount(ctxOwn, tenantOwn)
	if err != nil || tenantCount != 3 {
		t.Fatalf("GetTenantCallsTodayCount: expected 3, got %d (err=%v)", tenantCount, err)
	}
	apptCount, err := repo.GetTenantAppointmentsTodayCount(ctxOwn, tenantOwn)
	if err != nil || apptCount != 1 {
		t.Fatalf("GetTenantAppointmentsTodayCount: expected 1, got %d (err=%v)", apptCount, err)
	}
	agent1Count, err := repo.GetAgentCallsTodayCount(ctxOwn, agent1)
	if err != nil || agent1Count != 2 {
		t.Fatalf("GetAgentCallsTodayCount (agent1): expected 2, got %d (err=%v)", agent1Count, err)
	}
	agent2Count, err := repo.GetAgentCallsTodayCount(ctxOwn, agent2)
	if err != nil || agent2Count != 1 {
		t.Fatalf("GetAgentCallsTodayCount (agent2): expected 1, got %d (err=%v)", agent2Count, err)
	}
	avg, err := repo.GetAgentAvgDurationToday(ctxOwn, agent1)
	if err != nil || avg != 90 {
		t.Fatalf("GetAgentAvgDurationToday (agent1): expected 90, got %v (err=%v)", avg, err)
	}

	// Tenant isolation: a freshly minted tenant with no sessions must see zero.
	emptyTenant := uuid.New()
	testutil.EnsureTenant(t, pool, emptyTenant, "Dialer Today Empty Tenant")
	emptyCtx := testutil.WithTenantCtx(context.Background(), emptyTenant)
	emptyCount, err := repo.GetTenantCallsTodayCount(emptyCtx, emptyTenant)
	if err != nil || emptyCount != 0 {
		t.Fatalf("GetTenantCallsTodayCount (empty tenant): expected 0, got %d (err=%v)", emptyCount, err)
	}
}

func TestCallRepository_GetRecentCallsForTenant_And_ListCallsByContact(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn, _, ccID, agentID, ctxOwn := dialerCallFixture(t, pool)

	outcomeID := seedDialerOutcome(t, pool, tenantOwn, "Recent Reached", true, false)
	dur := 77
	seedDialerCallSession(t, pool, tenantOwn, ccID, agentID, &outcomeID, &dur)

	repo := NewPostgresCallRepository(pool)

	recent, err := repo.GetRecentCallsForTenant(ctxOwn, tenantOwn, 10)
	if err != nil {
		t.Fatalf("GetRecentCallsForTenant: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("GetRecentCallsForTenant: expected 1 row, got %d", len(recent))
	}
	if recent[0].ContactName != "Call Contact" || recent[0].AgentName != "Call Agent" {
		t.Fatalf("GetRecentCallsForTenant: display data mismatch: %+v", recent[0])
	}
	if recent[0].OutcomeLabel == nil || *recent[0].OutcomeLabel != "Recent Reached" {
		t.Fatalf("GetRecentCallsForTenant: outcome label mismatch: %+v", recent[0])
	}

	// limit<=0 defaults to 20 rather than returning zero rows.
	defaulted, err := repo.GetRecentCallsForTenant(ctxOwn, tenantOwn, 0)
	if err != nil || len(defaulted) != 1 {
		t.Fatalf("GetRecentCallsForTenant (limit=0): expected the same 1 row via the default limit, got %d (err=%v)", len(defaulted), err)
	}

	byContact, err := repo.ListCallsByContact(ctxOwn, tenantOwn, ccID)
	if err != nil || len(byContact) != 1 {
		t.Fatalf("ListCallsByContact: expected 1 row, got %d (err=%v)", len(byContact), err)
	}
	if byContact[0].DurationSecs != dur {
		t.Fatalf("ListCallsByContact: duration mismatch, got %d want %d", byContact[0].DurationSecs, dur)
	}

	// Cross-tenant: a foreign tenantID must see nothing from either read,
	// even though the campaign-contact ID is real.
	foreignTenant := uuid.New()
	testutil.EnsureTenant(t, pool, foreignTenant, "Dialer Recent Foreign Tenant")
	foreignCtx := testutil.WithTenantCtx(context.Background(), foreignTenant)
	foreignRecent, err := repo.GetRecentCallsForTenant(foreignCtx, foreignTenant, 10)
	if err != nil || len(foreignRecent) != 0 {
		t.Fatalf("GetRecentCallsForTenant (foreign tenant): expected 0 rows, got %d (err=%v)", len(foreignRecent), err)
	}
	foreignByContact, err := repo.ListCallsByContact(foreignCtx, foreignTenant, ccID)
	if err != nil || len(foreignByContact) != 0 {
		t.Fatalf("ListCallsByContact (foreign tenant): expected 0 rows, got %d (err=%v)", len(foreignByContact), err)
	}
}

// ---------------------------------------------------------------------------
// OutcomeRepository.List / EnsureDefaults
// ---------------------------------------------------------------------------

func TestOutcomeRepository_List_And_EnsureDefaults(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer Outcome Defaults Tenant")

	repo := NewPostgresOutcomeRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	if err := repo.EnsureDefaults(ctxOwn, tenantOwn); err != nil {
		t.Fatalf("EnsureDefaults (first call): %v", err)
	}
	defaults, err := repo.List(ctxOwn, tenantOwn, false)
	if err != nil || len(defaults) != 4 {
		t.Fatalf("List after EnsureDefaults: expected 4 rows, got %d (err=%v)", len(defaults), err)
	}
	for _, o := range defaults {
		t.Cleanup(func(id uuid.UUID) func() {
			return func() { testutil.CleanupRow(t, pool, "dialer_call_outcomes", id) }
		}(o.ID))
	}
	if defaults[0].SortOrder > defaults[len(defaults)-1].SortOrder {
		t.Fatalf("List: expected ascending sort_order, got %+v", defaults)
	}

	// EnsureDefaults is a documented no-op once outcomes already exist —
	// calling it again must not duplicate the seed set.
	if err := repo.EnsureDefaults(ctxOwn, tenantOwn); err != nil {
		t.Fatalf("EnsureDefaults (second call): %v", err)
	}
	again, err := repo.List(ctxOwn, tenantOwn, false)
	if err != nil || len(again) != 4 {
		t.Fatalf("List after repeated EnsureDefaults: expected still 4 rows, got %d (err=%v)", len(again), err)
	}

	// includeInactive=false must exclude a deactivated outcome; true includes it.
	inactive := defaults[0]
	inactive.IsActive = false
	if err := repo.Update(ctxOwn, inactive); err != nil {
		t.Fatalf("Update (deactivate): %v", err)
	}
	activeOnly, err := repo.List(ctxOwn, tenantOwn, false)
	if err != nil || len(activeOnly) != 3 {
		t.Fatalf("List (includeInactive=false): expected 3 rows, got %d (err=%v)", len(activeOnly), err)
	}
	withInactive, err := repo.List(ctxOwn, tenantOwn, true)
	if err != nil || len(withInactive) != 4 {
		t.Fatalf("List (includeInactive=true): expected 4 rows, got %d (err=%v)", len(withInactive), err)
	}
}

// ---------------------------------------------------------------------------
// AgentStatusRepository — active-agent enumeration and display-name lookup
// ---------------------------------------------------------------------------

func TestAgentStatusRepository_GetActiveAgentIDsForTenant_And_GetUserDisplayNames(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer Agent Status Read Tenant")

	activeAgent := seedDialerUser(t, pool, tenantOwn, "dialer-active-agent", "Active", "Agent")
	idleAgent := seedDialerUser(t, pool, tenantOwn, "dialer-idle-agent", "Idle", "Agent")

	statusRepo := NewPostgresAgentStatusRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	sysCtx := testutil.WithSystemCtx(context.Background())

	entry := &AgentStatusLogEntry{ID: uuid.New(), UserID: activeAgent, Status: AgentStatusAvailable, ChangedAt: time.Now().UTC()}
	if err := statusRepo.LogStatusChange(ctxOwn, entry); err != nil {
		t.Fatalf("LogStatusChange: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_agent_status_log", entry.ID) })

	ids, err := statusRepo.GetActiveAgentIDsForTenant(sysCtx, tenantOwn)
	if err != nil {
		t.Fatalf("GetActiveAgentIDsForTenant: %v", err)
	}
	if len(ids) != 1 || ids[0] != activeAgent {
		t.Fatalf("GetActiveAgentIDsForTenant: expected [%s], got %v", activeAgent, ids)
	}

	// A tenant with no logged status changes today must see nothing, even
	// though idleAgent exists under the same tenant.
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "Dialer Agent Status Empty Tenant")
	emptyIDs, err := statusRepo.GetActiveAgentIDsForTenant(sysCtx, otherTenant)
	if err != nil || len(emptyIDs) != 0 {
		t.Fatalf("GetActiveAgentIDsForTenant (empty tenant): expected 0, got %d (err=%v)", len(emptyIDs), err)
	}

	names, err := statusRepo.GetUserDisplayNames(sysCtx, []uuid.UUID{activeAgent, idleAgent, uuid.New()})
	if err != nil {
		t.Fatalf("GetUserDisplayNames: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("GetUserDisplayNames: expected 2 resolved entries (missing id silently omitted), got %d: %+v", len(names), names)
	}
	if got := names[activeAgent]; got[0] != "Active" || got[1] != "Agent" {
		t.Fatalf("GetUserDisplayNames: unexpected name for activeAgent: %+v", got)
	}

	// Empty input is a documented early return.
	empty, err := statusRepo.GetUserDisplayNames(sysCtx, nil)
	if err != nil || empty != nil {
		t.Fatalf("GetUserDisplayNames (empty input): expected nil,nil, got %v,%v", empty, err)
	}
}
