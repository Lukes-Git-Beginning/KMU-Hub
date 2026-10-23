package inventar

// RecordMovement must change the stock and write the movement row in one
// transaction. The mock cannot prove that (it shares no SQL with production),
// so this runs the service against the real schema.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRecordMovement_ChangesStockAgainstRealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Record Movement Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	item := seedPickingTxItem(t, repo, ctx, pool, tenantID, 10)

	quantity := func() int64 {
		t.Helper()
		reloaded, err := repo.GetItem(ctx, tenantID, item.ID)
		if err != nil {
			t.Fatalf("GetItem: %v", err)
		}
		return reloaded.Quantity
	}
	movementCount := func() int {
		t.Helper()
		_, total, err := repo.ListMovements(ctx, tenantID, item.ID, 0, 50)
		if err != nil {
			t.Fatalf("ListMovements: %v", err)
		}
		return total
	}

	in, err := svc.RecordMovement(ctx, RecordMovementInput{
		TenantID: tenantID, ItemID: item.ID, MovementType: MovementTypeIn, Quantity: 5, Reason: "Wareneingang",
	})
	if err != nil {
		t.Fatalf("RecordMovement in: %v", err)
	}
	if in.Quantity != 5 || in.MovementType != MovementTypeIn {
		t.Fatalf("unexpected inbound movement: %+v", in)
	}
	if got := quantity(); got != 15 {
		t.Fatalf("in 5 on 10: quantity = %d, want 15", got)
	}
	if got := movementCount(); got != 1 {
		t.Fatalf("after in: %d movement rows, want exactly 1", got)
	}

	_, err = svc.RecordMovement(ctx, RecordMovementInput{
		TenantID: tenantID, ItemID: item.ID, MovementType: MovementTypeOut, Quantity: 20,
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("out 20 on 15: err = %v, want ErrInsufficientStock", err)
	}
	if got := quantity(); got != 15 {
		t.Fatalf("rejected out changed the stock: %d, want 15", got)
	}
	if got := movementCount(); got != 1 {
		t.Fatalf("rejected out wrote a row: %d movement rows, want 1", got)
	}

	out, err := svc.RecordMovement(ctx, RecordMovementInput{
		TenantID: tenantID, ItemID: item.ID, MovementType: MovementTypeOut, Quantity: 6,
	})
	if err != nil {
		t.Fatalf("RecordMovement out: %v", err)
	}
	if out.Quantity != -6 {
		t.Fatalf("outbound movement quantity = %d, want -6", out.Quantity)
	}
	if got := quantity(); got != 9 {
		t.Fatalf("out 6 on 15: quantity = %d, want 9", got)
	}
	if got := movementCount(); got != 2 {
		t.Fatalf("after out: %d movement rows, want 2", got)
	}
}
