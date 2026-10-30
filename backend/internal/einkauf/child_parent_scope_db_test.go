package einkauf

// A PO line, a framework contract item and a supplier rating are reachable
// only through their own parent: addressed via another parent of the same
// tenant, update and delete answer NotFound and leave the row untouched.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestChildRoutes_RequireTheirOwnParent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close) // registered first, so it runs after the row cleanups

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Einkauf Child Parent Tenant")
	repo := NewPostgresRepository(pool)
	svc := NewServiceExtended(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	supplierA := newEinkaufExtendedTestSupplier(t, repo, ctx, tenantID, "Child Parent Supplier A")
	supplierB := newEinkaufExtendedTestSupplier(t, repo, ctx, tenantID, "Child Parent Supplier B")
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM supplier_ratings WHERE supplier_id IN ($1, $2)`, supplierA.ID, supplierB.ID)
		testutil.CleanupRow(t, pool, "suppliers", supplierA.ID)
		testutil.CleanupRow(t, pool, "suppliers", supplierB.ID)
	})

	t.Run("po lines", func(t *testing.T) {
		newPO := func() *PurchaseOrder {
			po, err := svc.CreatePO(ctx, CreatePOInput{TenantID: tenantID, SupplierID: supplierA.ID, PONumber: "PO-CP-" + uuid.New().String()[:8], Currency: "EUR"})
			require.NoError(t, err)
			t.Cleanup(func() {
				_, _ = pool.Exec(ctx, `DELETE FROM po_lines WHERE po_id = $1`, po.ID)
				testutil.CleanupRow(t, pool, "purchase_orders", po.ID)
			})
			return po
		}
		poA, poB := newPO(), newPO()
		line, err := svc.AddPOLine(ctx, AddPOLineInput{TenantID: tenantID, POID: poA.ID, ProductName: "Teil", Quantity: "2", UnitPrice: "10.00", TaxRate: "0.19", LinePosition: 1})
		require.NoError(t, err)

		qty := "9"
		_, err = svc.UpdatePOLine(ctx, UpdatePOLineInput{TenantID: tenantID, POID: poB.ID, LineID: line.ID, Quantity: &qty})
		require.ErrorIs(t, err, ErrPOLineNotFound)
		require.ErrorIs(t, svc.DeletePOLine(ctx, tenantID, poB.ID, line.ID), ErrPOLineNotFound)

		stored, err := repo.GetPOLine(ctx, tenantID, line.ID)
		require.NoError(t, err, "line must survive the foreign-path delete")
		assert.Equal(t, poA.ID, stored.POID)
		assert.Equal(t, "2.0000", stored.Quantity, "foreign-path update must not write")

		updated, err := svc.UpdatePOLine(ctx, UpdatePOLineInput{TenantID: tenantID, POID: poA.ID, LineID: line.ID, Quantity: &qty})
		require.NoError(t, err)
		assert.Equal(t, "9", updated.Quantity)
		require.NoError(t, svc.DeletePOLine(ctx, tenantID, poA.ID, line.ID))
		_, err = repo.GetPOLine(ctx, tenantID, line.ID)
		require.ErrorIs(t, err, ErrPOLineNotFound)
	})

	t.Run("contract items", func(t *testing.T) {
		newContract := func() *FrameworkContract {
			fc := &FrameworkContract{
				ID: uuid.New(), TenantID: tenantID, SupplierID: supplierA.ID,
				Title: "RV", ContractNr: "FC-CP-" + uuid.New().String()[:8],
				TotalValue: "5000.0000", UsedValue: "0.0000", Currency: "EUR",
				Status: ContractStatusActive, CreatedAt: now, UpdatedAt: now,
			}
			require.NoError(t, repo.CreateFrameworkContract(ctx, fc))
			t.Cleanup(func() {
				_, _ = pool.Exec(ctx, `DELETE FROM framework_contract_items WHERE contract_id = $1`, fc.ID)
				testutil.CleanupRow(t, pool, "framework_contracts", fc.ID)
			})
			return fc
		}
		fcA, fcB := newContract(), newContract()
		item, err := svc.CreateContractItem(ctx, CreateContractItemInput{TenantID: tenantID, ContractID: fcA.ID, Name: "Schraube", UnitPrice: "0.10", AgreedQty: "1000"})
		require.NoError(t, err)

		name := "Fremd"
		_, err = svc.UpdateContractItem(ctx, UpdateContractItemInput{TenantID: tenantID, ContractID: fcB.ID, ItemID: item.ID, Name: &name})
		require.ErrorIs(t, err, ErrContractItemNotFound)
		require.ErrorIs(t, svc.DeleteContractItem(ctx, tenantID, fcB.ID, item.ID), ErrContractItemNotFound)

		stored, err := repo.QueryRowContractItem(ctx, tenantID, item.ID)
		require.NoError(t, err, "item must survive the foreign-path delete")
		assert.Equal(t, "Schraube", stored.Name, "foreign-path update must not write")

		updated, err := svc.UpdateContractItem(ctx, UpdateContractItemInput{TenantID: tenantID, ContractID: fcA.ID, ItemID: item.ID, Name: &name})
		require.NoError(t, err)
		assert.Equal(t, "Fremd", updated.Name)
		require.NoError(t, svc.DeleteContractItem(ctx, tenantID, fcA.ID, item.ID))
		_, err = repo.QueryRowContractItem(ctx, tenantID, item.ID)
		require.ErrorIs(t, err, ErrContractItemNotFound)
	})

	t.Run("supplier ratings", func(t *testing.T) {
		rating, err := svc.CreateSupplierRating(ctx, CreateSupplierRatingInput{TenantID: tenantID, SupplierID: supplierA.ID, Category: RatingCategoryQuality, Rating: 4})
		require.NoError(t, err)

		require.ErrorIs(t, svc.DeleteSupplierRating(ctx, tenantID, supplierB.ID, rating.ID), ErrSupplierRatingNotFound)
		_, err = repo.GetSupplierRating(ctx, tenantID, rating.ID)
		require.NoError(t, err, "rating must survive the foreign-path delete")

		require.NoError(t, svc.DeleteSupplierRating(ctx, tenantID, supplierA.ID, rating.ID))
		_, err = repo.GetSupplierRating(ctx, tenantID, rating.ID)
		require.ErrorIs(t, err, ErrSupplierRatingNotFound)
	})
}
