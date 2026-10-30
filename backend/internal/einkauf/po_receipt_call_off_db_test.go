package einkauf

// Goods receipts of a PO bound to a framework contract, against the real
// schema as kmuhub_app: every receipt books a call-off for the net value of
// what newly arrived (never the full PO value twice), the budget is soft, and
// a contract that cannot take a call-off never blocks the receipt.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestGoodsReceipt_BooksContractCallOffs(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Einkauf Receipt Call-off Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewServiceExtended(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctx, tenantID, "Receipt Call-off Supplier")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "suppliers", supplier.ID) })

	contract := func(t *testing.T, total string, status ContractStatus) *FrameworkContract {
		t.Helper()
		fc := &FrameworkContract{
			ID: uuid.New(), TenantID: tenantID, SupplierID: supplier.ID,
			Title: "RV Abruf", ContractNr: "FC-" + uuid.New().String()[:8],
			TotalValue: total, UsedValue: "0.0000", Currency: "EUR",
			Status: status, CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, repo.CreateFrameworkContract(ctx, fc))
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "framework_contracts", fc.ID) })
		return fc
	}
	// boundPO creates a submitted PO bound to fc (nil = unbound) with the given
	// lines (quantity, unit price) and returns it with its line IDs.
	boundPO := func(t *testing.T, fc *FrameworkContract, lines ...[2]string) (*PurchaseOrder, []uuid.UUID) {
		t.Helper()
		in := CreatePOInput{TenantID: tenantID, SupplierID: supplier.ID, PONumber: "PO-RC-" + uuid.New().String()[:8], Currency: "EUR"}
		if fc != nil {
			in.FrameworkContractID = &fc.ID
		}
		po, err := svc.CreatePO(ctx, in)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = pool.Exec(ctx, `DELETE FROM framework_contract_calls WHERE po_id = $1`, po.ID)
			testutil.CleanupRow(t, pool, "purchase_orders", po.ID)
		})
		ids := make([]uuid.UUID, 0, len(lines))
		for i, l := range lines {
			line, err := svc.AddPOLine(ctx, AddPOLineInput{TenantID: tenantID, POID: po.ID, ProductName: "Teil", Quantity: l[0], UnitPrice: l[1], TaxRate: "0.19", LinePosition: i + 1})
			require.NoError(t, err)
			ids = append(ids, line.ID)
		}
		_, err = svc.SubmitPO(ctx, tenantID, po.ID)
		require.NoError(t, err)
		return po, ids
	}
	usedValue := func(t *testing.T, fc *FrameworkContract) string {
		t.Helper()
		got, err := svc.GetFrameworkContract(ctx, tenantID, fc.ID)
		require.NoError(t, err)
		return got.UsedValue
	}

	t.Run("each delivery calls off only the value that newly arrived", func(t *testing.T) {
		fc := contract(t, "1000.0000", ContractStatusActive)
		po, ids := boundPO(t, fc, [2]string{"10", "12.50"}, [2]string{"4", "100"})

		// First delivery: 4 of line 1 -> 4 * 12.50.
		got, err := svc.PartialReceive(ctx, tenantID, po.ID, []PartialReceiveItem{{LineID: ids[0], ReceivedQuantity: "4"}})
		require.NoError(t, err)
		assert.Equal(t, POStatusPartiallyReceived, got.Status)
		require.NotNil(t, got.ReceiptCallOff)
		require.NotNil(t, got.ReceiptCallOff.CallID)
		assert.Equal(t, "50.0000", got.ReceiptCallOff.Amount)
		assert.Equal(t, "1000.0000", got.ReceiptCallOff.Remaining)
		assert.False(t, got.ReceiptCallOff.BudgetExceeded)
		assert.Empty(t, got.ReceiptCallOff.Warning)
		assert.Equal(t, "50.0000", usedValue(t, fc))

		// Second delivery, cumulative quantities: line 1 to 10 (+6 * 12.50), line 2 to 1 (+1 * 100).
		got, err = svc.PartialReceive(ctx, tenantID, po.ID, []PartialReceiveItem{
			{LineID: ids[0], ReceivedQuantity: "10"},
			{LineID: ids[1], ReceivedQuantity: "1"},
		})
		require.NoError(t, err)
		require.NotNil(t, got.ReceiptCallOff)
		assert.Equal(t, "175.0000", got.ReceiptCallOff.Amount, "only the delta of the second delivery")
		assert.Equal(t, "225.0000", usedValue(t, fc))

		// Repeating the same cumulative quantities adds nothing and books nothing.
		got, err = svc.PartialReceive(ctx, tenantID, po.ID, []PartialReceiveItem{{LineID: ids[0], ReceivedQuantity: "10"}})
		require.NoError(t, err)
		assert.Nil(t, got.ReceiptCallOff)

		// The final full receipt books the 3 outstanding of line 2, not the PO total.
		got, err = svc.ReceiveGoods(ctx, tenantID, po.ID)
		require.NoError(t, err)
		assert.Equal(t, POStatusReceived, got.Status)
		require.NotNil(t, got.ReceiptCallOff)
		assert.Equal(t, "300.0000", got.ReceiptCallOff.Amount)
		assert.Equal(t, "525.0000", usedValue(t, fc), "used_value equals the PO's net total after the last delivery")

		calls, err := svc.ListContractCalls(ctx, tenantID, fc.ID)
		require.NoError(t, err)
		require.Len(t, calls, 3)
		for _, c := range calls {
			require.NotNil(t, c.POID)
			assert.Equal(t, po.ID, *c.POID)
			assert.Equal(t, "EUR", c.Currency)
		}
	})

	t.Run("a full receipt above the remaining value is booked and reported", func(t *testing.T) {
		fc := contract(t, "100.0000", ContractStatusActive)
		po, _ := boundPO(t, fc, [2]string{"3", "50"})

		got, err := svc.ReceiveGoods(ctx, tenantID, po.ID)
		require.NoError(t, err, "the receipt never fails on the budget")
		assert.Equal(t, POStatusReceived, got.Status)
		require.NotNil(t, got.ReceiptCallOff)
		require.NotNil(t, got.ReceiptCallOff.CallID, "the call-off is booked despite the budget")
		assert.True(t, got.ReceiptCallOff.BudgetExceeded)
		assert.Equal(t, "150.0000", got.ReceiptCallOff.Amount)
		assert.Equal(t, "100.0000", got.ReceiptCallOff.Remaining)
		assert.Contains(t, got.ReceiptCallOff.Warning, "exceeds")
		assert.Equal(t, "150.0000", usedValue(t, fc))

		// The manual call-off keeps its hard cap: the contract is now over budget.
		_, err = svc.CreateContractCall(ctx, CreateContractCallInput{TenantID: tenantID, ContractID: fc.ID, Amount: "1"})
		require.ErrorIs(t, err, ErrContractBudgetExceeded)
	})

	t.Run("a contract that cannot take a call-off never blocks the receipt", func(t *testing.T) {
		fc := contract(t, "1000.0000", ContractStatusDraft)
		po, _ := boundPO(t, fc, [2]string{"2", "10"})

		got, err := svc.ReceiveGoods(ctx, tenantID, po.ID)
		require.NoError(t, err)
		assert.Equal(t, POStatusReceived, got.Status)
		require.NotNil(t, got.ReceiptCallOff)
		assert.Nil(t, got.ReceiptCallOff.CallID)
		assert.Contains(t, got.ReceiptCallOff.Warning, "skipped")
		assert.Equal(t, "0.0000", usedValue(t, fc))

		stored, err := repo.GetPO(ctx, tenantID, po.ID)
		require.NoError(t, err)
		assert.Equal(t, POStatusReceived, stored.Status)
	})

	t.Run("an unbound PO books nothing", func(t *testing.T) {
		po, _ := boundPO(t, nil, [2]string{"2", "10"})
		got, err := svc.ReceiveGoods(ctx, tenantID, po.ID)
		require.NoError(t, err)
		assert.Nil(t, got.ReceiptCallOff)
	})

	t.Run("a line of another PO rolls the whole receipt back", func(t *testing.T) {
		fc := contract(t, "1000.0000", ContractStatusActive)
		po, ids := boundPO(t, fc, [2]string{"5", "10"})
		_, otherIDs := boundPO(t, nil, [2]string{"5", "10"})

		_, _, err := repo.ApplyGoodsReceipt(ctx, &GoodsReceipt{
			TenantID: tenantID, POID: po.ID, Status: POStatusReceived,
			Lines: []PartialReceiveItem{{LineID: ids[0], ReceivedQuantity: "5"}, {LineID: otherIDs[0], ReceivedQuantity: "5"}},
		})
		require.ErrorIs(t, err, ErrPOLineNotFound)

		_, err = svc.PartialReceive(ctx, tenantID, po.ID, []PartialReceiveItem{{LineID: otherIDs[0], ReceivedQuantity: "1"}})
		require.ErrorIs(t, err, ErrPOLineNotFound)

		stored, err := repo.GetPOWithLines(ctx, tenantID, po.ID)
		require.NoError(t, err)
		assert.Equal(t, POStatusSubmitted, stored.Status)
		require.Len(t, stored.Lines, 1)
		assert.True(t, numericEqual("0", stored.Lines[0].ReceivedQuantity), "got %s", stored.Lines[0].ReceivedQuantity)
		assert.Equal(t, "0.0000", usedValue(t, fc))
	})

	t.Run("a PO that is no longer receivable is refused inside the transaction", func(t *testing.T) {
		po, ids := boundPO(t, nil, [2]string{"1", "10"})
		_, err := svc.CancelPO(ctx, tenantID, po.ID)
		require.NoError(t, err)
		_, _, err = repo.ApplyGoodsReceipt(ctx, &GoodsReceipt{
			TenantID: tenantID, POID: po.ID, Status: POStatusReceived,
			Lines: []PartialReceiveItem{{LineID: ids[0], ReceivedQuantity: "1"}},
		})
		require.ErrorIs(t, err, ErrPONotReceivable)
	})
}
