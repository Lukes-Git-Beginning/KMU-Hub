package dialer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestRequeueStaleContacts_Integration proves the reaper requeues a contact
// abandoned in_progress, leaves alone one with a live session, a fresh claim and
// a non-in_progress row, and reaches across tenants (the sweep is system-wide).
func TestRequeueStaleContacts_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close) // registered first, so it runs after every row cleanup below

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Dialer Reaper Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Dialer Reaper Tenant B")

	sysCtx := testutil.WithSystemCtx(context.Background())
	repo := NewPostgresCampaignRepository(pool)
	svc := NewService(repo, nil, nil, nil, nil, nil, nil)

	old := time.Now().Add(-3 * time.Hour)
	fresh := time.Now().Add(-1 * time.Minute)

	type fixture struct {
		tenant   uuid.UUID
		user     uuid.UUID
		campaign uuid.UUID
		pos      int
	}
	seed := func(tenant uuid.UUID) *fixture {
		f := &fixture{tenant: tenant}
		f.user = testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "email": "dialer-reaper-" + uuid.NewString() + "@test.local", "password_hash": "$2a$10$placeholder",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", f.user) })
		f.campaign = testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "name": "Reaper Campaign", "created_by": f.user,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_campaigns", f.campaign) })
		return f
	}
	// Cleanups run LIFO: queue rows (and their sessions) are registered after the
	// parents above, so they are removed first.
	queueRow := func(f *fixture, status string, updatedAt time.Time) uuid.UUID {
		f.pos++
		// uq_campaign_contact: one queue row per (campaign, contact).
		contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
			"id": uuid.New(), "tenant_id": f.tenant, "first_name": "Reaper", "last_name": "Test", "created_by": f.user,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })
		id := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
			"id": uuid.New(), "tenant_id": f.tenant, "campaign_id": f.campaign, "contact_id": contactID,
			"position": f.pos, "status": status, "updated_at": updatedAt,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_campaign_contacts", id) })
		return id
	}
	session := func(f *fixture, ccID uuid.UUID, updatedAt time.Time) {
		id := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
			"id": uuid.New(), "tenant_id": f.tenant, "campaign_contact_id": ccID, "agent_id": f.user,
			"created_at": updatedAt, "updated_at": updatedAt,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_call_sessions", id) })
	}

	fa := seed(tenantA)
	fb := seed(tenantB)

	abandonedA := queueRow(fa, ContactStatusInProgress, old)
	abandonedB := queueRow(fb, ContactStatusInProgress, old)
	deadSession := queueRow(fa, ContactStatusInProgress, old) // session exists but is as old as the claim
	session(fa, deadSession, old)
	liveSession := queueRow(fa, ContactStatusInProgress, old) // claim is old, session is live
	session(fa, liveSession, fresh)
	freshClaim := queueRow(fa, ContactStatusInProgress, fresh)
	completed := queueRow(fa, ContactStatusCompleted, old)

	n, err := svc.RequeueStaleContacts(sysCtx, time.Hour)
	if err != nil {
		t.Fatalf("RequeueStaleContacts: %v", err)
	}
	if n < 3 {
		t.Fatalf("requeued %d contacts, want at least the 3 abandoned ones", n)
	}

	status := func(f *fixture, id uuid.UUID) string {
		cc, err := repo.GetCampaignContactByID(sysCtx, id, f.tenant)
		if err != nil {
			t.Fatalf("GetCampaignContactByID: %v", err)
		}
		return cc.Status
	}
	for name, c := range map[string]struct {
		f    *fixture
		id   uuid.UUID
		want string
	}{
		"abandoned (tenant A)":      {fa, abandonedA, ContactStatusPending},
		"abandoned (tenant B)":      {fb, abandonedB, ContactStatusPending},
		"session as old as claim":   {fa, deadSession, ContactStatusPending},
		"live session":              {fa, liveSession, ContactStatusInProgress},
		"fresh claim":               {fa, freshClaim, ContactStatusInProgress},
		"completed stays completed": {fa, completed, ContactStatusCompleted},
	} {
		if got := status(c.f, c.id); got != c.want {
			t.Errorf("%s: status = %q, want %q", name, got, c.want)
		}
	}
}

func TestRunStaleContactReaper_StopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// A cancelled ctx must end the loop; the test would hang otherwise.
	RunStaleContactReaper(ctx, NewService(&mockCampaignRepo{}, nil, nil, nil, nil, nil, nil), time.Hour, time.Hour)
}

func TestRequeueStaleContacts_NothingStale(t *testing.T) {
	n, err := NewService(&mockCampaignRepo{}, nil, nil, nil, nil, nil, nil).RequeueStaleContacts(context.Background(), time.Hour)
	if err != nil || n != 0 {
		t.Fatalf("got (%d, %v), want (0, nil)", n, err)
	}
}
