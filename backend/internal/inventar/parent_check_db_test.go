package inventar

// A child row must hang off a live parent. The FKs only catch a missing row, not
// a soft-deleted one, and UpsertInventurCount used to swallow the GetItem error
// and count a deleted item against Expected=0. Runs against the real schema.

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func newTestLocation(t *testing.T, repo *PostgresRepository, tenantID uuid.UUID, name string) *InventoryLocation {
	t.Helper()
	now := time.Now().UTC()
	loc := &InventoryLocation{ID: uuid.New(), TenantID: tenantID, Name: name, Type: "warehouse", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateLocation(testutil.WithTenantCtx(t.Context(), tenantID), loc); err != nil {
		t.Fatalf("seed location: %v", err)
	}
	return loc
}

func TestInventurCount_RequiresALiveItem(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	live := newTestItem(t, repo, ctx, pool, tenantID, "Schraube", "SKU-PC-LIVE", 7, 0)
	gone := newTestItem(t, repo, ctx, pool, tenantID, "Mutter", "SKU-PC-GONE", 4, 0)
	if err := repo.SoftDeleteItem(ctx, tenantID, gone.ID); err != nil {
		t.Fatalf("SoftDeleteItem: %v", err)
	}
	session, err := svc.CreateInventurSession(ctx, CreateInventurSessionInput{TenantID: tenantID, Name: "Jahresinventur"})
	if err != nil {
		t.Fatalf("CreateInventurSession: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventur_sessions", session.ID) })

	for name, itemID := range map[string]uuid.UUID{"deleted": gone.ID, "unknown": uuid.New()} {
		_, err := svc.UpsertInventurCount(ctx, UpsertInventurCountInput{TenantID: tenantID, SessionID: session.ID, ItemID: itemID, Counted: 3})
		if !errors.Is(err, ErrItemNotFound) {
			t.Errorf("%s item: err = %v, want ErrItemNotFound", name, err)
		}
	}

	got, err := svc.UpsertInventurCount(ctx, UpsertInventurCountInput{TenantID: tenantID, SessionID: session.ID, ItemID: live.ID, Counted: 5})
	if err != nil {
		t.Fatalf("count live item: %v", err)
	}
	if got.Expected != 7 {
		t.Errorf("Expected = %d, want 7 (the live item's stock)", got.Expected)
	}

	counts, err := repo.ListInventurCounts(ctx, tenantID, session.ID)
	if err != nil {
		t.Fatalf("ListInventurCounts: %v", err)
	}
	if len(counts) != 1 || counts[0].ItemID != live.ID {
		t.Errorf("counts = %d rows, want exactly the live item's", len(counts))
	}
}

func TestLocationReference_RequiresALiveLocation(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	live := newTestLocation(t, repo, tenantID, "Hauptlager")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_locations", live.ID) })
	gone := newTestLocation(t, repo, tenantID, "Altlager")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_locations", gone.ID) })
	if err := repo.SoftDeleteLocation(ctx, tenantID, gone.ID); err != nil {
		t.Fatalf("SoftDeleteLocation: %v", err)
	}
	unknown := uuid.New()

	for name, locID := range map[string]uuid.UUID{"deleted": gone.ID, "unknown": unknown} {
		_, err := svc.CreateItem(ctx, CreateItemInput{TenantID: tenantID, Name: "A", SKU: "SKU-LOC-" + name, LocationID: &locID})
		if !errors.Is(err, ErrLocationNotFound) {
			t.Errorf("CreateItem at %s location: err = %v, want ErrLocationNotFound", name, err)
		}
		_, err = svc.CreateInventurSession(ctx, CreateInventurSessionInput{TenantID: tenantID, Name: "Inv " + name, LocationID: &locID})
		if !errors.Is(err, ErrLocationNotFound) {
			t.Errorf("CreateInventurSession at %s location: err = %v, want ErrLocationNotFound", name, err)
		}
	}

	item, err := svc.CreateItem(ctx, CreateItemInput{TenantID: tenantID, Name: "B", SKU: "SKU-LOC-LIVE", LocationID: &live.ID})
	if err != nil {
		t.Fatalf("CreateItem at live location: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_items", item.ID) })

	for name, locID := range map[string]uuid.UUID{"deleted": gone.ID, "unknown": unknown} {
		_, err := svc.UpdateItem(ctx, UpdateItemInput{TenantID: tenantID, ItemID: item.ID, LocationID: &locID})
		if !errors.Is(err, ErrLocationNotFound) {
			t.Errorf("UpdateItem to %s location: err = %v, want ErrLocationNotFound", name, err)
		}
	}
	reloaded, err := repo.GetItem(ctx, tenantID, item.ID)
	if err != nil || reloaded.LocationID == nil || *reloaded.LocationID != live.ID {
		t.Errorf("item location after rejected updates = %v (err %v), want the live location", reloaded.LocationID, err)
	}

	if _, err := svc.CreateItem(ctx, CreateItemInput{TenantID: tenantID, Name: "C", SKU: "SKU-LOC-NONE"}); err != nil {
		t.Errorf("CreateItem without location: %v", err)
	}
}
