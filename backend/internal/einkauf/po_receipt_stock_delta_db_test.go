package einkauf

// The stock a goods receipt books is the quantity the write newly received,
// read under the PO lock, against the real schema as kmuhub_app. A service that
// read the lines before a concurrent receipt committed must not book that
// receipt's quantity a second time.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// staleLinesRepo serves ListPOLines from a snapshot taken earlier, as a service
// does whose read happened before another receipt committed.
type staleLinesRepo struct {
	RepositoryExtended
	snapshot []*POLine
}

func (r *staleLinesRepo) ListPOLines(context.Context, uuid.UUID, uuid.UUID) ([]*POLine, error) {
	return r.snapshot, nil
}

func TestGoodsReceipt_StockDeltaComesFromLockedQuantities(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Einkauf Receipt Stock Delta Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctx, tenantID, "Receipt Stock Delta Supplier")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "suppliers", supplier.ID) })

	adj := &recordingInventarAdjuster{}
	svc := NewServiceExtended(repo).WithInventarAdjuster(adj)

	// newPO creates a submitted PO with one line of 10 under SKU.
	newPO := func(t *testing.T, sku string) (*PurchaseOrder, uuid.UUID) {
		t.Helper()
		po, err := svc.CreatePO(ctx, CreatePOInput{TenantID: tenantID, SupplierID: supplier.ID, PONumber: "PO-SD-" + uuid.New().String()[:8], Currency: "EUR"})
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "purchase_orders", po.ID) })
		line, err := svc.AddPOLine(ctx, AddPOLineInput{TenantID: tenantID, POID: po.ID, ProductName: "Teil", SKU: sku, Quantity: "10", UnitPrice: "2", TaxRate: "0.19", LinePosition: 1})
		require.NoError(t, err)
		_, err = svc.SubmitPO(ctx, tenantID, po.ID)
		require.NoError(t, err)
		return po, line.ID
	}

	t.Run("a repeated line books the whole delta", func(t *testing.T) {
		po, lineID := newPO(t, "SKU-REPEAT")
		got, err := svc.PartialReceive(ctx, tenantID, po.ID, []PartialReceiveItem{
			{LineID: lineID, ReceivedQuantity: "4"},
			{LineID: lineID, ReceivedQuantity: "10"},
		})
		require.NoError(t, err)
		assert.Equal(t, POStatusReceived, got.Status)
		assert.Equal(t, int64(10), adj.booked["SKU-REPEAT"])
	})

	t.Run("a receipt that read stale quantities does not book them twice", func(t *testing.T) {
		po, lineID := newPO(t, "SKU-STALE")
		snapshot, err := repo.ListPOLines(ctx, tenantID, po.ID)
		require.NoError(t, err)
		require.Len(t, snapshot, 1)

		// The concurrent receipt commits 4 after the snapshot was taken.
		_, err = svc.PartialReceive(ctx, tenantID, po.ID, []PartialReceiveItem{{LineID: lineID, ReceivedQuantity: "4"}})
		require.NoError(t, err)
		require.Equal(t, int64(4), adj.booked["SKU-STALE"])

		stale := NewServiceExtended(&staleLinesRepo{RepositoryExtended: repo, snapshot: snapshot}).WithInventarAdjuster(adj)
		_, err = stale.ReceiveGoods(ctx, tenantID, po.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(10), adj.booked["SKU-STALE"], "4 from the first receipt + the 6 still open, not 4 + 10")

		stored, err := repo.GetPOWithLines(ctx, tenantID, po.ID)
		require.NoError(t, err)
		require.Len(t, stored.Lines, 1)
		assert.True(t, numericEqual("10", stored.Lines[0].ReceivedQuantity))
	})
}
