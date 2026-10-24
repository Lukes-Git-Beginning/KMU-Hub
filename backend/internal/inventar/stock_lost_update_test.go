package inventar

// Stock may only move through the locked movement transaction. An item edit
// reads the row without a lock and used to write its quantity back, undoing a
// movement committed in between (goods receipt via AdjustStock, a picking list,
// a manual in/out). Runs against the real schema: the mock shares no SQL.

import (
	"errors"
	"testing"
	"time"
)

func TestUpdateItem_StaleReadDoesNotUndoAStockMovement(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Kabel", "SKU-LOST-1", 10, 0)

	// The edit dialog read the item while the stock was still 10 ...
	stale, err := repo.GetItem(ctx, tenantID, item.ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	// ... a goods receipt books +5 ...
	if _, err := svc.AdjustStock(ctx, AdjustStockInput{TenantID: tenantID, ItemID: item.ID, Delta: 5, Reason: "Wareneingang"}); err != nil {
		t.Fatalf("AdjustStock: %v", err)
	}

	// ... and then the edit is saved.
	stale.Name = "Kabel 3x1,5"
	stale.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateItem(ctx, stale); err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}

	reloaded, err := repo.GetItem(ctx, tenantID, item.ID)
	if err != nil {
		t.Fatalf("GetItem after edit: %v", err)
	}
	if reloaded.Quantity != 15 || reloaded.Name != "Kabel 3x1,5" {
		t.Fatalf("got quantity=%d name=%q, want 15 and the new name", reloaded.Quantity, reloaded.Name)
	}
}

func TestTransferStock_MovesBothLegsOrNoneAgainstRealSchema(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	from := newTestItem(t, repo, ctx, pool, tenantID, "Lager A", "SKU-TR-A", 8, 0)
	to := newTestItem(t, repo, ctx, pool, tenantID, "Lager B", "SKU-TR-B", 1, 0)

	quantities := func() (int64, int64) {
		t.Helper()
		a, err := repo.GetItem(ctx, tenantID, from.ID)
		if err != nil {
			t.Fatalf("GetItem from: %v", err)
		}
		b, err := repo.GetItem(ctx, tenantID, to.ID)
		if err != nil {
			t.Fatalf("GetItem to: %v", err)
		}
		return a.Quantity, b.Quantity
	}

	if err := svc.TransferStock(ctx, TransferStockInput{TenantID: tenantID, FromItemID: from.ID, ToItemID: to.ID, Quantity: 3}); err != nil {
		t.Fatalf("TransferStock: %v", err)
	}
	if a, b := quantities(); a != 5 || b != 4 {
		t.Fatalf("after transfer of 3: from=%d to=%d, want 5 and 4", a, b)
	}
	for _, id := range []struct {
		name  string
		item  *Item
		delta int64
	}{{"from", from, -3}, {"to", to, 3}} {
		movements, total, err := repo.ListMovements(ctx, tenantID, id.item.ID, 0, 10)
		if err != nil {
			t.Fatalf("ListMovements %s: %v", id.name, err)
		}
		if total != 1 || movements[0].MovementType != MovementTypeTransfer || movements[0].Quantity != id.delta {
			t.Fatalf("%s movements: total=%d first=%+v, want one transfer of %d", id.name, total, movements, id.delta)
		}
	}

	// More than the source holds: neither item changes.
	err := svc.TransferStock(ctx, TransferStockInput{TenantID: tenantID, FromItemID: from.ID, ToItemID: to.ID, Quantity: 6})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("over-transfer: got %v, want ErrInsufficientStock", err)
	}
	if a, b := quantities(); a != 5 || b != 4 {
		t.Fatalf("after rejected transfer: from=%d to=%d, want 5 and 4 unchanged", a, b)
	}
}
