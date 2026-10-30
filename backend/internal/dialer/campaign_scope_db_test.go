package dialer

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestQueueActions_RequireTheirOwnCampaign pins that skip and requeue only
// reach a queue entry through the campaign it belongs to. The REST routes are
// /dialer/campaigns/{id}/contacts/{cid}/skip|requeue; before the fix the
// campaign id was dropped, so an entry of campaign A could be skipped or
// requeued through the path of campaign B.
func TestQueueActions_RequireTheirOwnCampaign(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dialer Campaign Scope Tenant")

	userID := seedDialerUser(t, pool, tenantID, "dialer-scope", "Scope", "Owner")
	campA := seedDialerCampaign(t, pool, tenantID, userID, "Scope Campaign A", CampaignStatusActive)
	campB := seedDialerCampaign(t, pool, tenantID, userID, "Scope Campaign B", CampaignStatusActive)
	// One contact per subtest: a contact sits in a campaign queue only once.
	skipContact := seedDialerContact(t, pool, tenantID, userID, "Skip", "Caller")
	requeueContact := seedDialerContact(t, pool, tenantID, userID, "Requeue", "Caller")

	repo := NewPostgresCampaignRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	statusOf := func(t *testing.T, id uuid.UUID) string {
		t.Helper()
		cc, err := repo.GetCampaignContactByID(ctx, id, tenantID)
		if err != nil {
			t.Fatalf("GetCampaignContactByID: %v", err)
		}
		return cc.Status
	}

	t.Run("skip", func(t *testing.T) {
		ccID := seedDialerCampaignContact(t, pool, tenantID, campA, skipContact, 1, ContactStatusPending)

		err := repo.SkipContact(ctx, ccID, campB, tenantID)
		if !errors.Is(err, ErrCampaignContactNotFound) {
			t.Fatalf("skip via campaign B: expected ErrCampaignContactNotFound, got %v", err)
		}
		if got := statusOf(t, ccID); got != ContactStatusPending {
			t.Fatalf("skip via campaign B changed the entry: status=%q", got)
		}

		if err := repo.SkipContact(ctx, ccID, campA, tenantID); err != nil {
			t.Fatalf("skip via campaign A: %v", err)
		}
		if got := statusOf(t, ccID); got != ContactStatusSkipped {
			t.Fatalf("skip via campaign A did not land: status=%q", got)
		}
	})

	t.Run("requeue", func(t *testing.T) {
		ccID := seedDialerCampaignContact(t, pool, tenantID, campA, requeueContact, 2, ContactStatusSkipped)

		err := repo.RequeueContact(ctx, ccID, campB, tenantID)
		if !errors.Is(err, ErrCampaignContactNotFound) {
			t.Fatalf("requeue via campaign B: expected ErrCampaignContactNotFound, got %v", err)
		}
		if got := statusOf(t, ccID); got != ContactStatusSkipped {
			t.Fatalf("requeue via campaign B changed the entry: status=%q", got)
		}

		if err := repo.RequeueContact(ctx, ccID, campA, tenantID); err != nil {
			t.Fatalf("requeue via campaign A: %v", err)
		}
		if got := statusOf(t, ccID); got != ContactStatusPending {
			t.Fatalf("requeue via campaign A did not land: status=%q", got)
		}
	})
}
