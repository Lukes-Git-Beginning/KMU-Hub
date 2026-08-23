package recurring

// Real-SQL tests for Service.Generate — the only place in the product that
// creates a money document without a user click. Unlike service_test.go
// (fakeRepo, no SQL at all), these wire the real PostgresRepository together
// with a real invoice.Service so the actual ON CONFLICT DO NOTHING guard in
// finance_recurring_runs, the real DATE round-trip and true connection
// concurrency are what get exercised — not an in-memory map.

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// newRealService wires the recurring service to real repositories: its own
// PostgresRepository plus a real invoice.Service backed by the real invoice
// repository, real number sequence and real pool — so tests can also call
// invSvc.Send to move a draft out of the way (finance_invoices has a single
// UNIQUE(tenant_id, invoice_number) index and every draft's number is the
// empty string, so a tenant can only ever hold ONE unsent draft at a time;
// see fix-invoice-number-unique-index-blocks-second-draft).
func newRealService(pool *pgxpool.Pool) (*Service, *PostgresRepository, *invoice.Service) {
	repo := NewPostgresRepository(pool)
	invRepo := invoice.NewPostgresRepository(pool)
	csRepo := quote.NewPostgresCompanySettingsRepo(pool)
	seqRepo := quote.NewPostgresNumberSequenceRepo(pool)
	invSvc := invoice.NewService(invRepo, seqRepo, csRepo, nil, pool)
	return NewService(repo, invSvc, invSvc), repo, invSvc
}

func billableLineItems(t *testing.T) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal([]models.LineItem{{
		Description: "CRM Lizenz Enterprise",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   decimal.RequireFromString("790.00"),
		TaxRate:     decimal.NewFromInt(19),
	}})
	require.NoError(t, err)
	return raw
}

func countInvoicesForSchedule(t *testing.T, pool *pgxpool.Pool, ctx context.Context, recurringID uuid.UUID) int {
	t.Helper()
	var count int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM finance_invoices WHERE recurring_id = $1", recurringID,
	).Scan(&count))
	return count
}

// A second scheduler pass or a retried request for the very same period must
// not bill the customer twice — the run ledger's UNIQUE constraint is the
// guard, not any in-memory state, so this must run against the real table.
func TestGenerate_RealSQL_ReplaySamePeriodReturnsExistingInvoice(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	svc, repo, _ := newRealService(pool)
	rec := newRecurringFixture(tenantID, userID)
	rec.LineItems = billableLineItems(t)
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	first, updated, err := svc.Generate(ctx, tenantID, rec.ID, userID)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", first.ID) })

	// Rewind the schedule to the same period, as a stale client or a replayed
	// request would — the run ledger, not the schedule state, must be the
	// guard that catches this.
	updated.NextRun = mustDateHelper("2026-03-15")
	require.NoError(t, repo.Update(ctx, updated))

	// Same call again, same day, same claimed period — must answer with the
	// invoice that already exists instead of creating a second one.
	second, _, err := svc.Generate(ctx, tenantID, rec.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "replay must return the already emitted invoice, not a new one")

	assert.Equal(t, 1, countInvoicesForSchedule(t, pool, ctx, rec.ID))
}

// Two gateway instances (or two racing scheduler ticks) calling Generate for
// the same schedule at the same moment must still emit exactly one invoice.
// A single warm pool connection cannot prove this — the two calls need to be
// in flight on the database at the same time, which requires two goroutines
// racing on the real pool so pgx hands out two separate connections.
func TestGenerate_RealSQL_ConcurrentRunsProduceExactlyOneInvoice(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	svc, repo, _ := newRealService(pool)
	rec := newRecurringFixture(tenantID, userID)
	rec.LineItems = billableLineItems(t)
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	const racers = 2
	var (
		wg     sync.WaitGroup
		start  = make(chan struct{})
		mu     sync.Mutex
		invIDs []uuid.UUID
		errs   []error
	)
	wg.Add(racers)
	for range racers {
		go func() {
			defer wg.Done()
			<-start
			inv, _, err := svc.Generate(ctx, tenantID, rec.ID, userID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			invIDs = append(invIDs, inv.ID)
		}()
	}
	close(start)
	wg.Wait()

	// The transient "being generated concurrently" error is an accepted
	// outcome for a racer that reads the claim before the winner attaches its
	// invoice — service.go:290 documents it. Any other error is a real bug.
	for _, err := range errs {
		assert.ErrorContains(t, err, "is being generated concurrently",
			"unexpected error from a concurrent Generate call: %v", err)
	}
	require.NotEmpty(t, invIDs, "at least one racer must succeed")
	for _, id := range invIDs {
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", id) })
		assert.Equal(t, invIDs[0], id, "every successful racer must report the same invoice id")
	}

	assert.Equal(t, 1, countInvoicesForSchedule(t, pool, ctx, rec.ID),
		"concurrent Generate calls for the same period must emit exactly one invoice")
}

// Ended and paused schedules must refuse generation even when the schedule
// was loaded fresh from Postgres, not from an in-memory fixture.
func TestGenerate_RealSQL_EndedAndPausedSchedulesRefuse(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	svc, repo, _ := newRealService(pool)

	cases := []struct {
		name   string
		status string
	}{
		{"paused", models.RecurringStatusPaused},
		{"ended", models.RecurringStatusEnded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := newRecurringFixture(tenantID, userID)
			rec.LineItems = billableLineItems(t)
			rec.Status = tc.status
			require.NoError(t, repo.Create(ctx, rec))
			t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

			_, _, err := svc.Generate(ctx, tenantID, rec.ID, userID)
			require.ErrorIs(t, err, ErrNotActive)
			assert.Equal(t, 0, countInvoicesForSchedule(t, pool, ctx, rec.ID))
		})
	}
}

// A monthly schedule anchored on the 31st must stay anchored on the 31st
// after a real Update round-trip through the DATE column — not clamp forever
// once it passes through February. Exercises addMonthsClamped's real value:
// re-deriving next_run from start_date on every generate, not by repeatedly
// advancing the stored value.
func TestGenerate_RealSQL_MonthEndScheduleStaysAnchored(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	svc, repo, invSvc := newRealService(pool)

	rec := newRecurringFixture(tenantID, userID)
	rec.LineItems = billableLineItems(t)
	rec.StartDate = mustDateHelper("2026-01-31")
	rec.NextRun = rec.StartDate
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	// January -> clamped to the 28th (2026 is not a leap year).
	inv1, updated, err := svc.Generate(ctx, tenantID, rec.ID, userID)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", inv1.ID) })
	assert.Equal(t, "2026-02-28", updated.NextRun.Format(dateLayout))

	// Re-load from Postgres — proves the DATE column round-trips the clamped
	// value correctly, not just the in-memory struct returned by Generate.
	reloaded, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, "2026-02-28", reloaded.NextRun.Format(dateLayout))

	// Send the January invoice before generating February's: finance_invoices
	// carries a single UNIQUE(tenant_id, invoice_number) index and every draft's
	// number is the empty string, so this tenant could not hold two drafts at
	// once (see fix-invoice-number-unique-index-blocks-second-draft) — sending
	// is also what a real schedule would do between two periods in practice.
	require.NoError(t, invSvc.Send(ctx, tenantID, inv1.ID, userID))

	// February -> back to the 31st, not stuck on the 28th forever.
	inv2, updated2, err := svc.Generate(ctx, tenantID, rec.ID, userID)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", inv2.ID) })
	assert.Equal(t, "2026-03-31", updated2.NextRun.Format(dateLayout))
	assert.NotEqual(t, inv1.ID, inv2.ID, "two distinct periods must emit two distinct invoices")

	assert.Equal(t, 2, countInvoicesForSchedule(t, pool, ctx, rec.ID))
}

// The frontend's end-date-deletion signal (RecurringInvoiceDialog.tsx:158,
// accepted backend-side since a5f5aa6f) must actually clear end_date to NULL
// in Postgres, not merely in the returned struct — repo.Update writes
// end_date = $15 unconditionally, so a nullable-column round-trip needs a
// real read-back to prove it.
func TestUpdate_RealSQL_ClearEndDatePersistsNull(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	svc, repo, _ := newRealService(pool)

	end := mustDateHelper("2026-06-01")
	rec := newRecurringFixture(tenantID, userID)
	rec.LineItems = billableLineItems(t)
	rec.EndDate = &end
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	loaded, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.EndDate, "fixture must actually carry an end date before clearing it")

	_, err = svc.Update(ctx, tenantID, rec.ID, UpdateInput{ClearEndDate: true})
	require.NoError(t, err)

	reloaded, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	assert.Nil(t, reloaded.EndDate, "end_date must be NULL in Postgres after ClearEndDate, not just nil in memory")
}
