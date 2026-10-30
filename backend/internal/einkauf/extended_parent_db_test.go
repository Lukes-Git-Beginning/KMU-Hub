package einkauf

// Catalog items, supplier ratings and framework contracts hang off a supplier,
// contract items off a contract: a deleted or unknown parent answers NotFound
// (not a 201 or a foreign-key 500), an expired contract takes no more items.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestExtendedCreates_RequireLiveParent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close) // registered first, so it runs after the row cleanups

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Einkauf Extended Parent Tenant")
	repo := NewPostgresRepository(pool)
	svc := NewServiceExtended(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	live := newEinkaufExtendedTestSupplier(t, repo, ctx, tenantID, "Live Supplier")
	gone := newEinkaufExtendedTestSupplier(t, repo, ctx, tenantID, "Deleted Supplier")
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM supplier_ratings WHERE supplier_id IN ($1, $2)`, live.ID, gone.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM supplier_catalog_items WHERE supplier_id IN ($1, $2)`, live.ID, gone.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM framework_contract_items WHERE tenant_id = $1`, tenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM framework_contracts WHERE supplier_id IN ($1, $2)`, live.ID, gone.ID)
		testutil.CleanupRow(t, pool, "suppliers", live.ID)
		testutil.CleanupRow(t, pool, "suppliers", gone.ID)
	})
	require.NoError(t, repo.DeleteSupplier(ctx, tenantID, gone.ID))

	for name, supplierID := range map[string]uuid.UUID{"deleted": gone.ID, "unknown": uuid.New()} {
		t.Run("supplier "+name, func(t *testing.T) {
			_, err := svc.CreateCatalogItem(ctx, CreateCatalogItemInput{TenantID: tenantID, SupplierID: supplierID, Name: "Teil", Price: "1.00"})
			require.ErrorIs(t, err, ErrSupplierNotFound)
			_, err = svc.CreateSupplierRating(ctx, CreateSupplierRatingInput{TenantID: tenantID, SupplierID: supplierID, Category: RatingCategoryQuality, Rating: 4})
			require.ErrorIs(t, err, ErrSupplierNotFound)
			_, err = svc.CreateFrameworkContract(ctx, CreateFrameworkContractInput{TenantID: tenantID, SupplierID: supplierID, Title: "RV"})
			require.ErrorIs(t, err, ErrSupplierNotFound)
		})
	}

	var rows int
	require.NoError(t, pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM supplier_catalog_items WHERE supplier_id = $1) +
		(SELECT count(*) FROM supplier_ratings WHERE supplier_id = $1) +
		(SELECT count(*) FROM framework_contracts WHERE supplier_id = $1)`, gone.ID).Scan(&rows))
	assert.Zero(t, rows, "nothing may be written for a deleted supplier")

	t.Run("live supplier takes all three", func(t *testing.T) {
		_, err := svc.CreateCatalogItem(ctx, CreateCatalogItemInput{TenantID: tenantID, SupplierID: live.ID, Name: "Teil", Price: "1.00"})
		require.NoError(t, err)
		_, err = svc.CreateSupplierRating(ctx, CreateSupplierRatingInput{TenantID: tenantID, SupplierID: live.ID, Category: RatingCategoryQuality, Rating: 4})
		require.NoError(t, err)
		_, err = svc.CreateFrameworkContract(ctx, CreateFrameworkContractInput{TenantID: tenantID, SupplierID: live.ID, Title: "RV"})
		require.NoError(t, err)
	})

	t.Run("contract items follow the contract status", func(t *testing.T) {
		now := time.Now().UTC()
		newContract := func(status ContractStatus) *FrameworkContract {
			fc := &FrameworkContract{
				ID: uuid.New(), TenantID: tenantID, SupplierID: live.ID,
				Title: "RV", ContractNr: "FC-EP-" + uuid.New().String()[:8],
				TotalValue: "5000.0000", UsedValue: "0.0000", Currency: "EUR",
				Status: status, CreatedAt: now, UpdatedAt: now,
			}
			require.NoError(t, repo.CreateFrameworkContract(ctx, fc))
			return fc
		}
		in := func(c *FrameworkContract) CreateContractItemInput {
			return CreateContractItemInput{TenantID: tenantID, ContractID: c.ID, Name: "Pos"}
		}

		expired := newContract(ContractStatusExpired)
		_, err := svc.CreateContractItem(ctx, in(expired))
		require.ErrorIs(t, err, ErrContractNotActive)
		var n int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM framework_contract_items WHERE contract_id = $1`, expired.ID).Scan(&n))
		assert.Zero(t, n, "expired contract must stay without new items")

		_, err = svc.CreateContractItem(ctx, CreateContractItemInput{TenantID: tenantID, ContractID: uuid.New(), Name: "Pos"})
		require.ErrorIs(t, err, ErrContractNotFound)

		for _, st := range []ContractStatus{ContractStatusDraft, ContractStatusActive} {
			_, err = svc.CreateContractItem(ctx, in(newContract(st)))
			require.NoError(t, err, "status %s", st)
		}
	})
}
