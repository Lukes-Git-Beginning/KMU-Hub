package inventar

// BookPickingListTx and the picking_list_items upsert are the two pieces this
// fix depends on that a mock cannot prove: the ON CONFLICT upsert and the
// claim's conditional UPDATE only behave correctly against the real unique
// constraint and real transaction isolation, not against a Go map. This
// complements tenant_write_test.go (plain INSERT paths) and
// picking_service_test.go (service-level logic against the mock).

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedPickingTxItem(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, qty int64) *Item {
	t.Helper()
	now := time.Now().UTC()
	item := &Item{
		ID: uuid.New(), TenantID: tenantID,
		Name: "Picking Tx Item", SKU: "SKU-" + uuid.New().String()[:8],
		Quantity: qty, MinQuantity: 0, Unit: "Stk",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateItem(ctx, item); err != nil {
		t.Fatalf("seed item: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_items", item.ID) })
	return item
}

func TestBookPickingListTx_UpsertConflictAndClaimAgainstRealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Picking Tx Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	item := seedPickingTxItem(t, repo, ctx, pool, tenantID, 50)

	list := &PickingList{
		ID: uuid.New(), TenantID: tenantID, Reference: "TX-1", Status: PickingStatusOpen,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePickingList(ctx, list); err != nil {
		t.Fatalf("CreatePickingList: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "picking_lists", list.ID) })

	row := &PickingListItem{
		ID: uuid.New(), TenantID: tenantID, PickingListID: list.ID, ItemID: item.ID,
		QuantityRequested: 20, QuantityPicked: 10,
	}
	if err := repo.UpsertPickingListItem(ctx, row); err != nil {
		t.Fatalf("UpsertPickingListItem (insert): %v", err)
	}
	firstRowID := row.ID

	t.Run("ON CONFLICT updates in place, no duplicate row", func(t *testing.T) {
		again := &PickingListItem{
			ID: uuid.New(), TenantID: tenantID, PickingListID: list.ID, ItemID: item.ID,
			QuantityRequested: 20, QuantityPicked: 20,
		}
		if err := repo.UpsertPickingListItem(ctx, again); err != nil {
			t.Fatalf("UpsertPickingListItem (conflict): %v", err)
		}
		if again.ID != firstRowID {
			t.Fatalf("ON CONFLICT must keep the original row id, got %s want %s", again.ID, firstRowID)
		}
		items, err := repo.ListPickingListItems(ctx, tenantID, list.ID)
		if err != nil {
			t.Fatalf("ListPickingListItems: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("expected exactly one row after conflicting upsert, got %d", len(items))
		}
		if items[0].QuantityPicked != 20 {
			t.Fatalf("expected quantity_picked=20 after conflict update, got %d", items[0].QuantityPicked)
		}
	})

	t.Run("BookPickingListTx claims and moves stock atomically", func(t *testing.T) {
		claimed, items, err := repo.BookPickingListTx(ctx, tenantID, list.ID, []StockMovementInput{
			{ItemID: item.ID, Delta: -20, MovementType: MovementTypeOut, Reason: "test booking"},
		})
		if err != nil {
			t.Fatalf("BookPickingListTx: %v", err)
		}
		if !claimed {
			t.Fatalf("first booking must claim the list")
		}
		if len(items) != 1 || items[0].Quantity != 30 {
			t.Fatalf("expected quantity 30 after booking, got %+v", items)
		}

		reloaded, err := repo.GetItem(ctx, tenantID, item.ID)
		if err != nil {
			t.Fatalf("GetItem: %v", err)
		}
		if reloaded.Quantity != 30 {
			t.Fatalf("item quantity not persisted: got %d, want 30", reloaded.Quantity)
		}

		reloadedList, err := repo.GetPickingList(ctx, tenantID, list.ID)
		if err != nil {
			t.Fatalf("GetPickingList: %v", err)
		}
		if reloadedList.Status != PickingStatusCompleted {
			t.Fatalf("expected list completed, got %s", reloadedList.Status)
		}
	})

	t.Run("claim predicate makes a second booking a no-op", func(t *testing.T) {
		claimed, items, err := repo.BookPickingListTx(ctx, tenantID, list.ID, []StockMovementInput{
			{ItemID: item.ID, Delta: -20, MovementType: MovementTypeOut, Reason: "second attempt"},
		})
		if err != nil {
			t.Fatalf("BookPickingListTx (second): %v", err)
		}
		if claimed {
			t.Fatalf("second booking must not claim an already-completed list")
		}
		if items != nil {
			t.Fatalf("second booking must not report moved items, got %+v", items)
		}

		reloaded, err := repo.GetItem(ctx, tenantID, item.ID)
		if err != nil {
			t.Fatalf("GetItem: %v", err)
		}
		if reloaded.Quantity != 30 {
			t.Fatalf("stock moved a second time: got %d, want 30", reloaded.Quantity)
		}
	})
}

func TestBookPickingListTx_PartialFailureRollsBackClaimAndStock(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Picking Tx Rollback Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	ok := seedPickingTxItem(t, repo, ctx, pool, tenantID, 50)
	short := seedPickingTxItem(t, repo, ctx, pool, tenantID, 2)

	list := &PickingList{
		ID: uuid.New(), TenantID: tenantID, Reference: "TX-2", Status: PickingStatusOpen,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePickingList(ctx, list); err != nil {
		t.Fatalf("CreatePickingList: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "picking_lists", list.ID) })

	claimed, items, err := repo.BookPickingListTx(ctx, tenantID, list.ID, []StockMovementInput{
		{ItemID: ok.ID, Delta: -20, MovementType: MovementTypeOut, Reason: "test"},
		{ItemID: short.ID, Delta: -5, MovementType: MovementTypeOut, Reason: "test"},
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if claimed {
		t.Fatalf("a failed booking must not report claimed")
	}
	if items != nil {
		t.Fatalf("a failed booking must not report moved items, got %+v", items)
	}

	reloaded, err := repo.GetItem(ctx, tenantID, ok.ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if reloaded.Quantity != 50 {
		t.Fatalf("the bookable position moved despite the batch failing: got %d, want 50", reloaded.Quantity)
	}

	stillOpenList, err := repo.GetPickingList(ctx, tenantID, list.ID)
	if err != nil {
		t.Fatalf("GetPickingList: %v", err)
	}
	if stillOpenList.Status == PickingStatusCompleted {
		t.Fatalf("list must not be completed when the batch failed")
	}

	movements, total, err := repo.ListMovements(ctx, tenantID, ok.ID, 0, 10)
	if err != nil {
		t.Fatalf("ListMovements: %v", err)
	}
	if total != 0 || len(movements) != 0 {
		t.Fatalf("no movement rows may exist after a rolled-back batch, got %d", total)
	}
}
