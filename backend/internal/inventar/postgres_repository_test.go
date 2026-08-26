package inventar

// Covers the PostgresRepository methods left untested by tenant_write_test.go
// (plain INSERT paths) and picking_booking_tx_test.go (ON CONFLICT/claim
// semantics): Update/SoftDelete/Get/List across items, warnings, locations,
// inventur sessions and picking lists, plus SKUExists, item attachments and
// CompleteInventurSessionTx. Runs against the real schema, not a mock, so a
// forgotten WHERE clause or missing column shows up as a genuine query
// failure or wrong row count.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func newTestItem(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, name, sku string, qty, minQty int64) *Item {
	t.Helper()
	now := time.Now().UTC()
	item := &Item{
		ID: uuid.New(), TenantID: tenantID,
		Name: name, SKU: sku, Quantity: qty, MinQuantity: minQty, Unit: "Stk",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateItem(ctx, item); err != nil {
		t.Fatalf("seed item %s: %v", name, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_items", item.ID) })
	return item
}

func setupRepo(t *testing.T) (*PostgresRepository, *pgxpool.Pool, context.Context, uuid.UUID) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inventar Repo Test Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	return repo, pool, ctx, tenantID
}

// ============================================================================
// Items
// ============================================================================

func TestUpdateItem_UpdatesFieldsAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Schraube", "SKU-UPD-1", 10, 2)

	item.Name = "Schraube M10"
	item.Quantity = 25
	item.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateItem(ctx, item); err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}

	reloaded, err := repo.GetItem(ctx, tenantID, item.ID)
	if err != nil {
		t.Fatalf("GetItem after update: %v", err)
	}
	if reloaded.Name != "Schraube M10" || reloaded.Quantity != 25 {
		t.Fatalf("update did not persist: got name=%q quantity=%d", reloaded.Name, reloaded.Quantity)
	}

	unknown := &Item{ID: uuid.New(), TenantID: tenantID, Name: "x", SKU: "x", Unit: "Stk", UpdatedAt: time.Now().UTC()}
	if err := repo.UpdateItem(ctx, unknown); !errors.Is(err, ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound for unknown item, got %v", err)
	}
}

func TestSoftDeleteItem_HidesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Mutter", "SKU-DEL-1", 5, 1)

	if err := repo.SoftDeleteItem(ctx, tenantID, item.ID); err != nil {
		t.Fatalf("SoftDeleteItem: %v", err)
	}

	if _, err := repo.GetItem(ctx, tenantID, item.ID); !errors.Is(err, ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound after soft delete, got %v", err)
	}

	if err := repo.SoftDeleteItem(ctx, tenantID, item.ID); !errors.Is(err, ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound on double delete, got %v", err)
	}
}

func TestListItems_FiltersSearchLocationLowStockAndPaginates(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	warehouse := "Lager Nord"
	other := "Lager Sued"
	low := newTestItem(t, repo, ctx, pool, tenantID, "Akkuschrauber", "SKU-LIST-1", 1, 5)
	low.Location = &warehouse
	low.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateItem(ctx, low); err != nil {
		t.Fatalf("set location on low item: %v", err)
	}

	ok := newTestItem(t, repo, ctx, pool, tenantID, "Bohrer Set", "SKU-LIST-2", 50, 5)
	ok.Location = &other
	ok.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateItem(ctx, ok); err != nil {
		t.Fatalf("set location on ok item: %v", err)
	}

	t.Run("search matches name case-insensitively", func(t *testing.T) {
		items, total, err := repo.ListItems(ctx, tenantID, ListItemsFilter{Search: "akku"}, 0, 10)
		if err != nil {
			t.Fatalf("ListItems search: %v", err)
		}
		if total != 1 || len(items) != 1 || items[0].ID != low.ID {
			t.Fatalf("expected exactly the akku item, got total=%d items=%d", total, len(items))
		}
	})

	t.Run("location filter", func(t *testing.T) {
		items, total, err := repo.ListItems(ctx, tenantID, ListItemsFilter{Location: &other}, 0, 10)
		if err != nil {
			t.Fatalf("ListItems location: %v", err)
		}
		if total != 1 || len(items) != 1 || items[0].ID != ok.ID {
			t.Fatalf("expected exactly the other-location item, got total=%d items=%d", total, len(items))
		}
	})

	t.Run("low stock filter", func(t *testing.T) {
		items, total, err := repo.ListItems(ctx, tenantID, ListItemsFilter{LowStock: true}, 0, 10)
		if err != nil {
			t.Fatalf("ListItems low stock: %v", err)
		}
		if total != 1 || len(items) != 1 || items[0].ID != low.ID {
			t.Fatalf("expected exactly the low-stock item, got total=%d items=%d", total, len(items))
		}
	})

	t.Run("pagination reports full total with a smaller page", func(t *testing.T) {
		items, total, err := repo.ListItems(ctx, tenantID, ListItemsFilter{}, 0, 1)
		if err != nil {
			t.Fatalf("ListItems paginated: %v", err)
		}
		if total != 2 {
			t.Fatalf("expected total=2 regardless of page size, got %d", total)
		}
		if len(items) != 1 {
			t.Fatalf("expected exactly one item on the first page, got %d", len(items))
		}
	})

	t.Run("no match yields zero total", func(t *testing.T) {
		items, total, err := repo.ListItems(ctx, tenantID, ListItemsFilter{Search: "does-not-exist"}, 0, 10)
		if err != nil {
			t.Fatalf("ListItems no match: %v", err)
		}
		if total != 0 || len(items) != 0 {
			t.Fatalf("expected zero results, got total=%d items=%d", total, len(items))
		}
	})
}

func TestSKUExists_DetectsDuplicatesAndExcludesSelf(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Zange", "SKU-EXISTS-1", 3, 1)

	exists, err := repo.SKUExists(ctx, tenantID, "SKU-EXISTS-1", nil)
	if err != nil {
		t.Fatalf("SKUExists: %v", err)
	}
	if !exists {
		t.Fatal("expected SKU to exist")
	}

	notExists, err := repo.SKUExists(ctx, tenantID, "SKU-EXISTS-NOPE", nil)
	if err != nil {
		t.Fatalf("SKUExists (missing): %v", err)
	}
	if notExists {
		t.Fatal("expected unused SKU to not exist")
	}

	excluded, err := repo.SKUExists(ctx, tenantID, "SKU-EXISTS-1", &item.ID)
	if err != nil {
		t.Fatalf("SKUExists (excludeID): %v", err)
	}
	if excluded {
		t.Fatal("expected SKUExists to report false when excluding the item's own row")
	}
}

// ============================================================================
// Movements
// ============================================================================

func TestGetMovement_ReturnsRowOrNotFound(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Feile", "SKU-MOV-1", 20, 2)
	movement := &Movement{
		ID: uuid.New(), TenantID: tenantID, ItemID: item.ID,
		MovementType: MovementTypeIn, Quantity: 5, Reason: "Wareneingang",
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateMovement(ctx, movement); err != nil {
		t.Fatalf("CreateMovement: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_movements", movement.ID) })

	got, err := repo.GetMovement(ctx, tenantID, movement.ID)
	if err != nil {
		t.Fatalf("GetMovement: %v", err)
	}
	if got.Reason != "Wareneingang" || got.Quantity != 5 {
		t.Fatalf("unexpected movement row: %+v", got)
	}

	if _, err := repo.GetMovement(ctx, tenantID, uuid.New()); !errors.Is(err, ErrMovementNotFound) {
		t.Fatalf("expected ErrMovementNotFound, got %v", err)
	}
}

func TestListMovements_ReturnsRowsForItemOrdered(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Saege", "SKU-MOV-2", 20, 2)
	for i, reason := range []string{"first", "second"} {
		mv := &Movement{
			ID: uuid.New(), TenantID: tenantID, ItemID: item.ID,
			MovementType: MovementTypeIn, Quantity: int64(i + 1), Reason: reason,
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Second),
		}
		if err := repo.CreateMovement(ctx, mv); err != nil {
			t.Fatalf("CreateMovement %s: %v", reason, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_movements", mv.ID) })
	}

	movements, total, err := repo.ListMovements(ctx, tenantID, item.ID, 0, 10)
	if err != nil {
		t.Fatalf("ListMovements: %v", err)
	}
	if total != 2 || len(movements) != 2 {
		t.Fatalf("expected 2 movements, got total=%d items=%d", total, len(movements))
	}
	if movements[0].Reason != "second" {
		t.Fatalf("expected newest-first order, got %q first", movements[0].Reason)
	}
}

// ============================================================================
// Warnings
// ============================================================================

func TestWarningLifecycle_UpdateGetActiveAndList(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Hammer", "SKU-WARN-1", 1, 5)
	warning := &Warning{
		ID: uuid.New(), TenantID: tenantID, ItemID: item.ID,
		Threshold: 5, CurrentQuantity: 1, Status: WarningStatusActive,
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateWarning(ctx, warning); err != nil {
		t.Fatalf("CreateWarning: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "stock_warnings", warning.ID) })

	active, err := repo.GetActiveWarningForItem(ctx, tenantID, item.ID)
	if err != nil {
		t.Fatalf("GetActiveWarningForItem: %v", err)
	}
	if active.ID != warning.ID {
		t.Fatalf("expected active warning %s, got %s", warning.ID, active.ID)
	}

	now := time.Now().UTC()
	userID := uuid.New()
	warning.Status = WarningStatusAcknowledged
	warning.AcknowledgedAt = &now
	warning.AcknowledgedBy = &userID
	if err := repo.UpdateWarning(ctx, warning); err != nil {
		t.Fatalf("UpdateWarning: %v", err)
	}

	reloaded, err := repo.GetWarning(ctx, tenantID, warning.ID)
	if err != nil {
		t.Fatalf("GetWarning: %v", err)
	}
	if reloaded.Status != WarningStatusAcknowledged || reloaded.AcknowledgedBy == nil || *reloaded.AcknowledgedBy != userID {
		t.Fatalf("update did not persist: %+v", reloaded)
	}

	// No active warning remains for this item once it's acknowledged.
	if _, err := repo.GetActiveWarningForItem(ctx, tenantID, item.ID); !errors.Is(err, ErrWarningNotFound) {
		t.Fatalf("expected ErrWarningNotFound once acknowledged, got %v", err)
	}

	if _, err := repo.GetWarning(ctx, tenantID, uuid.New()); !errors.Is(err, ErrWarningNotFound) {
		t.Fatalf("expected ErrWarningNotFound for unknown id, got %v", err)
	}

	status := WarningStatusAcknowledged
	warnings, total, err := repo.ListWarnings(ctx, tenantID, &status, 0, 10)
	if err != nil {
		t.Fatalf("ListWarnings filtered: %v", err)
	}
	if total != 1 || len(warnings) != 1 || warnings[0].ID != warning.ID {
		t.Fatalf("expected exactly the acknowledged warning, got total=%d items=%d", total, len(warnings))
	}

	all, totalAll, err := repo.ListWarnings(ctx, tenantID, nil, 0, 10)
	if err != nil {
		t.Fatalf("ListWarnings unfiltered: %v", err)
	}
	if totalAll != 1 || len(all) != 1 {
		t.Fatalf("expected the one warning with no status filter, got total=%d items=%d", totalAll, len(all))
	}
}

// ============================================================================
// Locations
// ============================================================================

func TestLocationLifecycle_UpdateSoftDeleteAndList(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	now := time.Now().UTC()
	loc := &InventoryLocation{
		ID: uuid.New(), TenantID: tenantID, Name: "Lager A", Address: "Weg 1",
		Type: LocationTypeWarehouse, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateLocation(ctx, loc); err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_locations", loc.ID) })

	loc.Name = "Lager A Umbenannt"
	loc.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateLocation(ctx, loc); err != nil {
		t.Fatalf("UpdateLocation: %v", err)
	}

	reloaded, err := repo.GetLocation(ctx, tenantID, loc.ID)
	if err != nil {
		t.Fatalf("GetLocation: %v", err)
	}
	if reloaded.Name != "Lager A Umbenannt" {
		t.Fatalf("update did not persist, got name=%q", reloaded.Name)
	}

	locs, total, err := repo.ListLocations(ctx, tenantID, 0, 10)
	if err != nil {
		t.Fatalf("ListLocations: %v", err)
	}
	if total != 1 || len(locs) != 1 {
		t.Fatalf("expected exactly one location, got total=%d items=%d", total, len(locs))
	}

	if err := repo.SoftDeleteLocation(ctx, tenantID, loc.ID); err != nil {
		t.Fatalf("SoftDeleteLocation: %v", err)
	}
	if _, err := repo.GetLocation(ctx, tenantID, loc.ID); !errors.Is(err, ErrLocationNotFound) {
		t.Fatalf("expected ErrLocationNotFound after soft delete, got %v", err)
	}
	if err := repo.UpdateLocation(ctx, &InventoryLocation{ID: uuid.New(), TenantID: tenantID, UpdatedAt: time.Now().UTC()}); !errors.Is(err, ErrLocationNotFound) {
		t.Fatalf("expected ErrLocationNotFound updating unknown location, got %v", err)
	}
	if err := repo.SoftDeleteLocation(ctx, tenantID, loc.ID); !errors.Is(err, ErrLocationNotFound) {
		t.Fatalf("expected ErrLocationNotFound on double delete, got %v", err)
	}

	_, totalAfterDelete, err := repo.ListLocations(ctx, tenantID, 0, 10)
	if err != nil {
		t.Fatalf("ListLocations after delete: %v", err)
	}
	if totalAfterDelete != 0 {
		t.Fatalf("expected soft-deleted location to disappear from ListLocations, got total=%d", totalAfterDelete)
	}
}

// TestSoftDeleteLocation_ItemStillReferencingIt_LeavesDanglingLocationID
// documents the actual referential-integrity behaviour of SoftDeleteLocation
// (Service.DeleteLocation calls it directly, with no reference check first):
// the inventory_items.location_id FK carries ON DELETE SET NULL
// (migrations/000184_inventory_locations.up.sql), but that clause only fires
// on a hard DELETE. SoftDeleteLocation only sets deleted_at, so a hard DELETE
// never happens and the FK action never runs — an item's location_id keeps
// pointing at a location that GetLocation/ListLocations now treat as gone.
func TestSoftDeleteLocation_ItemStillReferencingIt_LeavesDanglingLocationID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	now := time.Now().UTC()
	loc := &InventoryLocation{
		ID: uuid.New(), TenantID: tenantID, Name: "Lager mit Bestand", Address: "Weg 2",
		Type: LocationTypeWarehouse, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateLocation(ctx, loc); err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_locations", loc.ID) })

	item := &Item{
		ID: uuid.New(), TenantID: tenantID,
		Name: "Referenziert Lager", SKU: "SKU-" + uuid.New().String()[:8],
		Quantity: 5, MinQuantity: 0, Unit: "Stk", LocationID: &loc.ID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateItem(ctx, item); err != nil {
		t.Fatalf("seed item: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_items", item.ID) })

	// SoftDeleteLocation succeeds unconditionally -- no check against items
	// still referencing this location.
	if err := repo.SoftDeleteLocation(ctx, tenantID, loc.ID); err != nil {
		t.Fatalf("SoftDeleteLocation: %v", err)
	}
	if _, err := repo.GetLocation(ctx, tenantID, loc.ID); !errors.Is(err, ErrLocationNotFound) {
		t.Fatalf("expected ErrLocationNotFound for the now-deleted location, got %v", err)
	}

	// The item is untouched: location_id still points at the deleted location
	// instead of being nulled out, because ON DELETE SET NULL never fired.
	reloaded, err := repo.GetItem(ctx, tenantID, item.ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if reloaded.LocationID == nil || *reloaded.LocationID != loc.ID {
		t.Fatalf("expected item.location_id to still be the deleted location %s (dangling reference), got %v", loc.ID, reloaded.LocationID)
	}
}

// ============================================================================
// Inventur Sessions
// ============================================================================

func TestInventurSessionLifecycle_UpdateGetListAndDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	now := time.Now().UTC()
	session := &InventurSession{
		ID: uuid.New(), TenantID: tenantID, Name: "Session A", Date: now,
		Status: InventurStatusOpen, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateInventurSession(ctx, session); err != nil {
		t.Fatalf("CreateInventurSession: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventur_sessions", session.ID) })

	session.Status = InventurStatusCounting
	session.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateInventurSession(ctx, session); err != nil {
		t.Fatalf("UpdateInventurSession: %v", err)
	}

	reloaded, err := repo.GetInventurSession(ctx, tenantID, session.ID)
	if err != nil {
		t.Fatalf("GetInventurSession: %v", err)
	}
	if reloaded.Status != InventurStatusCounting {
		t.Fatalf("update did not persist, got status=%q", reloaded.Status)
	}

	sessions, total, err := repo.ListInventurSessions(ctx, tenantID, 0, 10)
	if err != nil {
		t.Fatalf("ListInventurSessions: %v", err)
	}
	if total != 1 || len(sessions) != 1 {
		t.Fatalf("expected exactly one session, got total=%d items=%d", total, len(sessions))
	}

	if _, err := repo.GetInventurSession(ctx, tenantID, uuid.New()); !errors.Is(err, ErrInventurSessionNotFound) {
		t.Fatalf("expected ErrInventurSessionNotFound, got %v", err)
	}
	if err := repo.UpdateInventurSession(ctx, &InventurSession{ID: uuid.New(), TenantID: tenantID, UpdatedAt: time.Now().UTC()}); !errors.Is(err, ErrInventurSessionNotFound) {
		t.Fatalf("expected ErrInventurSessionNotFound updating unknown session, got %v", err)
	}

	if err := repo.DeleteInventurSession(ctx, tenantID, session.ID); err != nil {
		t.Fatalf("DeleteInventurSession: %v", err)
	}
	if _, err := repo.GetInventurSession(ctx, tenantID, session.ID); !errors.Is(err, ErrInventurSessionNotFound) {
		t.Fatalf("expected ErrInventurSessionNotFound after delete, got %v", err)
	}
	if err := repo.DeleteInventurSession(ctx, tenantID, session.ID); !errors.Is(err, ErrInventurSessionNotFound) {
		t.Fatalf("expected ErrInventurSessionNotFound on double delete, got %v", err)
	}
}

func TestCompleteInventurSessionTx_AppliesMovementsAndRejectsUnknownSession(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Winkelschleifer", "SKU-INV-TX-1", 30, 2)

	now := time.Now().UTC()
	session := &InventurSession{
		ID: uuid.New(), TenantID: tenantID, Name: "Session TX", Date: now,
		Status: InventurStatusReview, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateInventurSession(ctx, session); err != nil {
		t.Fatalf("CreateInventurSession: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventur_sessions", session.ID) })

	items, err := repo.CompleteInventurSessionTx(ctx, tenantID, session.ID, []StockMovementInput{
		{ItemID: item.ID, Delta: -10, MovementType: MovementTypeOut, Reason: "Inventurdifferenz"},
	})
	if err != nil {
		t.Fatalf("CompleteInventurSessionTx: %v", err)
	}
	if len(items) != 1 || items[0].Quantity != 20 {
		t.Fatalf("expected quantity 20 after completion, got %+v", items)
	}

	reloadedSession, err := repo.GetInventurSession(ctx, tenantID, session.ID)
	if err != nil {
		t.Fatalf("GetInventurSession after completion: %v", err)
	}
	if reloadedSession.Status != InventurStatusCompleted {
		t.Fatalf("expected session completed, got %q", reloadedSession.Status)
	}

	movements, totalMovements, err := repo.ListMovements(ctx, tenantID, item.ID, 0, 10)
	if err != nil {
		t.Fatalf("ListMovements after completion: %v", err)
	}
	if totalMovements != 1 || len(movements) != 1 {
		t.Fatalf("expected exactly one movement row, got total=%d", totalMovements)
	}

	if _, err := repo.CompleteInventurSessionTx(ctx, tenantID, uuid.New(), []StockMovementInput{
		{ItemID: item.ID, Delta: -1, MovementType: MovementTypeOut, Reason: "x"},
	}); !errors.Is(err, ErrInventurSessionNotFound) {
		t.Fatalf("expected ErrInventurSessionNotFound for unknown session, got %v", err)
	}
}

// ============================================================================
// Item Attachments
// ============================================================================

func TestItemAttachmentLifecycle_CreateListAndDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Bohrmaschine", "SKU-ATT-1", 5, 1)

	att := &ItemAttachment{
		ID: uuid.New(), TenantID: tenantID, ItemID: item.ID,
		Name: "datenblatt.pdf", ObjectKey: "att/" + uuid.New().String(), FileType: "application/pdf",
	}
	if err := repo.CreateItemAttachment(ctx, att); err != nil {
		t.Fatalf("CreateItemAttachment: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_item_attachments", att.ID) })
	if att.CreatedAt.IsZero() || att.UpdatedAt.IsZero() {
		t.Fatal("expected CreateItemAttachment to populate timestamps via RETURNING")
	}

	attachments, err := repo.ListItemAttachments(ctx, tenantID, item.ID)
	if err != nil {
		t.Fatalf("ListItemAttachments: %v", err)
	}
	if len(attachments) != 1 || attachments[0].ID != att.ID {
		t.Fatalf("expected exactly the created attachment, got %+v", attachments)
	}

	if err := repo.DeleteItemAttachment(ctx, tenantID, att.ID); err != nil {
		t.Fatalf("DeleteItemAttachment: %v", err)
	}

	afterDelete, err := repo.ListItemAttachments(ctx, tenantID, item.ID)
	if err != nil {
		t.Fatalf("ListItemAttachments after delete: %v", err)
	}
	if len(afterDelete) != 0 {
		t.Fatalf("expected no attachments after delete, got %d", len(afterDelete))
	}

	if err := repo.DeleteItemAttachment(ctx, tenantID, att.ID); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("expected ErrAttachmentNotFound on double delete, got %v", err)
	}
}

// ============================================================================
// Picking Lists
// ============================================================================

func TestPickingListLifecycle_UpdateListFilterAndDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	now := time.Now().UTC()
	list := &PickingList{
		ID: uuid.New(), TenantID: tenantID, Reference: "PL-1", Status: PickingStatusOpen,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePickingList(ctx, list); err != nil {
		t.Fatalf("CreatePickingList: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "picking_lists", list.ID) })

	list.Status = PickingStatusPicking
	list.Reference = "PL-1-Renamed"
	list.UpdatedAt = time.Now().UTC()
	if err := repo.UpdatePickingList(ctx, list); err != nil {
		t.Fatalf("UpdatePickingList: %v", err)
	}

	reloaded, err := repo.GetPickingList(ctx, tenantID, list.ID)
	if err != nil {
		t.Fatalf("GetPickingList: %v", err)
	}
	if reloaded.Status != PickingStatusPicking || reloaded.Reference != "PL-1-Renamed" {
		t.Fatalf("update did not persist: %+v", reloaded)
	}

	t.Run("status filter matches", func(t *testing.T) {
		status := PickingStatusPicking
		lists, total, err := repo.ListPickingLists(ctx, tenantID, &status, 0, 10)
		if err != nil {
			t.Fatalf("ListPickingLists filtered: %v", err)
		}
		if total != 1 || len(lists) != 1 || lists[0].ID != list.ID {
			t.Fatalf("expected exactly the picking list, got total=%d items=%d", total, len(lists))
		}
	})

	t.Run("status filter excludes other statuses", func(t *testing.T) {
		status := PickingStatusCompleted
		lists, total, err := repo.ListPickingLists(ctx, tenantID, &status, 0, 10)
		if err != nil {
			t.Fatalf("ListPickingLists filtered (no match): %v", err)
		}
		if total != 0 || len(lists) != 0 {
			t.Fatalf("expected zero completed lists, got total=%d items=%d", total, len(lists))
		}
	})

	t.Run("no filter returns all", func(t *testing.T) {
		lists, total, err := repo.ListPickingLists(ctx, tenantID, nil, 0, 10)
		if err != nil {
			t.Fatalf("ListPickingLists unfiltered: %v", err)
		}
		if total != 1 || len(lists) != 1 {
			t.Fatalf("expected the one list with no filter, got total=%d items=%d", total, len(lists))
		}
	})

	if err := repo.UpdatePickingList(ctx, &PickingList{ID: uuid.New(), TenantID: tenantID, UpdatedAt: time.Now().UTC()}); !errors.Is(err, ErrPickingListNotFound) {
		t.Fatalf("expected ErrPickingListNotFound updating unknown list, got %v", err)
	}

	if err := repo.DeletePickingList(ctx, tenantID, list.ID); err != nil {
		t.Fatalf("DeletePickingList: %v", err)
	}
	if _, err := repo.GetPickingList(ctx, tenantID, list.ID); !errors.Is(err, ErrPickingListNotFound) {
		t.Fatalf("expected ErrPickingListNotFound after delete, got %v", err)
	}
	if err := repo.DeletePickingList(ctx, tenantID, list.ID); !errors.Is(err, ErrPickingListNotFound) {
		t.Fatalf("expected ErrPickingListNotFound on double delete, got %v", err)
	}
}

func TestDeletePickingListItem_RemovesRowAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Karton", "SKU-PLI-1", 100, 5)
	now := time.Now().UTC()
	list := &PickingList{
		ID: uuid.New(), TenantID: tenantID, Reference: "PL-DEL", Status: PickingStatusOpen,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePickingList(ctx, list); err != nil {
		t.Fatalf("CreatePickingList: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "picking_lists", list.ID) })

	row := &PickingListItem{
		ID: uuid.New(), TenantID: tenantID, PickingListID: list.ID, ItemID: item.ID,
		QuantityRequested: 10,
	}
	if err := repo.UpsertPickingListItem(ctx, row); err != nil {
		t.Fatalf("UpsertPickingListItem: %v", err)
	}

	if err := repo.DeletePickingListItem(ctx, tenantID, row.ID); err != nil {
		t.Fatalf("DeletePickingListItem: %v", err)
	}

	remaining, err := repo.ListPickingListItems(ctx, tenantID, list.ID)
	if err != nil {
		t.Fatalf("ListPickingListItems after delete: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no picking list items after delete, got %d", len(remaining))
	}

	if err := repo.DeletePickingListItem(ctx, tenantID, row.ID); !errors.Is(err, ErrPickingListItemNotFound) {
		t.Fatalf("expected ErrPickingListItemNotFound on double delete, got %v", err)
	}
	if err := repo.DeletePickingListItem(ctx, tenantID, uuid.New()); !errors.Is(err, ErrPickingListItemNotFound) {
		t.Fatalf("expected ErrPickingListItemNotFound for unknown id, got %v", err)
	}
}
