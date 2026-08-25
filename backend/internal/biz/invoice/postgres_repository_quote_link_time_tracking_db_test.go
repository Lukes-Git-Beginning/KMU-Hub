package invoice

// postgres_repository_quote_link_time_tracking_db_test.go exercises GetByQuoteID
// (postgres_repository.go:446) and LinkTimeTracking (postgres_repository.go:472)
// against real Postgres.
//
// FINDING (resolved in fix-quote-to-invoice-duplicate-creation, hardened in
// harden-quote-conversion-unique-index): GetByQuoteID's signature returns exactly
// one *models.Invoice, and the schema still allows several invoices to share a
// source_quote_id -- after a storno it legitimately does, which is what
// TestPostgresRepository_GetByQuoteID_MultipleInvoices still proves. What
// changed: the query orders explicitly (live invoice before cancelled, newest
// first), Service.CreateFromQuote rejects a second conversion with
// ErrQuoteAlreadyConverted via a read-then-write check, and Migration 000324
// backs that check with a partial unique index on
// (tenant_id, source_quote_id) WHERE source_quote_id IS NOT NULL AND
// status <> 'cancelled' -- so two genuinely concurrent conversions no longer
// both succeed: the loser's Create hits the index and repo.Create maps it to
// ErrQuoteAlreadyConverted (isQuoteAlreadyConvertedConflict). The index is why
// this test can no longer create two LIVE invoices for the same quote directly
// through the repository -- it cancels the first before creating the second.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresRepository_GetByQuoteID_ReturnsInvoiceWithLineItems(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "GetByQuoteID Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	quoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"status": "accepted", "customer_name": "Quote Link Test GmbH",
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_quotes", quoteID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	inv.SourceQuoteID = &quoteID
	createListTestInvoice(t, repo, pool, ctx, inv)

	got, err := repo.GetByQuoteID(ctx, tenantID, quoteID)
	require.NoError(t, err)
	assert.Equal(t, inv.ID, got.ID)
	require.NotNil(t, got.SourceQuoteID)
	assert.Equal(t, quoteID, *got.SourceQuoteID)

	var lines []models.LineItem
	require.NoError(t, json.Unmarshal(got.LineItems, &lines))
	assert.Len(t, lines, 1, "GetByQuoteID must populate LineItems like the other read paths")
}

// TestPostgresRepository_GetByQuoteID_ForeignTenantReturnsNotFound proves the
// tenant scoping is on the invoice's own tenant_id, not derived from the quote:
// a matching source_quote_id under a foreign tenant argument finds nothing.
func TestPostgresRepository_GetByQuoteID_ForeignTenantReturnsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	ownerTenant := uuid.New()
	testutil.EnsureTenant(t, pool, ownerTenant, "GetByQuoteID Owner Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", ownerTenant)

	foreignTenant := uuid.New()
	testutil.EnsureTenant(t, pool, foreignTenant, "GetByQuoteID Foreign Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", foreignTenant)

	quoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"id": uuid.New(), "tenant_id": ownerTenant,
		"status": "accepted", "customer_name": "Quote Link Foreign Tenant Test GmbH",
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_quotes", quoteID) })

	repo := NewPostgresRepository(pool)
	ownerCtx := testutil.WithTenantCtx(context.Background(), ownerTenant)

	now := time.Now().UTC()
	inv := newListTestInvoice(ownerTenant, now, now.AddDate(0, 0, 14))
	inv.SourceQuoteID = &quoteID
	createListTestInvoice(t, repo, pool, ownerCtx, inv)

	foreignCtx := testutil.WithTenantCtx(context.Background(), foreignTenant)
	_, err := repo.GetByQuoteID(foreignCtx, foreignTenant, quoteID)
	assert.ErrorIs(t, err, ErrInvoiceNotFound, "a foreign tenant id must not see another tenant's invoice, even with the right quote id")
}

// TestPostgresRepository_GetByQuoteID_MultipleInvoices is the FINDING documented
// in the file header made concrete: the schema allows several invoices to share
// a source_quote_id (a storno legitimately produces this), and GetByQuoteID must
// resolve the ambiguity deterministically rather than returning an arbitrary row.
// The two rows can no longer BOTH be live (Migration 000324's partial unique
// index forbids it, see isQuoteAlreadyConvertedConflict) -- so the first is
// cancelled before the second is created, mirroring how the shape legitimately
// arises via Service.Cancel's storno path.
func TestPostgresRepository_GetByQuoteID_MultipleInvoices(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "GetByQuoteID Multiple Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	quoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"status": "accepted", "customer_name": "Quote Link Multiple Test GmbH",
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_quotes", quoteID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	first := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	first.SourceQuoteID = &quoteID
	createListTestInvoice(t, repo, pool, ctx, first)
	require.NoError(t, repo.UpdateStatus(ctx, tenantID, first.ID, models.InvoiceStatusCancelled),
		"the partial unique index only allows one LIVE invoice per quote -- the first must free its slot before the second is created")

	second := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	second.SourceQuoteID = &quoteID
	createListTestInvoice(t, repo, pool, ctx, second)

	got, err := repo.GetByQuoteID(ctx, tenantID, quoteID)
	require.NoError(t, err, "two invoices for the same quote (one cancelled) is a legitimate, schema-allowed shape")
	assert.Equal(t, second.ID, got.ID,
		"GetByQuoteID must deterministically prefer the live invoice over the cancelled one")
}

// TestPostgresRepository_Create_RejectsSecondLiveInvoiceForSameQuote pins
// Migration 000324 directly against the repository, independent of
// Service.CreateFromQuote's read-then-write check: two concurrent inserts for
// the same (tenant_id, source_quote_id) cannot both land as live invoices, and
// the loser gets ErrQuoteAlreadyConverted rather than a raw pg unique-violation
// (isQuoteAlreadyConvertedConflict in postgres_repository.go).
func TestPostgresRepository_Create_RejectsSecondLiveInvoiceForSameQuote(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Create Duplicate Quote Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	quoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"status": "accepted", "customer_name": "Create Duplicate Quote Test GmbH",
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_quotes", quoteID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	first := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	first.SourceQuoteID = &quoteID
	createListTestInvoice(t, repo, pool, ctx, first)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", first.ID) })

	second := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	second.SourceQuoteID = &quoteID
	err := repo.Create(ctx, second)
	require.ErrorIs(t, err, ErrQuoteAlreadyConverted,
		"the index must map to the same sentinel Service.CreateFromQuote's read-then-write check returns, not a raw pg error")
}

// TestPostgresRepository_GetByQuoteID_PrefersLiveInvoiceOverCancelled pins the
// ordering the duplicate-conversion guard in Service.CreateFromQuote depends on:
// after a storno the quote may be invoiced again, so both rows exist -- and the
// cancelled one is the NEWER of the two here, which is exactly the case an
// unordered query (or a plain created_at DESC) would get wrong. The "cancelled"
// row is inserted already cancelled (rather than created live and stornoed
// after) so the fixture itself never puts two live invoices under the same
// quote at once -- Migration 000324's partial unique index would reject that.
func TestPostgresRepository_GetByQuoteID_PrefersLiveInvoiceOverCancelled(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "GetByQuoteID Storno Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	quoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"status": "accepted", "customer_name": "Quote Link Storno Test GmbH",
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_quotes", quoteID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	live := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	live.SourceQuoteID = &quoteID
	createListTestInvoice(t, repo, pool, ctx, live)

	cancelled := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	cancelled.SourceQuoteID = &quoteID
	cancelled.Status = models.InvoiceStatusCancelled
	createListTestInvoice(t, repo, pool, ctx, cancelled)

	got, err := repo.GetByQuoteID(ctx, tenantID, quoteID)

	require.NoError(t, err)
	assert.Equal(t, live.ID, got.ID,
		"a cancelled invoice must never mask the live one -- CreateFromQuote would otherwise allow a duplicate conversion")
}

func TestPostgresRepository_LinkTimeTracking_PersistsAndSecondCallOverwrites(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "LinkTimeTracking Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	createListTestInvoice(t, repo, pool, ctx, inv)

	first := json.RawMessage(`{"employee_id":"emp-1","total_minutes":120}`)
	require.NoError(t, repo.LinkTimeTracking(ctx, tenantID, inv.ID, first))

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.JSONEq(t, string(first), string(got.TimeTrackingSource))

	// Second call on the same invoice overwrites rather than merging/appending --
	// a caller that expects an audit trail of multiple time-tracking batches
	// would lose the first one silently.
	second := json.RawMessage(`{"employee_id":"emp-2","total_minutes":45}`)
	require.NoError(t, repo.LinkTimeTracking(ctx, tenantID, inv.ID, second))

	got, err = repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.JSONEq(t, string(second), string(got.TimeTrackingSource), "LinkTimeTracking overwrites the column, it does not merge or append")
}

// TestPostgresRepository_LinkTimeTracking_RejectsInvalidJSON proves the jsonb
// column type is the only validation LinkTimeTracking gets -- Postgres rejects
// malformed JSON at the SQL layer, the repository has no client-side check.
func TestPostgresRepository_LinkTimeTracking_RejectsInvalidJSON(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "LinkTimeTracking Invalid JSON Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	createListTestInvoice(t, repo, pool, ctx, inv)

	err := repo.LinkTimeTracking(ctx, tenantID, inv.ID, json.RawMessage(`{not valid json`))
	require.Error(t, err, "Postgres must reject malformed JSON written to a jsonb column")

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.Nil(t, got.TimeTrackingSource, "a rejected write must leave the column untouched, not partially applied")
}

// TestPostgresRepository_LinkTimeTracking_CrossTenantIsNoop mirrors the same
// silent-noop shape SetLock/UpdateStatus have (postgres_repository_lock_db_test.go,
// postgres_repository_status_db_test.go): a foreign tenant id affects zero rows
// and returns no error.
func TestPostgresRepository_LinkTimeTracking_CrossTenantIsNoop(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	ownerTenant := uuid.New()
	testutil.EnsureTenant(t, pool, ownerTenant, "LinkTimeTracking Owner Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", ownerTenant)

	foreignTenant := uuid.New()
	testutil.EnsureTenant(t, pool, foreignTenant, "LinkTimeTracking Foreign Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", foreignTenant)

	repo := NewPostgresRepository(pool)
	ownerCtx := testutil.WithTenantCtx(context.Background(), ownerTenant)

	now := time.Now().UTC()
	inv := newListTestInvoice(ownerTenant, now, now.AddDate(0, 0, 14))
	createListTestInvoice(t, repo, pool, ownerCtx, inv)

	foreignCtx := testutil.WithTenantCtx(context.Background(), foreignTenant)
	require.NoError(t, repo.LinkTimeTracking(foreignCtx, foreignTenant, inv.ID, json.RawMessage(`{"employee_id":"emp-x"}`)))

	got, err := repo.GetByID(ownerCtx, ownerTenant, inv.ID)
	require.NoError(t, err)
	assert.Nil(t, got.TimeTrackingSource, "a cross-tenant LinkTimeTracking must affect zero rows")
}
