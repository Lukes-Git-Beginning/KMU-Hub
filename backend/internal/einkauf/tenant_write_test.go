package einkauf

// Nacht 1 added the PO-cancel and line-mutation write paths (CreatePOLine,
// UpdatePOLine, DeletePOLine, RecomputePOTotal, UpdatePOStatus) but the only
// existing DB test (tenant_isolation_phase2_test.go) seeds rows via
// testutil.SeedRow (a raw INSERT under system context) and never calls the
// real repository -- exactly the shape of gap Nacht 1 found elsewhere. This
// file closes that gap: every write below already carries an explicit
// tenant_id predicate (no dead write found), but that was unproven against a
// real repository call. Each write is exercised once from a foreign-tenant
// ctx, passing the row's *real* tenantID as the explicit parameter -- so only
// RLS, not the WHERE clause, could be stopping it -- before repeating the
// same call from the owning ctx and confirming it lands.

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSupplierWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf Supplier Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf Supplier Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	s := &Supplier{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Name:      "Write Test Supplier",
		Email:     "supplier@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateSupplier(ctxOwn, s); err != nil {
		t.Fatalf("CreateSupplier: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "suppliers", s.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "suppliers", s.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "suppliers", s.ID, 0)

	// UpdateSupplier carries an explicit tenant_id predicate, but pass the
	// row's real tenantID from a foreign ctx so only RLS -- not the WHERE
	// clause -- can be stopping it.
	foreign := *s
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateSupplier(ctxOther, &foreign); err != nil {
		t.Fatalf("UpdateSupplier (foreign ctx): %v", err)
	}
	got, err := repo.GetSupplier(sysCtx, tenantOwn, s.ID)
	if err != nil {
		t.Fatalf("GetSupplier: %v", err)
	}
	if got.Name != "Write Test Supplier" {
		t.Fatalf("a foreign-tenant write reached the supplier: name=%q", got.Name)
	}

	foreign.Name = "Renamed"
	if err := repo.UpdateSupplier(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateSupplier (own ctx): %v", err)
	}
	got, err = repo.GetSupplier(ctxOwn, tenantOwn, s.ID)
	if err != nil {
		t.Fatalf("GetSupplier (own ctx): %v", err)
	}
	if got.Name != "Renamed" {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	// DeleteSupplier: foreign ctx must not remove the row, own ctx must
	// (soft delete via deleted_at).
	if err := repo.DeleteSupplier(ctxOther, tenantOwn, s.ID); err == nil {
		t.Fatalf("DeleteSupplier (foreign ctx): expected an error, got nil")
	}
	if _, err := repo.GetSupplier(sysCtx, tenantOwn, s.ID); err != nil {
		t.Fatalf("supplier should still be retrievable after a foreign-tenant delete attempt: %v", err)
	}
	if err := repo.DeleteSupplier(ctxOwn, tenantOwn, s.ID); err != nil {
		t.Fatalf("DeleteSupplier (own ctx): %v", err)
	}
	if _, err := repo.GetSupplier(sysCtx, tenantOwn, s.ID); err != ErrSupplierNotFound {
		t.Fatalf("own-tenant delete did not land: err=%v", err)
	}
}

func TestPurchaseOrderWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf PO Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf PO Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	supplier := &Supplier{
		ID: uuid.New(), TenantID: tenantOwn, Name: "PO Write Supplier",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSupplier(ctxOwn, supplier); err != nil {
		t.Fatalf("CreateSupplier: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	po := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		PONumber: "PO-" + uuid.New().String()[:8], Status: POStatusDraft,
		OrderDate: now, TotalAmount: "0", Currency: "EUR",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePO(ctxOwn, po); err != nil {
		t.Fatalf("CreatePO: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "purchase_orders", po.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "purchase_orders", po.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "purchase_orders", po.ID, 0)

	// UpdatePO
	foreign := *po
	foreign.Notes = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdatePO(ctxOther, &foreign); err == nil {
		t.Fatalf("UpdatePO (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetPO(sysCtx, tenantOwn, po.ID)
	if err != nil {
		t.Fatalf("GetPO: %v", err)
	}
	if got.Notes == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the PO")
	}
	foreign.Notes = "Updated"
	if err := repo.UpdatePO(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdatePO (own ctx): %v", err)
	}
	got, _ = repo.GetPO(ctxOwn, tenantOwn, po.ID)
	if got.Notes != "Updated" {
		t.Fatalf("own-tenant write did not land: notes=%q", got.Notes)
	}

	// UpdatePOStatus
	if err := repo.UpdatePOStatus(ctxOther, tenantOwn, po.ID, POStatusSubmitted); err == nil {
		t.Fatalf("UpdatePOStatus (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetPO(sysCtx, tenantOwn, po.ID)
	if got.Status == POStatusSubmitted {
		t.Fatalf("a foreign-tenant status write reached the PO")
	}
	if err := repo.UpdatePOStatus(ctxOwn, tenantOwn, po.ID, POStatusSubmitted); err != nil {
		t.Fatalf("UpdatePOStatus (own ctx): %v", err)
	}
	got, _ = repo.GetPO(ctxOwn, tenantOwn, po.ID)
	if got.Status != POStatusSubmitted {
		t.Fatalf("own-tenant status write did not land: status=%q", got.Status)
	}

	// --- PO lines ---
	line := &POLine{
		ID: uuid.New(), TenantID: tenantOwn, POID: po.ID,
		ProductName: "Screw M6", Quantity: "10.0000", UnitPrice: "0.5000",
		TaxRate: "0", ReceivedQuantity: "0",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePOLine(ctxOwn, line); err != nil {
		t.Fatalf("CreatePOLine: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "po_lines", line.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "po_lines", line.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "po_lines", line.ID, 0)

	foreignLine := *line
	foreignLine.ProductName = "Hacked Line"
	foreignLine.UpdatedAt = time.Now().UTC()
	if err := repo.UpdatePOLine(ctxOther, &foreignLine); err != nil {
		t.Fatalf("UpdatePOLine (foreign ctx): %v", err)
	}
	gotLine, err := repo.GetPOLine(sysCtx, tenantOwn, line.ID)
	if err != nil {
		t.Fatalf("GetPOLine: %v", err)
	}
	if gotLine.ProductName == "Hacked Line" {
		t.Fatalf("a foreign-tenant write reached the PO line")
	}
	foreignLine.ProductName = "Updated Line"
	if err := repo.UpdatePOLine(ctxOwn, &foreignLine); err != nil {
		t.Fatalf("UpdatePOLine (own ctx): %v", err)
	}
	gotLine, _ = repo.GetPOLine(ctxOwn, tenantOwn, line.ID)
	if gotLine.ProductName != "Updated Line" {
		t.Fatalf("own-tenant write did not land: product_name=%q", gotLine.ProductName)
	}

	// UpdatePOLineReceivedQuantity
	if err := repo.UpdatePOLineReceivedQuantity(ctxOther, tenantOwn, line.ID, "5.0000"); err != nil {
		t.Fatalf("UpdatePOLineReceivedQuantity (foreign ctx): %v", err)
	}
	gotLine, _ = repo.GetPOLine(sysCtx, tenantOwn, line.ID)
	if gotLine.ReceivedQuantity == "5.0000" {
		t.Fatalf("a foreign-tenant write reached po_lines.received_quantity")
	}
	if err := repo.UpdatePOLineReceivedQuantity(ctxOwn, tenantOwn, line.ID, "5.0000"); err != nil {
		t.Fatalf("UpdatePOLineReceivedQuantity (own ctx): %v", err)
	}
	gotLine, _ = repo.GetPOLine(ctxOwn, tenantOwn, line.ID)
	if gotLine.ReceivedQuantity != "5.0000" {
		t.Fatalf("own-tenant write did not land: received_quantity=%q", gotLine.ReceivedQuantity)
	}

	// RecomputePOTotal
	if err := repo.RecomputePOTotal(ctxOther, tenantOwn, po.ID); err == nil {
		t.Fatalf("RecomputePOTotal (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetPO(sysCtx, tenantOwn, po.ID)
	if got.TotalAmount != "0" {
		t.Fatalf("a foreign-tenant recompute reached the PO total: total_amount=%q", got.TotalAmount)
	}
	if err := repo.RecomputePOTotal(ctxOwn, tenantOwn, po.ID); err != nil {
		t.Fatalf("RecomputePOTotal (own ctx): %v", err)
	}
	got, _ = repo.GetPO(ctxOwn, tenantOwn, po.ID)
	if got.TotalAmount == "0" {
		t.Fatalf("own-tenant recompute did not land: total_amount=%q", got.TotalAmount)
	}

	// DeletePOLine: foreign ctx must not remove the row, own ctx must.
	if err := repo.DeletePOLine(ctxOther, tenantOwn, line.ID); err != nil {
		t.Fatalf("DeletePOLine (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "po_lines", line.ID, 1)
	if err := repo.DeletePOLine(ctxOwn, tenantOwn, line.ID); err != nil {
		t.Fatalf("DeletePOLine (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "po_lines", line.ID, 0)

	// DeletePO requires status='draft'; UpdatePOStatus above moved it to
	// 'submitted', so revert to draft first via the same repo method.
	if err := repo.UpdatePOStatus(ctxOwn, tenantOwn, po.ID, POStatusDraft); err != nil {
		t.Fatalf("UpdatePOStatus (reset to draft): %v", err)
	}
	if err := repo.DeletePO(ctxOther, tenantOwn, po.ID); err == nil {
		t.Fatalf("DeletePO (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "purchase_orders", po.ID, 1)
	if err := repo.DeletePO(ctxOwn, tenantOwn, po.ID); err != nil {
		t.Fatalf("DeletePO (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "purchase_orders", po.ID, 0)
}

// TestRecomputePOTotal_SumsRawWithoutPerLineRounding proves the Bug-Suche
// hypothesis from BACKLOG.yml (cov-einkauf-service-extended-real-paths):
// "pruefen, ob Positionspreise mal Menge je Zeile gerundet werden, bevor sie
// in die Bestellsumme gehen" -- the same class of bug Lauf 11 found in the
// GoBD export (fix-gobd-export-tax-grouping-rate-key-collision).
//
// Two lines each total 0.125 (qty=1 x price=0.125). A per-line-rounded
// implementation would round each line to 0.13 (or 0.12) before summing,
// landing on 0.26 (or 0.24). RecomputePOTotal instead runs
// SUM(quantity * unit_price) inside Postgres against NUMERIC(15,4) columns
// (total_amount is NUMERIC(15,4) too, see migrations/000085_create_einkauf),
// so the raw, unrounded 0.25 is what lands in total_amount. Result: no bug
// found -- documented here, not filed as a fix-unit.
func TestRecomputePOTotal_SumsRawWithoutPerLineRounding(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Einkauf Rounding Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	supplier := &Supplier{
		ID: uuid.New(), TenantID: tenantID, Name: "Rounding Supplier",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSupplier(ctx, supplier); err != nil {
		t.Fatalf("CreateSupplier: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	po := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantID, SupplierID: supplier.ID,
		PONumber: "PO-ROUND-" + uuid.New().String()[:8], Status: POStatusDraft,
		OrderDate: now, TotalAmount: "0", Currency: "EUR",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePO(ctx, po); err != nil {
		t.Fatalf("CreatePO: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "purchase_orders", po.ID)

	for i := range 2 {
		line := &POLine{
			ID: uuid.New(), TenantID: tenantID, POID: po.ID,
			ProductName: "Fractional Line", Quantity: "1.0000", UnitPrice: "0.1250",
			TaxRate: "0", ReceivedQuantity: "0",
			CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreatePOLine(ctx, line); err != nil {
			t.Fatalf("CreatePOLine %d: %v", i, err)
		}
		defer testutil.CleanupRow(t, pool, "po_lines", line.ID)
	}

	if err := repo.RecomputePOTotal(ctx, tenantID, po.ID); err != nil {
		t.Fatalf("RecomputePOTotal: %v", err)
	}
	got, err := repo.GetPO(ctx, tenantID, po.ID)
	if err != nil {
		t.Fatalf("GetPO: %v", err)
	}

	total, err := strconv.ParseFloat(got.TotalAmount, 64)
	if err != nil {
		t.Fatalf("total_amount %q did not parse as a number: %v", got.TotalAmount, err)
	}
	if total != 0.25 {
		t.Fatalf("expected raw-sum total 0.25 (would be 0.26 or 0.24 if lines were rounded to cents before summing), got %v (total_amount=%q)", total, got.TotalAmount)
	}
}

