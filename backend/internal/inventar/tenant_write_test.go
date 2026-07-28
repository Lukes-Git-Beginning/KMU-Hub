package inventar

// Exercises the real write path (PostgresRepository, not testutil.SeedRow) so a
// forgotten tenant_id column would show up as a genuine INSERT/constraint failure
// instead of being silently bypassed by a system-context fixture. Complements
// tenant_isolation_phase2_test.go, which only proves RLS SELECT filtering on
// hand-seeded rows.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestInventarWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Inventar Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Inventar Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	item := &Item{
		ID:          uuid.New(),
		TenantID:    tenantWrite,
		Name:        "Widget",
		SKU:         "SKU-" + uuid.New().String()[:8],
		Quantity:    10,
		MinQuantity: 2,
		Unit:        "Stk",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.CreateItem(ctxA, item); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "inventory_items", item.ID)

	movement := &Movement{
		ID:           uuid.New(),
		TenantID:     tenantWrite,
		ItemID:       item.ID,
		MovementType: MovementTypeIn,
		Quantity:     5,
		CreatedAt:    now,
	}
	if err := repo.CreateMovement(ctxA, movement); err != nil {
		t.Fatalf("CreateMovement: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "inventory_movements", movement.ID)

	warning := &Warning{
		ID:              uuid.New(),
		TenantID:        tenantWrite,
		ItemID:          item.ID,
		Threshold:       2,
		CurrentQuantity: 1,
		Status:          WarningStatusActive,
		CreatedAt:       now,
	}
	if err := repo.CreateWarning(ctxA, warning); err != nil {
		t.Fatalf("CreateWarning: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "stock_warnings", warning.ID)

	loc := &InventoryLocation{
		ID:        uuid.New(),
		TenantID:  tenantWrite,
		Name:      "Lager 1",
		Type:      LocationTypeWarehouse,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateLocation(ctxA, loc); err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "inventory_locations", loc.ID)

	session := &InventurSession{
		ID:        uuid.New(),
		TenantID:  tenantWrite,
		Name:      "Session KW01",
		Date:      now,
		Status:    InventurStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateInventurSession(ctxA, session); err != nil {
		t.Fatalf("CreateInventurSession: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "inventur_sessions", session.ID)

	count := &InventurCount{
		ID:        uuid.New(),
		TenantID:  tenantWrite,
		SessionID: session.ID,
		ItemID:    item.ID,
		Expected:  10,
	}
	if err := repo.UpsertInventurCount(ctxA, count); err != nil {
		t.Fatalf("UpsertInventurCount: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "inventur_counts", count.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"inventory_items", item.ID},
		{"inventory_movements", movement.ID},
		{"stock_warnings", warning.ID},
		{"inventory_locations", loc.ID},
		{"inventur_sessions", session.ID},
		{"inventur_counts", count.ID},
	}

	for _, row := range rows {
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}

	// ListInventurCounts previously read by session_id alone (no tenant_id
	// predicate) — relied entirely on RLS. Prove it now returns nothing under
	// the wrong tenant's ctx even though the session_id is known.
	counts, err := repo.ListInventurCounts(ctxB, tenantOther, session.ID)
	if err != nil {
		t.Fatalf("ListInventurCounts under foreign tenant: %v", err)
	}
	if len(counts) != 0 {
		t.Fatalf("ListInventurCounts under foreign tenant: expected 0 rows, got %d", len(counts))
	}
}
