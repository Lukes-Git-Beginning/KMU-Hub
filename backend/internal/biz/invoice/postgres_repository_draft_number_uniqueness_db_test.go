package invoice

// postgres_repository_draft_number_uniqueness_db_test.go pins the shape of the
// unique index idx_finance_invoices_number after migration 000323 made it
// partial (WHERE invoice_number <> '').
//
// Two properties have to hold at once and pull in opposite directions:
// unnumbered drafts (invoice_number = '', assigned only by Service.Send) must
// be allowed any number of times per tenant, while assigned numbers must stay
// unique per tenant — GoBD forbids handing out the same invoice number twice.
// Before 000323 the index was not partial, so the empty strings competed with
// each other and a tenant could hold exactly one unsent draft; the second
// Create failed with a raw 23505 that surfaced as a 500.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestPostgresRepository_Create_MultipleUnnumberedDraftsPerTenant proves a
// tenant can hold more than one draft without an invoice number. Every draft
// carries '' (the column is NOT NULL DEFAULT '', so there is no NULL to fall
// back on) — with a non-partial unique index the second Create here fails.
func TestPostgresRepository_Create_MultipleUnnumberedDraftsPerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice Draft Number Uniqueness Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	when := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	// Three, not two: a second draft could still pass a "one duplicate is
	// tolerated" index by accident, three cannot.
	for i := 0; i < 3; i++ {
		inv := newListTestInvoice(tenantID, when, when.AddDate(0, 0, 30))
		inv.InvoiceNumber = "" // what Service.Create writes: "Assigned when sent"
		require.NoErrorf(t, repo.Create(ctx, inv),
			"draft %d without an invoice number must be creatable alongside the earlier ones", i+1)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", inv.ID) })
	}

	var drafts int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM finance_invoices WHERE tenant_id = $1 AND invoice_number = ''`,
		tenantID).Scan(&drafts))
	assert.Equal(t, 3, drafts, "all three unnumbered drafts must be stored")
}

// TestPostgresRepository_Create_AssignedNumberStaysUniquePerTenant proves the
// partial index did not give up what it was there for: the same assigned
// number twice within a tenant is still rejected by the database, and it is
// still THIS index that rejects it.
func TestPostgresRepository_Create_AssignedNumberStaysUniquePerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice Assigned Number Uniqueness Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	when := time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC)
	number := "RE-2026-PARTIALIDX-0001"

	first := newListTestInvoice(tenantID, when, when.AddDate(0, 0, 30))
	first.InvoiceNumber = number
	first.Status = models.InvoiceStatusSent
	require.NoError(t, repo.Create(ctx, first))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", first.ID) })

	second := newListTestInvoice(tenantID, when, when.AddDate(0, 0, 30))
	second.InvoiceNumber = number
	second.Status = models.InvoiceStatusSent
	err := repo.Create(ctx, second)
	require.Error(t, err, "the same assigned invoice number must not be storable twice for one tenant")

	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr), "expected a Postgres error, got %v", err)
	assert.Equal(t, "23505", pgErr.Code)
	assert.Equal(t, "idx_finance_invoices_number", pgErr.ConstraintName,
		"the assigned-number guarantee must still come from the partial index, not from some other constraint")
}

// TestPostgresRepository_Create_SameAssignedNumberInTwoTenants proves the
// uniqueness stays tenant-scoped after the index change — two tenants running
// their own number ranges must not collide.
func TestPostgresRepository_Create_SameAssignedNumberInTwoTenants(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Invoice Number Cross Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Invoice Number Cross Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresRepository(pool)
	when := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	number := "RE-2026-CROSSTENANT-0001"

	for _, tenantID := range []uuid.UUID{tenantA, tenantB} {
		ctx := testutil.WithTenantCtx(context.Background(), tenantID)
		inv := newListTestInvoice(tenantID, when, when.AddDate(0, 0, 30))
		inv.InvoiceNumber = number
		inv.Status = models.InvoiceStatusSent
		require.NoError(t, repo.Create(ctx, inv))
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", inv.ID) })
	}
}
