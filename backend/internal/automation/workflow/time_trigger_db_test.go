package workflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestListActiveTimeBased_FiltersByProvidedTypes exercises the fix behind
// backlog unit g-automation-cron-poller: the previous query filtered on the
// literal string "time_based", which no TriggerDefinition.Type is ever set
// to (validateAutomation rejects any trigger_type not present in the
// registry) -- ListActiveTimeBased always returned zero rows, so
// trigger.TimeTriggerPoller never actually fired biz.invoice.overdue or
// calendar.event.upcoming automations. The caller now supplies the concrete
// TimeBased-flagged type strings.
func TestListActiveTimeBased_FiltersByProvidedTypes(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "ListActiveTimeBased Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("time-trigger-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	timeBased := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id":    tenantID,
		"name":         "Overdue invoices",
		"description":  "",
		"owner_id":     userID,
		"trigger_type": "biz.invoice.overdue",
		"is_active":    true,
	})
	defer testutil.CleanupRow(t, pool, "automations", timeBased)

	notTimeBased := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id":    tenantID,
		"name":         "Deal stage changed",
		"description":  "",
		"owner_id":     userID,
		"trigger_type": "crm.deal.stage_changed",
		"is_active":    true,
	})
	defer testutil.CleanupRow(t, pool, "automations", notTimeBased)

	inactiveTimeBased := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id":    tenantID,
		"name":         "Overdue invoices (disabled)",
		"description":  "",
		"owner_id":     userID,
		"trigger_type": "biz.invoice.overdue",
		"is_active":    false,
	})
	defer testutil.CleanupRow(t, pool, "automations", inactiveTimeBased)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	// Empty type list -- the poller never has this, but the repo must not
	// interpret it as "match everything" (ANY('{}') would already return
	// zero rows in Postgres; the short-circuit just avoids the round trip).
	empty, err := repo.ListActiveTimeBased(ctx, nil)
	if err != nil {
		t.Fatalf("ListActiveTimeBased(nil): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 automations for a nil type list, got %d", len(empty))
	}

	got, err := repo.ListActiveTimeBased(ctx, []string{"biz.invoice.overdue", "calendar.event.upcoming"})
	if err != nil {
		t.Fatalf("ListActiveTimeBased: %v", err)
	}

	var foundOverdue, foundOther, foundInactive bool
	for _, a := range got {
		switch a.ID {
		case timeBased:
			foundOverdue = true
			if a.LastPolledAt != nil {
				t.Fatalf("freshly seeded automation should have last_polled_at = NULL, got %v", a.LastPolledAt)
			}
		case notTimeBased:
			foundOther = true
		case inactiveTimeBased:
			foundInactive = true
		}
	}
	if !foundOverdue {
		t.Fatal("expected the biz.invoice.overdue automation to be returned")
	}
	if foundOther {
		t.Fatal("crm.deal.stage_changed is not in the requested type list and must not be returned")
	}
	if foundInactive {
		t.Fatal("inactive automations must not be returned even if the trigger type matches")
	}
}

// TestClaimTimeTrigger_AtomicClaim verifies the optimistic-concurrency claim
// that replaces poller.go's former history-query dedup (wasRecentlyExecuted):
// a claim only succeeds if last_polled_at still matches the value the caller
// last observed, so two racing pollers reading the same automation cannot
// both win.
func TestClaimTimeTrigger_AtomicClaim(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "ClaimTimeTrigger Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("claim-trigger-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	autoID := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id":    tenantID,
		"name":         "Claim test automation",
		"description":  "",
		"owner_id":     userID,
		"trigger_type": "biz.invoice.overdue",
		"is_active":    true,
	})
	defer testutil.CleanupRow(t, pool, "automations", autoID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	t1 := time.Now().UTC().Truncate(time.Microsecond)

	// A second instance that read the row before either claim landed would
	// also observe previousLastPolledAt = nil. Simulate it racing in after
	// the first claim already won: it must lose.
	claimedFirst, err := repo.ClaimTimeTrigger(ctx, autoID, nil, t1)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if !claimedFirst {
		t.Fatal("first claim against a NULL last_polled_at must succeed")
	}

	claimedRace, err := repo.ClaimTimeTrigger(ctx, autoID, nil, t1)
	if err != nil {
		t.Fatalf("racing claim: %v", err)
	}
	if claimedRace {
		t.Fatal("a second claim against the now-stale nil value must lose the race")
	}

	// The next tick reads the row again (now last_polled_at = t1) and claims
	// with the correct previous value -- this must succeed.
	t2 := t1.Add(5 * time.Minute)
	claimedSecondTick, err := repo.ClaimTimeTrigger(ctx, autoID, &t1, t2)
	if err != nil {
		t.Fatalf("second-tick claim: %v", err)
	}
	if !claimedSecondTick {
		t.Fatal("a claim presenting the correct current last_polled_at must succeed")
	}

	// A stale previous value (still t1, after the row has moved to t2) must
	// now fail -- this is the exact shape of two overlapping ticks racing.
	claimedStale, err := repo.ClaimTimeTrigger(ctx, autoID, &t1, t2.Add(time.Minute))
	if err != nil {
		t.Fatalf("stale claim: %v", err)
	}
	if claimedStale {
		t.Fatal("a claim presenting a stale last_polled_at must not succeed")
	}
}
