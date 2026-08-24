package invoice

// postgres_repository_list_db_test.go exercises List (postgres_repository.go)
// against real Postgres. List is the largest single SQL method in this
// package (filter/sort/paginate/count) and, per the Block-B audit, had no
// test — neither the tagged `integration` suite nor an untagged one. These
// tests are untagged (testutil.SkipIfNoDB) so they run in the local gate and
// count toward this package's coverage number, unlike integration_test.go.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// listTestLine returns a single minimal, constraint-satisfying line item.
func listTestLine() []models.LineItem {
	return []models.LineItem{{
		ID:          "line-1",
		Position:    1,
		Description: "Testposition",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   decimal.NewFromInt(100),
		TaxRate:     decimal.NewFromInt(19),
		LineTotal:   decimal.NewFromInt(100),
	}}
}

// newListTestInvoice builds a minimal valid, unsaved Invoice for List filter
// tests. Callers set Status/InvoiceDate/DueDate/ContactID/RecurringID as
// needed before calling repo.Create.
func newListTestInvoice(tenantID uuid.UUID, invoiceDate, dueDate time.Time) *models.Invoice {
	raw, _ := json.Marshal(listTestLine())
	now := time.Now().UTC().Truncate(time.Second)
	return &models.Invoice{
		ID:            uuid.New(),
		TenantID:      tenantID,
		InvoiceNumber: "LIST-" + uuid.NewString()[:8],
		Status:        models.InvoiceStatusDraft,
		CustomerName:  "List Filter Test GmbH",
		TaxMode:       models.TaxModeStandard,
		LineItems:     raw,
		Subtotal:      decimal.NewFromInt(100),
		TotalTax:      decimal.NewFromInt(19),
		GrossTotal:    decimal.NewFromInt(119),
		Currency:      "EUR",
		InvoiceDate:   invoiceDate,
		DueDate:       dueDate,
		CreatedBy:     uuid.New(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// createListTestInvoice creates the invoice via the real repository (the
// actual write path, not a raw SeedRow) and registers cleanup.
func createListTestInvoice(t *testing.T, repo *PostgresRepository, pool *pgxpool.Pool, ctx context.Context, inv *models.Invoice) {
	t.Helper()
	require.NoError(t, repo.Create(ctx, inv))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", inv.ID) })
}

// seedListTestContact creates a user (contacts.created_by FK target) and a
// contact for it, returning the contact id.
func seedListTestContact(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "list-contact-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "List", "last_name": "Owner",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"first_name": "Kontakt", "last_name": "Test-" + uuid.NewString()[:8],
		"created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })
	return contactID
}

// seedListTestRecurring creates a minimal finance_recurring_invoices row,
// returning its id.
func seedListTestRecurring(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "finance_recurring_invoices", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"title":               "List Filter Schedule",
		"recurrence_interval": "monthly",
		"start_date":          time.Now().UTC(),
		"next_run":            time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_recurring_invoices", id) })
	return id
}

// TestPostgresRepository_List_StatusFilter proves Status filters both the
// returned page and the total counter, and that unmatched rows never appear.
func TestPostgresRepository_List_StatusFilter(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice List Status Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	draft := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 30))
	draft.Status = models.InvoiceStatusDraft
	createListTestInvoice(t, repo, pool, ctx, draft)

	sent := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 30))
	sent.Status = models.InvoiceStatusSent
	createListTestInvoice(t, repo, pool, ctx, sent)

	got, total, err := repo.List(ctx, tenantID, ListFilter{Status: models.InvoiceStatusDraft, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "status filter must scope the total counter too")
	require.Len(t, got, 1)
	assert.Equal(t, draft.ID, got[0].ID)
}

// TestPostgresRepository_List_DateRangeInclusiveBoundaries proves DateFrom and
// DateTo are both inclusive: an invoice dated exactly on either boundary must
// be returned, one day outside either boundary must not.
func TestPostgresRepository_List_DateRangeInclusiveBoundaries(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice List Date Range Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	due := from.AddDate(0, 0, 30)

	onFrom := newListTestInvoice(tenantID, from, due)
	createListTestInvoice(t, repo, pool, ctx, onFrom)

	onTo := newListTestInvoice(tenantID, to, due)
	createListTestInvoice(t, repo, pool, ctx, onTo)

	beforeFrom := newListTestInvoice(tenantID, from.AddDate(0, 0, -1), due)
	createListTestInvoice(t, repo, pool, ctx, beforeFrom)

	afterTo := newListTestInvoice(tenantID, to.AddDate(0, 0, 1), due)
	createListTestInvoice(t, repo, pool, ctx, afterTo)

	got, total, err := repo.List(ctx, tenantID, ListFilter{DateFrom: &from, DateTo: &to, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	ids := map[uuid.UUID]bool{}
	for _, inv := range got {
		ids[inv.ID] = true
	}
	assert.True(t, ids[onFrom.ID], "invoice dated exactly on DateFrom must be included (inclusive lower bound)")
	assert.True(t, ids[onTo.ID], "invoice dated exactly on DateTo must be included (inclusive upper bound)")
	assert.False(t, ids[beforeFrom.ID], "invoice one day before DateFrom must be excluded")
	assert.False(t, ids[afterTo.ID], "invoice one day after DateTo must be excluded")
}

// TestPostgresRepository_List_OverdueFilter proves Overdue returns only sent
// invoices whose due_date is in the past — not draft invoices past due, and
// not sent invoices not yet due.
func TestPostgresRepository_List_OverdueFilter(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice List Overdue Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	overdue := newListTestInvoice(tenantID, now.AddDate(0, 0, -40), now.AddDate(0, 0, -5))
	overdue.Status = models.InvoiceStatusSent
	createListTestInvoice(t, repo, pool, ctx, overdue)

	notYetDue := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 5))
	notYetDue.Status = models.InvoiceStatusSent
	createListTestInvoice(t, repo, pool, ctx, notYetDue)

	draftPastDue := newListTestInvoice(tenantID, now.AddDate(0, 0, -40), now.AddDate(0, 0, -5))
	draftPastDue.Status = models.InvoiceStatusDraft
	createListTestInvoice(t, repo, pool, ctx, draftPastDue)

	got, total, err := repo.List(ctx, tenantID, ListFilter{Overdue: true, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, got, 1)
	assert.Equal(t, overdue.ID, got[0].ID)
}

// TestPostgresRepository_List_ContactIDFilter proves ContactID scopes the
// list to invoices linked to that contact, excluding both other contacts'
// invoices and invoices with no contact at all.
func TestPostgresRepository_List_ContactIDFilter(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice List Contact Tenant")
	// t.Cleanup (not defer): must run after seedListTestContact's own
	// t.Cleanup-registered user/contact deletes (LIFO order), or the tenant
	// row is gone first and the users/contacts FK-to-tenants delete fails.
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	contactA := seedListTestContact(t, pool, tenantID)
	contactB := seedListTestContact(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()
	due := now.AddDate(0, 0, 30)

	forA := newListTestInvoice(tenantID, now, due)
	forA.ContactID = &contactA
	createListTestInvoice(t, repo, pool, ctx, forA)

	forB := newListTestInvoice(tenantID, now, due)
	forB.ContactID = &contactB
	createListTestInvoice(t, repo, pool, ctx, forB)

	noContact := newListTestInvoice(tenantID, now, due)
	createListTestInvoice(t, repo, pool, ctx, noContact)

	got, total, err := repo.List(ctx, tenantID, ListFilter{ContactID: &contactA, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, got, 1)
	assert.Equal(t, forA.ID, got[0].ID)
}

// TestPostgresRepository_List_RecurringIDFilter proves RecurringID scopes the
// list to invoices emitted by that schedule, excluding other schedules and
// manually created invoices (recurring_id NULL).
func TestPostgresRepository_List_RecurringIDFilter(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice List Recurring Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	scheduleA := seedListTestRecurring(t, pool, tenantID)
	scheduleB := seedListTestRecurring(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()
	due := now.AddDate(0, 0, 30)

	fromA := newListTestInvoice(tenantID, now, due)
	fromA.RecurringID = &scheduleA
	createListTestInvoice(t, repo, pool, ctx, fromA)

	fromB := newListTestInvoice(tenantID, now, due)
	fromB.RecurringID = &scheduleB
	createListTestInvoice(t, repo, pool, ctx, fromB)

	manual := newListTestInvoice(tenantID, now, due)
	createListTestInvoice(t, repo, pool, ctx, manual)

	got, total, err := repo.List(ctx, tenantID, ListFilter{RecurringID: &scheduleA, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, got, 1)
	assert.Equal(t, fromA.ID, got[0].ID)
}

// TestPostgresRepository_List_TotalCountIndependentOfLimit proves the total
// counter reflects every filter-matching row, not just the page returned —
// more matching rows exist than the limit, and non-matching rows exist too.
func TestPostgresRepository_List_TotalCountIndependentOfLimit(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice List Total Count Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()
	due := now.AddDate(0, 0, 30)

	const matching = 5
	for range matching {
		inv := newListTestInvoice(tenantID, now, due)
		inv.Status = models.InvoiceStatusDraft
		createListTestInvoice(t, repo, pool, ctx, inv)
	}
	for range 2 {
		inv := newListTestInvoice(tenantID, now, due)
		inv.Status = models.InvoiceStatusSent
		createListTestInvoice(t, repo, pool, ctx, inv)
	}

	got, total, err := repo.List(ctx, tenantID, ListFilter{Status: models.InvoiceStatusDraft, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, matching, total, "total must count all matching rows, not just the returned page")
	assert.Len(t, got, 2, "the returned page must still respect Limit")
}

// TestPostgresRepository_List_CrossTenantIsolation proves a tenant's List
// call never returns another tenant's invoices, in either the page or the
// total counter.
func TestPostgresRepository_List_CrossTenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Invoice List CrossTenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Invoice List CrossTenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()
	due := now.AddDate(0, 0, 30)

	invA := newListTestInvoice(tenantA, now, due)
	createListTestInvoice(t, repo, pool, ctxA, invA)

	invB := newListTestInvoice(tenantB, now, due)
	createListTestInvoice(t, repo, pool, ctxB, invB)

	got, total, err := repo.List(ctxA, tenantA, ListFilter{Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "total must not include the other tenant's invoice")
	for _, inv := range got {
		assert.Equal(t, tenantA, inv.TenantID, "List leaked a cross-tenant invoice")
	}
}
