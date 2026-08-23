package invoice

// postgres_repository_quote_link_time_tracking_db_test.go exercises GetByQuoteID
// (postgres_repository.go:446) and LinkTimeTracking (postgres_repository.go:472)
// against real Postgres.
//
// FINDING (documented, not fixed here — a business-logic decision, not a
// repository bug): GetByQuoteID's signature returns exactly one *models.Invoice,
// but neither the schema (no UNIQUE constraint on finance_invoices.source_quote_id)
// nor Service.CreateFromQuote (internal/biz/invoice/service.go:779, called from
// the ConvertQuoteToInvoice RPC with no prior existence check) guarantees a quote
// has at most one invoice. TestPostgresRepository_GetByQuoteID_MultipleInvoices
// proves the repository silently returns an arbitrary one of several matching rows
// (no ORDER BY) instead of erroring — the same shape of gap that caused the time-entry
// double-billing bug fixed in Lauf 10 (fix-biz-time-entry-invoice-double-billing).
// A retried or double-clicked "convert quote to invoice" call today creates a second,
// full-amount invoice for the same accepted quote. Filed as
// fix-quote-to-invoice-duplicate-creation for a future iteration: this repository's
// scope is proving the read behavior, not changing CreateFromQuote's validation.
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
// in the file header made concrete: nothing in the schema or this query prevents
// two invoices from sharing a source_quote_id, and GetByQuoteID's single-result
// signature silently picks one (no ORDER BY -- undefined which) instead of
// signaling the ambiguity to the caller.
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

	second := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	second.SourceQuoteID = &quoteID
	createListTestInvoice(t, repo, pool, ctx, second)

	got, err := repo.GetByQuoteID(ctx, tenantID, quoteID)
	require.NoError(t, err, "no schema constraint rejects a second invoice for the same quote -- this is the gap")
	assert.Contains(t, []uuid.UUID{first.ID, second.ID}, got.ID,
		"GetByQuoteID returns one of the two matching invoices with no defined ordering -- neither is wrong, but a caller expecting exactly-one would be")
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
