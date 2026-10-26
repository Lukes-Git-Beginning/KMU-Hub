package einkauf

// The framework-contract binding against the real schema as kmuhub_app: the
// repository persists framework_contract_id on create and update, and the
// service refuses a contract the caller's tenant cannot see. The column's FK
// alone would not stop that — referential checks ignore RLS.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPOFrameworkContract_PersistsAndStaysInTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf PO Contract Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf PO Contract Other Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewServiceExtended(repo)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctxOwn, tenantOwn, "PO Contract Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)
	otherSupplier := newEinkaufExtendedTestSupplier(t, repo, ctxOther, tenantOther, "PO Contract Other Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", otherSupplier.ID)

	contract := func(ctx context.Context, tenantID, supplierID uuid.UUID, currency string) *FrameworkContract {
		fc := &FrameworkContract{
			ID: uuid.New(), TenantID: tenantID, SupplierID: supplierID,
			Title: "RV " + currency, ContractNr: "FC-" + uuid.New().String()[:8],
			TotalValue: "5000.0000", UsedValue: "0.0000", Currency: currency,
			Status: ContractStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, repo.CreateFrameworkContract(ctx, fc))
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "framework_contracts", fc.ID) })
		return fc
	}
	eur := contract(ctxOwn, tenantOwn, supplier.ID, "EUR")
	eur2 := contract(ctxOwn, tenantOwn, supplier.ID, "EUR")
	chf := contract(ctxOwn, tenantOwn, supplier.ID, "CHF")
	foreign := contract(ctxOther, tenantOther, otherSupplier.ID, "EUR")

	stored := func(poID uuid.UUID) *uuid.UUID {
		t.Helper()
		po, err := repo.GetPO(ctxOwn, tenantOwn, poID)
		require.NoError(t, err)
		return po.FrameworkContractID
	}

	// Create: a foreign-tenant contract reads as not found, a CHF one as a mismatch.
	_, err := svc.CreatePO(ctxOwn, CreatePOInput{TenantID: tenantOwn, SupplierID: supplier.ID, PONumber: "PO-FC-X-" + uuid.New().String()[:8], Currency: "EUR", FrameworkContractID: &foreign.ID})
	require.ErrorIs(t, err, ErrContractNotFound)
	_, err = svc.CreatePO(ctxOwn, CreatePOInput{TenantID: tenantOwn, SupplierID: supplier.ID, PONumber: "PO-FC-Y-" + uuid.New().String()[:8], Currency: "EUR", FrameworkContractID: &chf.ID})
	require.ErrorIs(t, err, ErrPOContractCurrencyMismatch)

	po, err := svc.CreatePO(ctxOwn, CreatePOInput{TenantID: tenantOwn, SupplierID: supplier.ID, PONumber: "PO-FC-" + uuid.New().String()[:8], Currency: "EUR", FrameworkContractID: &eur.ID})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "purchase_orders", po.ID) })
	require.NotNil(t, stored(po.ID))
	assert.Equal(t, eur.ID, *stored(po.ID), "CreatePO must persist the binding")

	// Update: rebinding lands, a rejected one leaves the stored binding alone.
	_, err = svc.UpdatePO(ctxOwn, UpdatePOInput{TenantID: tenantOwn, POID: po.ID, FrameworkContractID: &eur2.ID})
	require.NoError(t, err)
	assert.Equal(t, eur2.ID, *stored(po.ID), "UpdatePO must persist the new binding")

	_, err = svc.UpdatePO(ctxOwn, UpdatePOInput{TenantID: tenantOwn, POID: po.ID, FrameworkContractID: &foreign.ID})
	require.ErrorIs(t, err, ErrContractNotFound)
	usd := "USD"
	_, err = svc.UpdatePO(ctxOwn, UpdatePOInput{TenantID: tenantOwn, POID: po.ID, Currency: &usd})
	require.ErrorIs(t, err, ErrPOContractCurrencyMismatch)
	assert.Equal(t, eur2.ID, *stored(po.ID), "a rejected update must not touch the stored binding")

	unbind := uuid.Nil
	_, err = svc.UpdatePO(ctxOwn, UpdatePOInput{TenantID: tenantOwn, POID: po.ID, FrameworkContractID: &unbind})
	require.NoError(t, err)
	assert.Nil(t, stored(po.ID), "uuid.Nil must clear the binding")

	// The foreign tenant cannot see the PO, bound or not.
	_, err = repo.GetPO(ctxOther, tenantOther, po.ID)
	require.ErrorIs(t, err, ErrPONotFound)
}
