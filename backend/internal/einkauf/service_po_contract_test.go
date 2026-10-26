package einkauf

// A purchase order binds to a framework contract only within its tenant and
// currency (feat-einkauf-po-framework-contract-field). The DB-side proof that
// the binding is persisted and scoped lives in po_contract_db_test.go.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePO_FrameworkContractBinding(t *testing.T) {
	ctx := context.Background()
	tenantID, otherTenant := uuid.New(), uuid.New()

	for _, tc := range []struct {
		name     string
		contract func(repo *mockRepositoryExtended, supplierID uuid.UUID) uuid.UUID
		currency string
		wantErr  error
	}{
		{"own contract, same currency", func(repo *mockRepositoryExtended, s uuid.UUID) uuid.UUID {
			return addFrameworkContract(repo, tenantID, s, "RV", "RV-1", ContractStatusActive).ID
		}, "EUR", nil},
		{"currency compared case-insensitively", func(repo *mockRepositoryExtended, s uuid.UUID) uuid.UUID {
			return addFrameworkContract(repo, tenantID, s, "RV", "RV-1", ContractStatusActive).ID
		}, "eur", nil},
		{"default PO currency EUR matches", func(repo *mockRepositoryExtended, s uuid.UUID) uuid.UUID {
			return addFrameworkContract(repo, tenantID, s, "RV", "RV-1", ContractStatusActive).ID
		}, "", nil},
		{"contract in another currency", func(repo *mockRepositoryExtended, s uuid.UUID) uuid.UUID {
			fc := addFrameworkContract(repo, tenantID, s, "RV", "RV-1", ContractStatusActive)
			fc.Currency = "CHF"
			return fc.ID
		}, "EUR", ErrPOContractCurrencyMismatch},
		{"contract of another tenant", func(repo *mockRepositoryExtended, s uuid.UUID) uuid.UUID {
			return addFrameworkContract(repo, otherTenant, s, "RV", "RV-1", ContractStatusActive).ID
		}, "EUR", ErrContractNotFound},
		{"unknown contract", func(*mockRepositoryExtended, uuid.UUID) uuid.UUID { return uuid.New() }, "EUR", ErrContractNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepositoryExtended()
			svc := NewServiceExtended(repo)
			supplier := addSupplier(repo.mockRepository, tenantID, "Lieferant")
			contractID := tc.contract(repo, supplier.ID)

			po, err := svc.CreatePO(ctx, CreatePOInput{
				TenantID: tenantID, SupplierID: supplier.ID, PONumber: "PO-1",
				Currency: tc.currency, FrameworkContractID: &contractID,
			})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Empty(t, repo.pos, "a rejected binding must not create the PO")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, po.FrameworkContractID)
			assert.Equal(t, contractID, *po.FrameworkContractID)
			assert.Equal(t, contractID, *repo.pos[po.ID].FrameworkContractID)
		})
	}
}

func TestCreatePO_WithoutContractStaysUnbound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)
	tenantID := uuid.New()
	supplier := addSupplier(repo.mockRepository, tenantID, "Lieferant")

	po, err := svc.CreatePO(context.Background(), CreatePOInput{TenantID: tenantID, SupplierID: supplier.ID, PONumber: "PO-1", Currency: "USD"})
	require.NoError(t, err)
	assert.Nil(t, po.FrameworkContractID)
}

func TestCreatePO_ContractNeedsExtendedService(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()
	supplier := addSupplier(repo, tenantID, "Lieferant")
	id := uuid.New()

	_, err := svc.CreatePO(context.Background(), CreatePOInput{TenantID: tenantID, SupplierID: supplier.ID, PONumber: "PO-1", FrameworkContractID: &id})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdatePO_FrameworkContractBinding(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	setup := func(t *testing.T) (*Service, *mockRepositoryExtended, *PurchaseOrder, *FrameworkContract, *FrameworkContract) {
		t.Helper()
		repo := newMockRepositoryExtended()
		supplier := addSupplier(repo.mockRepository, tenantID, "Lieferant")
		po := addPO(repo.mockRepository, tenantID, supplier.ID, "PO-1", POStatusDraft)
		po.Currency = "EUR"
		eur := addFrameworkContract(repo, tenantID, supplier.ID, "RV EUR", "RV-1", ContractStatusActive)
		chf := addFrameworkContract(repo, tenantID, supplier.ID, "RV CHF", "RV-2", ContractStatusActive)
		chf.Currency = "CHF"
		return NewServiceExtended(repo), repo, po, eur, chf
	}
	ptr := func(id uuid.UUID) *uuid.UUID { return &id }
	str := func(s string) *string { return &s }

	t.Run("bind, then clear with uuid.Nil", func(t *testing.T) {
		svc, _, po, eur, _ := setup(t)
		got, err := svc.UpdatePO(ctx, UpdatePOInput{TenantID: tenantID, POID: po.ID, FrameworkContractID: ptr(eur.ID)})
		require.NoError(t, err)
		require.NotNil(t, got.FrameworkContractID)
		assert.Equal(t, eur.ID, *got.FrameworkContractID)

		got, err = svc.UpdatePO(ctx, UpdatePOInput{TenantID: tenantID, POID: po.ID, FrameworkContractID: ptr(uuid.Nil)})
		require.NoError(t, err)
		assert.Nil(t, got.FrameworkContractID)
	})

	t.Run("binding a contract in another currency", func(t *testing.T) {
		svc, _, po, _, chf := setup(t)
		_, err := svc.UpdatePO(ctx, UpdatePOInput{TenantID: tenantID, POID: po.ID, FrameworkContractID: ptr(chf.ID)})
		require.ErrorIs(t, err, ErrPOContractCurrencyMismatch)
	})

	t.Run("binding and switching currency together", func(t *testing.T) {
		svc, _, po, _, chf := setup(t)
		got, err := svc.UpdatePO(ctx, UpdatePOInput{TenantID: tenantID, POID: po.ID, Currency: str("CHF"), FrameworkContractID: ptr(chf.ID)})
		require.NoError(t, err)
		assert.Equal(t, chf.ID, *got.FrameworkContractID)
	})

	t.Run("currency change on a bound PO", func(t *testing.T) {
		svc, _, po, eur, _ := setup(t)
		po.FrameworkContractID = ptr(eur.ID)
		_, err := svc.UpdatePO(ctx, UpdatePOInput{TenantID: tenantID, POID: po.ID, Currency: str("USD")})
		require.ErrorIs(t, err, ErrPOContractCurrencyMismatch)
	})

	t.Run("foreign-tenant contract", func(t *testing.T) {
		svc, repo, po, _, _ := setup(t)
		foreign := addFrameworkContract(repo, uuid.New(), po.SupplierID, "Fremd", "RV-9", ContractStatusActive)
		_, err := svc.UpdatePO(ctx, UpdatePOInput{TenantID: tenantID, POID: po.ID, FrameworkContractID: ptr(foreign.ID)})
		require.ErrorIs(t, err, ErrContractNotFound)
	})

	t.Run("untouched binding is not re-checked", func(t *testing.T) {
		svc, repo, po, eur, _ := setup(t)
		po.FrameworkContractID = ptr(eur.ID)
		// The contract vanishing must not block an unrelated edit (the FK
		// clears the column on delete; this guards against a stale lookup).
		delete(repo.contracts, eur.ID)
		got, err := svc.UpdatePO(ctx, UpdatePOInput{TenantID: tenantID, POID: po.ID, Notes: str("nur Notiz")})
		require.NoError(t, err)
		assert.Equal(t, "nur Notiz", got.Notes)
	})

	t.Run("re-sending the same binding and currency", func(t *testing.T) {
		svc, _, po, eur, _ := setup(t)
		po.FrameworkContractID = ptr(eur.ID)
		eur.Currency = "CHF" // contract currency drifted after binding
		_, err := svc.UpdatePO(ctx, UpdatePOInput{TenantID: tenantID, POID: po.ID, Currency: str("EUR"), FrameworkContractID: ptr(eur.ID)})
		require.NoError(t, err, "an echo of the stored values must stay editable")
	})

	t.Run("closed PO rejects before any contract lookup", func(t *testing.T) {
		svc, _, po, eur, _ := setup(t)
		po.Status = POStatusClosed
		_, err := svc.UpdatePO(ctx, UpdatePOInput{TenantID: tenantID, POID: po.ID, FrameworkContractID: ptr(eur.ID)})
		require.ErrorIs(t, err, ErrPONotDraft)
	})
}
