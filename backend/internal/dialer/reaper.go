package dialer

import (
	"context"
	"log/slog"
	"time"

	"github.com/kmuhub/kmuhub/internal/safego"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// staleReapBatch caps one sweep; a backlog larger than this drains over the next ticks.
const staleReapBatch = 500

// RequeueStaleContacts puts campaign contacts back in the queue that have been
// in_progress since before now-staleAfter with no open call session newer than
// that. GetNextPendingContact flips a contact to in_progress on claim; if the
// agent's browser dies before LogCallOutcome/CompleteWrapUp, nothing else ever
// resets it. Returns the number of contacts requeued.
//
// It runs across all tenants (sysctx) and requeues through RequeueContact, which
// still filters on the row's own tenant_id. A per-contact error is logged and
// does not stop the sweep.
func (s *Service) RequeueStaleContacts(ctx context.Context, staleAfter time.Duration) (int, error) {
	ctx = sysctx.With(ctx)
	cutoff := time.Now().Add(-staleAfter)

	stale, err := s.campaigns.ListStaleInProgressContacts(ctx, cutoff, staleReapBatch)
	if err != nil {
		return 0, err
	}

	requeued := 0
	for _, sc := range stale {
		// The list may be seconds old: an agent can have logged an outcome since.
		// RequeueContact does not gate on status, so re-read before resetting.
		cc, err := s.campaigns.GetCampaignContactByID(ctx, sc.ID, sc.TenantID)
		if err != nil {
			slog.WarnContext(ctx, "dialer: stale-contact recheck failed", "campaign_contact_id", sc.ID, "error", err)
			continue
		}
		if cc.Status != ContactStatusInProgress || !cc.UpdatedAt.Before(cutoff) {
			continue
		}
		if err := s.RequeueContact(ctx, sc.ID, cc.CampaignID, sc.TenantID); err != nil {
			slog.WarnContext(ctx, "dialer: stale-contact requeue failed", "campaign_contact_id", sc.ID, "error", err)
			continue
		}
		requeued++
	}
	if requeued > 0 {
		slog.InfoContext(ctx, "dialer: requeued stale in-progress contacts", "count", requeued, "stale_after", staleAfter.String())
	}
	return requeued, nil
}

// RunStaleContactReaper sweeps every interval until ctx is cancelled. Errors are
// logged, never propagated -- a failed sweep must not take the dialer down.
// lean: no advisory lock; a second replica's sweep finds nothing left to requeue.
func RunStaleContactReaper(ctx context.Context, svc *Service, staleAfter, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			safego.Do("dialer stale-contact sweep", func() {
				if _, err := svc.RequeueStaleContacts(ctx, staleAfter); err != nil {
					slog.ErrorContext(ctx, "dialer: stale-contact sweep failed", "error", err)
				}
			})
		}
	}
}
