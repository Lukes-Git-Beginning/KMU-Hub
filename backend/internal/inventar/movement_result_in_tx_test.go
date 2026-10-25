package inventar

// AdjustStock and RecordMovement answer from what the booking transaction
// returned. They used to read the item/movement again after the commit: when
// that second read failed (pool, cancelled ctx), the caller got an error for a
// booking that had gone through, and a retry booked it twice.

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestAdjustStockAndRecordMovement_DoNotReadAfterTheBooking(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	repo := newMockRepository()
	item := &Item{ID: uuid.New(), TenantID: tenantID, Name: "Kabel", SKU: "SKU-TX-1", Quantity: 10}
	repo.items[item.ID] = item
	svc := NewService(repo)
	ctx := context.Background()

	// Every read after the booking fails; the booking itself must still answer.
	repo.getItemErr = errors.New("pool closed")
	repo.getMovementErr = errors.New("pool closed")

	adjusted, err := svc.AdjustStock(ctx, AdjustStockInput{TenantID: tenantID, ItemID: item.ID, Delta: 5, Reason: "Wareneingang"})
	if err != nil {
		t.Fatalf("AdjustStock with failing reads: %v", err)
	}
	if adjusted.Quantity != 15 || adjusted.Name != "Kabel" {
		t.Fatalf("AdjustStock returned %+v, want quantity 15 and the full item", adjusted)
	}

	mv, err := svc.RecordMovement(ctx, RecordMovementInput{TenantID: tenantID, ItemID: item.ID, MovementType: MovementTypeOut, Quantity: 4})
	if err != nil {
		t.Fatalf("RecordMovement with failing reads: %v", err)
	}
	if mv.Quantity != -4 || mv.MovementType != MovementTypeOut || mv.ItemID != item.ID {
		t.Fatalf("RecordMovement returned %+v, want the signed out movement", mv)
	}
	if item.Quantity != 11 {
		t.Fatalf("stock = %d, want 11 (booked exactly once)", item.Quantity)
	}
}

func TestStockBooking_ReturnsTheStoredItemAndMovementAgainstRealSchema(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Schraube", "SKU-TX-DB", 10, 0)

	adjusted, err := svc.AdjustStock(ctx, AdjustStockInput{TenantID: tenantID, ItemID: item.ID, Delta: 3, Reason: "Zugang"})
	if err != nil {
		t.Fatalf("AdjustStock: %v", err)
	}
	stored, err := repo.GetItem(ctx, tenantID, item.ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if !reflect.DeepEqual(adjusted, stored) {
		t.Fatalf("AdjustStock result differs from the stored item:\n got  %+v\n want %+v", adjusted, stored)
	}

	mv, err := svc.RecordMovement(ctx, RecordMovementInput{TenantID: tenantID, ItemID: item.ID, MovementType: MovementTypeOut, Quantity: 2, Reason: "Verbrauch"})
	if err != nil {
		t.Fatalf("RecordMovement: %v", err)
	}
	storedMv, err := repo.GetMovement(ctx, tenantID, mv.ID)
	if err != nil {
		t.Fatalf("GetMovement: %v", err)
	}
	if !reflect.DeepEqual(mv, storedMv) {
		t.Fatalf("RecordMovement result differs from the stored row:\n got  %+v\n want %+v", mv, storedMv)
	}
	if storedMv.Quantity != -2 {
		t.Fatalf("stored quantity = %d, want -2 (signed)", storedMv.Quantity)
	}
}
