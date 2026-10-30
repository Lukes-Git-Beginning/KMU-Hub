package einkauf

// The lines of a closed or cancelled PO are frozen like its header: add,
// update and delete answer ErrPONotDraft and leave the rows and the PO total
// untouched. Under draft the same calls keep working.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPOLines_FrozenForClosedAndCancelledPO(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close) // registered first, so it runs after the row cleanups

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Einkauf PO Line Status Tenant")
	repo := NewPostgresRepository(pool)
	svc := NewServiceExtended(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctx, tenantID, "PO Line Status Supplier")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "suppliers", supplier.ID) })

	for _, frozen := range []POStatus{POStatusClosed, POStatusCancelled} {
		t.Run(string(frozen), func(t *testing.T) {
			po, err := svc.CreatePO(ctx, CreatePOInput{TenantID: tenantID, SupplierID: supplier.ID, PONumber: "PO-LS-" + uuid.New().String()[:8], Currency: "EUR"})
			require.NoError(t, err)
			t.Cleanup(func() {
				_, _ = pool.Exec(ctx, `DELETE FROM po_lines WHERE po_id = $1`, po.ID)
				testutil.CleanupRow(t, pool, "purchase_orders", po.ID)
			})

			// Draft: the three calls work.
			line, err := svc.AddPOLine(ctx, AddPOLineInput{TenantID: tenantID, POID: po.ID, ProductName: "Teil", Quantity: "2", UnitPrice: "10.00", TaxRate: "0.19", LinePosition: 1})
			require.NoError(t, err)
			qty := "3"
			_, err = svc.UpdatePOLine(ctx, UpdatePOLineInput{TenantID: tenantID, POID: po.ID, LineID: line.ID, Quantity: &qty})
			require.NoError(t, err)
			totalBefore := poTotal(t, svc, ctx, tenantID, po.ID)

			require.NoError(t, repo.UpdatePOStatus(ctx, tenantID, po.ID, frozen))

			_, err = svc.AddPOLine(ctx, AddPOLineInput{TenantID: tenantID, POID: po.ID, ProductName: "Neu", Quantity: "1", UnitPrice: "5.00", TaxRate: "0.19", LinePosition: 2})
			require.ErrorIs(t, err, ErrPONotDraft)
			qty = "9"
			_, err = svc.UpdatePOLine(ctx, UpdatePOLineInput{TenantID: tenantID, POID: po.ID, LineID: line.ID, Quantity: &qty})
			require.ErrorIs(t, err, ErrPONotDraft)
			require.ErrorIs(t, svc.DeletePOLine(ctx, tenantID, po.ID, line.ID), ErrPONotDraft)

			lines, err := svc.ListPOLines(ctx, tenantID, po.ID)
			require.NoError(t, err)
			require.Len(t, lines, 1, "no line added, none deleted")
			assert.Equal(t, "3.0000", lines[0].Quantity, "frozen line must not be written")
			assert.Equal(t, totalBefore, poTotal(t, svc, ctx, tenantID, po.ID), "total of a frozen PO must not move")
		})
	}
}

func poTotal(t *testing.T, svc *Service, ctx context.Context, tenantID, poID uuid.UUID) string {
	t.Helper()
	po, err := svc.GetPO(ctx, tenantID, poID)
	require.NoError(t, err)
	return po.TotalAmount
}
