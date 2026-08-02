package einvoice

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestTenantIsolation_IncomingInvoices verifies that RLS prevents one tenant
// from reading incoming invoices belonging to another tenant.
func TestTenantIsolation_IncomingInvoices(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	// Freshly minted tenants, not the shared testutil.TenantA/TenantB: the fixture
	// XML carries a fixed supplier and invoice number, so importing it into a tenant
	// that a previous run already used trips the duplicate check instead of testing
	// isolation.
	tenantA := uuid.New()
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Tenant A EInvoice")
	testutil.EnsureTenant(t, pool, tenantB, "Tenant B EInvoice")

	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	xmlData, err := os.ReadFile("testdata/cii_minimal.xml")
	require.NoError(t, err)

	repoA := NewPostgresRepository(pool)
	repoB := NewPostgresRepository(pool)

	svcA := NewService(repoA)
	svcB := NewService(repoB)

	// Import invoice for tenant A
	invA, err := svcA.Import(ctxA, ImportInput{
		TenantID:  tenantA,
		CreatedBy: uuid.New(),
		Filename:  "isolation-test-a.xml",
		Content:   xmlData,
		MimeType:  "application/xml",
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_incoming_invoices", invA.ID) })

	// Tenant B must NOT see tenant A's invoice
	_, err = svcB.Get(ctxB, tenantB, invA.ID)
	assert.ErrorIs(t, err, ErrNotFound, "tenant B must not access tenant A's invoice")

	// Tenant B list must NOT contain tenant A's invoice
	resultB, _, err := svcB.List(ctxB, tenantB, ListInput{})
	require.NoError(t, err)
	for _, inv := range resultB {
		assert.NotEqual(t, invA.ID, inv.ID, "tenant B list must not contain tenant A's invoice")
	}

	// Tenant A CAN read its own invoice
	invARead, err := svcA.Get(ctxA, tenantA, invA.ID)
	require.NoError(t, err)
	assert.Equal(t, invA.ID, invARead.ID)
}
