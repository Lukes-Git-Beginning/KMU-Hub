package recurring

// Real-schema DB tests for PostgresRepository against finance_recurring_invoices
// and finance_recurring_runs. Unlike the fakeRepo-backed tests in service_test.go,
// these exercise the actual SQL: column lists staying in sync with recurringColumns,
// the ON CONFLICT DO NOTHING claim race guard, and the invoice_id IS NULL condition
// that stops ReleasePeriod from deleting an already-attached run. Every write and
// every legitimate read runs under testutil.WithTenantCtx for its owning tenant —
// both tables have FORCE ROW LEVEL SECURITY, so an unscoped context is rejected
// outright rather than just filtered.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedRecurringFixture seeds a tenant and a user, the two parents every
// finance_recurring_invoices row needs (created_by is nullable but exercised
// here for realism).
func seedRecurringFixture(t *testing.T, pool *pgxpool.Pool) (tenantID, userID uuid.UUID) {
	t.Helper()
	tenantID = uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Recurring Repo DB Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	userID = testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         "recurring-repo-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	return tenantID, userID
}

func newRecurringFixture(tenantID, userID uuid.UUID) *models.RecurringInvoice {
	now := time.Now().UTC().Truncate(time.Second)
	start := mustDateHelper("2026-03-15")
	return &models.RecurringInvoice{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Title:            "CRM-Lizenz Enterprise",
		CustomerName:     "Musterfirma GmbH",
		TaxMode:          models.TaxModeStandard,
		LineItems:        []byte(`[]`),
		TaxBreakdownRaw:  []byte(`{}`),
		Currency:         models.DefaultCurrency,
		Interval:         models.RecurringIntervalMonthly,
		Status:           models.RecurringStatusActive,
		StartDate:        start,
		NextRun:          start,
		PaymentTermsDays: 14,
		CreatedBy:        &userID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func mustDateHelper(value string) time.Time {
	t, err := time.Parse(dateLayout, value)
	if err != nil {
		panic(err)
	}
	return t
}

func TestPostgresRepository_CreateAndGetByID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	rec := newRecurringFixture(tenantID, userID)

	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	got, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, rec.Title, got.Title)
	assert.Equal(t, rec.Interval, got.Interval)
	assert.Equal(t, models.RecurringStatusActive, got.Status)
	require.NotNil(t, got.CreatedBy)
	assert.Equal(t, userID, *got.CreatedBy)
	assert.Nil(t, got.EndDate)
	assert.Nil(t, got.LastGeneratedAt)
}

func TestPostgresRepository_GetByID_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	rec := newRecurringFixture(tenantID, userID)
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	// Unknown id.
	_, err := repo.GetByID(ctx, tenantID, uuid.New())
	require.ErrorIs(t, err, ErrNotFound)

	// Known id, wrong tenant filter — the tenant_id = $1 predicate must not
	// leak the row across tenants even under an authorized session.
	_, err = repo.GetByID(ctx, uuid.New(), rec.ID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPostgresRepository_List(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)

	active := newRecurringFixture(tenantID, userID)
	require.NoError(t, repo.Create(ctx, active))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", active.ID) })

	paused := newRecurringFixture(tenantID, userID)
	paused.Status = models.RecurringStatusPaused
	require.NoError(t, repo.Create(ctx, paused))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", paused.ID) })

	// A different tenant's schedule must never surface in this tenant's list.
	otherTenantID, otherUserID := seedRecurringFixture(t, pool)
	otherCtx := testutil.WithTenantCtx(context.Background(), otherTenantID)
	foreign := newRecurringFixture(otherTenantID, otherUserID)
	require.NoError(t, repo.Create(otherCtx, foreign))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", foreign.ID) })

	items, total, err := repo.List(ctx, tenantID, ListFilter{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, items, 2)

	filtered, filteredTotal, err := repo.List(ctx, tenantID, ListFilter{Status: models.RecurringStatusPaused})
	require.NoError(t, err)
	assert.Equal(t, 1, filteredTotal)
	require.Len(t, filtered, 1)
	assert.Equal(t, paused.ID, filtered[0].ID)

	// Limit above 200 and a negative offset must clamp, not error.
	clamped, clampedTotal, err := repo.List(ctx, tenantID, ListFilter{Limit: 5000, Offset: -10})
	require.NoError(t, err)
	assert.Equal(t, 2, clampedTotal)
	assert.Len(t, clamped, 2)

	empty, emptyTotal, err := repo.List(ctx, uuid.New(), ListFilter{})
	require.NoError(t, err)
	assert.Equal(t, 0, emptyTotal)
	assert.NotNil(t, empty, "empty result must be [] not null on the wire")
	assert.Empty(t, empty)
}

func TestPostgresRepository_Update(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	rec := newRecurringFixture(tenantID, userID)
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	rec.Title = "Aktualisierter Titel"
	rec.Status = models.RecurringStatusEnded
	rec.GeneratedCount = 3
	lastGenerated := mustDateHelper("2026-06-15")
	rec.LastGeneratedAt = &lastGenerated
	rec.NextRun = mustDateHelper("2026-07-15")
	require.NoError(t, repo.Update(ctx, rec))

	got, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, "Aktualisierter Titel", got.Title)
	assert.Equal(t, models.RecurringStatusEnded, got.Status)
	assert.Equal(t, 3, got.GeneratedCount)
	require.NotNil(t, got.LastGeneratedAt)
	assert.True(t, got.LastGeneratedAt.Equal(lastGenerated))
}

func TestPostgresRepository_Update_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	rec := newRecurringFixture(tenantID, userID)
	// Never created — the UPDATE affects zero rows.
	err := repo.Update(ctx, rec)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPostgresRepository_Delete(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	rec := newRecurringFixture(tenantID, userID)
	require.NoError(t, repo.Create(ctx, rec))

	require.NoError(t, repo.Delete(ctx, tenantID, rec.ID))
	_, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.ErrorIs(t, err, ErrNotFound)

	err = repo.Delete(ctx, tenantID, uuid.New())
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPostgresRepository_ClaimPeriod_ReplayReturnsTheSameRun(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	rec := newRecurringFixture(tenantID, userID)
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	period := mustDateHelper("2026-03-15")
	first, claimed, err := repo.ClaimPeriod(ctx, tenantID, rec.ID, period)
	require.NoError(t, err)
	assert.True(t, claimed)
	require.NotEqual(t, uuid.Nil, first.ID)
	assert.Nil(t, first.InvoiceID)

	// The UNIQUE (tenant_id, recurring_id, period_date) constraint is the real
	// guard here, not an application-side check — a second claim for the exact
	// same period must find the existing row via ON CONFLICT DO NOTHING and
	// answer claimed=false with the SAME run id, never a second row.
	second, claimedAgain, err := repo.ClaimPeriod(ctx, tenantID, rec.ID, period)
	require.NoError(t, err)
	assert.False(t, claimedAgain)
	assert.Equal(t, first.ID, second.ID, "a replayed claim must return the already-existing run, not a new one")

	// A different period for the same schedule claims cleanly.
	otherPeriod := mustDateHelper("2026-04-15")
	third, claimedThird, err := repo.ClaimPeriod(ctx, tenantID, rec.ID, otherPeriod)
	require.NoError(t, err)
	assert.True(t, claimedThird)
	assert.NotEqual(t, first.ID, third.ID)
}

// seedFinanceInvoice seeds the minimal finance_invoices row AttachInvoice's FK
// requires — tenant_id, invoice_date, due_date and created_by have no default.
func seedFinanceInvoice(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":    tenantID,
		"invoice_date": time.Now().UTC(),
		"due_date":     time.Now().UTC().AddDate(0, 0, 14),
		"created_by":   userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", id) })
	return id
}

func TestPostgresRepository_AttachInvoice(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	rec := newRecurringFixture(tenantID, userID)
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	period := mustDateHelper("2026-03-15")
	run, claimed, err := repo.ClaimPeriod(ctx, tenantID, rec.ID, period)
	require.NoError(t, err)
	require.True(t, claimed)

	invoiceID := seedFinanceInvoice(t, pool, tenantID, userID)
	require.NoError(t, repo.AttachInvoice(ctx, tenantID, run.ID, invoiceID))

	// A replayed claim must now carry the attached invoice id.
	replayed, claimedAgain, err := repo.ClaimPeriod(ctx, tenantID, rec.ID, period)
	require.NoError(t, err)
	assert.False(t, claimedAgain)
	require.NotNil(t, replayed.InvoiceID)
	assert.Equal(t, invoiceID, *replayed.InvoiceID)
}

func TestPostgresRepository_ReleasePeriod(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, userID := seedRecurringFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	rec := newRecurringFixture(tenantID, userID)
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", rec.ID) })

	// Unattached claim: release must remove it, freeing the period for a retry.
	unattachedPeriod := mustDateHelper("2026-03-15")
	run, claimed, err := repo.ClaimPeriod(ctx, tenantID, rec.ID, unattachedPeriod)
	require.NoError(t, err)
	require.True(t, claimed)

	require.NoError(t, repo.ReleasePeriod(ctx, tenantID, run.ID))

	retried, claimedAgain, err := repo.ClaimPeriod(ctx, tenantID, rec.ID, unattachedPeriod)
	require.NoError(t, err)
	assert.True(t, claimedAgain, "the period must be claimable again after release")
	assert.NotEqual(t, run.ID, retried.ID)

	// Attached claim: release must be a no-op — deleting an already-invoiced
	// run would let a retry re-emit a second invoice for the same period.
	attachedPeriod := mustDateHelper("2026-04-15")
	attachedRun, claimed, err := repo.ClaimPeriod(ctx, tenantID, rec.ID, attachedPeriod)
	require.NoError(t, err)
	require.True(t, claimed)
	invoiceID := seedFinanceInvoice(t, pool, tenantID, userID)
	require.NoError(t, repo.AttachInvoice(ctx, tenantID, attachedRun.ID, invoiceID))

	require.NoError(t, repo.ReleasePeriod(ctx, tenantID, attachedRun.ID))

	stillThere, claimedYetAgain, err := repo.ClaimPeriod(ctx, tenantID, rec.ID, attachedPeriod)
	require.NoError(t, err)
	assert.False(t, claimedYetAgain, "releasing an attached run must not delete it")
	assert.Equal(t, attachedRun.ID, stillThere.ID)
	require.NotNil(t, stillThere.InvoiceID)
	assert.Equal(t, invoiceID, *stillThere.InvoiceID)
}
