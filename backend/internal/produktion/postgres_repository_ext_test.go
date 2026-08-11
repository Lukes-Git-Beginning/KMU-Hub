package produktion

// Exercises the BOM/WorkStep/Machine/QualityCheck write paths in
// postgres_repository_ext.go against the real PostgresRepository (not
// testutil.SeedRow), the same pattern as tenant_write_test.go and
// einkauf/tenant_write_test.go: a forgotten tenant_id predicate or a missing
// RLS policy shows up as a genuine cross-tenant leak instead of being masked
// by a system-context fixture.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestBOMWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Produktion BOM Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Produktion BOM Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()
	sku := "SKU-" + uuid.New().String()[:8]

	bom := &BOM{
		ID:          uuid.New(),
		TenantID:    tenantOwn,
		ProductName: "Widget Assembly",
		SKU:         sku,
		Version:     "1.0",
		Active:      true,
		Notes:       "initial",
		Items: []*BomItem{
			{ID: uuid.New(), TenantID: tenantOwn, MaterialName: "Screw M6", Quantity: 4, Unit: "Stk", SortOrder: 2},
			{ID: uuid.New(), TenantID: tenantOwn, MaterialName: "Bracket", Quantity: 1, Unit: "Stk", SortOrder: 1},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	for _, item := range bom.Items {
		item.BomID = bom.ID
	}
	if err := repo.CreateBOM(ctxOwn, bom); err != nil {
		t.Fatalf("CreateBOM: %v", err)
	}
	defer func() {
		testutil.CleanupRow(t, pool, "production_bom_items", bom.Items[0].ID)
		testutil.CleanupRow(t, pool, "production_bom_items", bom.Items[1].ID)
		testutil.CleanupRow(t, pool, "production_boms", bom.ID)
	}()

	// Real cross-tenant write proof: created under tenantOwn, must be
	// invisible under tenantOther via a real repository call, not a
	// hand-seeded fixture.
	testutil.AssertRowCount(t, pool, ctxOwn, "production_boms", bom.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "production_boms", bom.ID, 0)

	// Duplicate SKU in the same tenant is rejected.
	dup := &BOM{
		ID: uuid.New(), TenantID: tenantOwn, ProductName: "Other Product",
		SKU: sku, Version: "1.0", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateBOM(ctxOwn, dup); err != ErrBOMSKUTaken {
		t.Fatalf("CreateBOM duplicate sku: expected ErrBOMSKUTaken, got %v", err)
	}

	// GetBOM loads items ordered by sort_order ASC, not insertion order.
	got, err := repo.GetBOM(ctxOwn, tenantOwn, bom.ID)
	if err != nil {
		t.Fatalf("GetBOM: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("GetBOM: expected 2 items, got %d", len(got.Items))
	}
	if got.Items[0].MaterialName != "Bracket" || got.Items[1].MaterialName != "Screw M6" {
		t.Fatalf("GetBOM: items not ordered by sort_order, got %q then %q", got.Items[0].MaterialName, got.Items[1].MaterialName)
	}

	// GetBOM under a foreign tenant does not find the row.
	if _, err := repo.GetBOM(ctxOther, tenantOther, bom.ID); err != ErrBOMNotFound {
		t.Fatalf("GetBOM (foreign tenant): expected ErrBOMNotFound, got %v", err)
	}

	// ListBOMs returns the row for the owner and loads its items via the
	// bulk ANY($1) path.
	list, total, err := repo.ListBOMs(ctxOwn, tenantOwn, nil, 0, 10)
	if err != nil {
		t.Fatalf("ListBOMs: %v", err)
	}
	if total < 1 || len(list) < 1 {
		t.Fatalf("ListBOMs: expected at least 1 result, got total=%d len=%d", total, len(list))
	}
	var found *BOM
	for _, b := range list {
		if b.ID == bom.ID {
			found = b
		}
	}
	if found == nil {
		t.Fatalf("ListBOMs: created BOM not found in list")
	}
	if len(found.Items) != 2 {
		t.Fatalf("ListBOMs: expected bulk-loaded items len=2, got %d", len(found.Items))
	}

	// ListBOMs with activeOnly filter.
	activeOnly := true
	activeList, _, err := repo.ListBOMs(ctxOwn, tenantOwn, &activeOnly, 0, 10)
	if err != nil {
		t.Fatalf("ListBOMs (activeOnly): %v", err)
	}
	activeFound := false
	for _, b := range activeList {
		if b.ID == bom.ID {
			activeFound = true
		}
	}
	if !activeFound {
		t.Fatalf("ListBOMs (activeOnly): expected active BOM to be included")
	}

	// UpdateBOM from a foreign tenant context carries an explicit tenant_id
	// predicate and must not modify the row.
	foreign := *bom
	foreign.ProductName = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateBOM(ctxOther, &foreign); err != nil {
		t.Fatalf("UpdateBOM (foreign ctx): %v", err)
	}
	stillOwn, err := repo.GetBOM(sysCtx, tenantOwn, bom.ID)
	if err != nil {
		t.Fatalf("GetBOM (sys ctx): %v", err)
	}
	if stillOwn.ProductName == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the BOM")
	}

	foreign.ProductName = "Renamed"
	if err := repo.UpdateBOM(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateBOM (own ctx): %v", err)
	}
	updated, err := repo.GetBOM(ctxOwn, tenantOwn, bom.ID)
	if err != nil {
		t.Fatalf("GetBOM after update: %v", err)
	}
	if updated.ProductName != "Renamed" {
		t.Fatalf("own-tenant write did not land: product_name=%q", updated.ProductName)
	}

	// DeleteBOM.
	if err := repo.DeleteBOM(ctxOwn, tenantOwn, bom.ID); err != nil {
		t.Fatalf("DeleteBOM: %v", err)
	}
	if _, err := repo.GetBOM(sysCtx, tenantOwn, bom.ID); err != ErrBOMNotFound {
		t.Fatalf("GetBOM after delete: expected ErrBOMNotFound, got %v", err)
	}
}

func TestWorkStepWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Produktion WorkStep Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Produktion WorkStep Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	order := &ProductionOrder{
		ID: uuid.New(), TenantID: tenantOwn, OrderNumber: "ORD-" + uuid.New().String()[:8],
		ProductName: "Widget Y", Quantity: 3, Status: OrderStatusPlanned,
		PlannedStart: now, PlannedEnd: now.Add(24 * time.Hour), Priority: 3,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateOrder(ctxOwn, order); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_orders", order.ID)

	stepA := &WorkStep{
		ID: uuid.New(), TenantID: tenantOwn, OrderID: order.ID, StepNr: 2,
		Name: "Assemble", DurationMinutes: 30, Status: WorkStepStatusPending,
		CreatedAt: now, UpdatedAt: now,
	}
	stepB := &WorkStep{
		ID: uuid.New(), TenantID: tenantOwn, OrderID: order.ID, StepNr: 1,
		Name: "Prepare", DurationMinutes: 10, Status: WorkStepStatusPending,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateWorkStep(ctxOwn, stepA); err != nil {
		t.Fatalf("CreateWorkStep (A): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_work_steps", stepA.ID)
	if err := repo.CreateWorkStep(ctxOwn, stepB); err != nil {
		t.Fatalf("CreateWorkStep (B): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_work_steps", stepB.ID)

	testutil.AssertRowCount(t, pool, ctxOwn, "production_work_steps", stepA.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "production_work_steps", stepA.ID, 0)

	// ListWorkSteps orders by step_nr ASC, regardless of insertion order.
	steps, err := repo.ListWorkSteps(ctxOwn, tenantOwn, order.ID)
	if err != nil {
		t.Fatalf("ListWorkSteps: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("ListWorkSteps: expected 2 steps, got %d", len(steps))
	}
	if steps[0].ID != stepB.ID || steps[1].ID != stepA.ID {
		t.Fatalf("ListWorkSteps: expected step_nr ASC order (B=1 then A=2), got %q then %q", steps[0].Name, steps[1].Name)
	}

	// GetWorkStep not-found path.
	if _, err := repo.GetWorkStep(ctxOwn, tenantOwn, uuid.New()); err != ErrWorkStepNotFound {
		t.Fatalf("GetWorkStep (missing): expected ErrWorkStepNotFound, got %v", err)
	}

	// UpdateWorkStep from a foreign tenant context must not land.
	foreign := *stepA
	foreign.Status = WorkStepStatusCompleted
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateWorkStep(ctxOther, &foreign); err != nil {
		t.Fatalf("UpdateWorkStep (foreign ctx): %v", err)
	}
	stillPending, err := repo.GetWorkStep(ctxOwn, tenantOwn, stepA.ID)
	if err != nil {
		t.Fatalf("GetWorkStep: %v", err)
	}
	if stillPending.Status == WorkStepStatusCompleted {
		t.Fatalf("a foreign-tenant write reached the work step")
	}

	foreign.Status = WorkStepStatusInProgress
	if err := repo.UpdateWorkStep(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateWorkStep (own ctx): %v", err)
	}
	updated, err := repo.GetWorkStep(ctxOwn, tenantOwn, stepA.ID)
	if err != nil {
		t.Fatalf("GetWorkStep after update: %v", err)
	}
	if updated.Status != WorkStepStatusInProgress {
		t.Fatalf("own-tenant write did not land: status=%q", updated.Status)
	}

	// DeleteWorkStep.
	if err := repo.DeleteWorkStep(ctxOwn, tenantOwn, stepB.ID); err != nil {
		t.Fatalf("DeleteWorkStep: %v", err)
	}
	if _, err := repo.GetWorkStep(ctxOwn, tenantOwn, stepB.ID); err != ErrWorkStepNotFound {
		t.Fatalf("GetWorkStep after delete: expected ErrWorkStepNotFound, got %v", err)
	}
}

func TestMachineWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Produktion Machine Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Produktion Machine Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	m1 := &Machine{
		ID: uuid.New(), TenantID: tenantOwn, Name: "CNC-01", Type: "CNC",
		Status: MachineStatusAvailable, CreatedAt: now, UpdatedAt: now,
	}
	m2 := &Machine{
		ID: uuid.New(), TenantID: tenantOwn, Name: "CNC-02", Type: "CNC",
		Status: MachineStatusMaintenance, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateMachine(ctxOwn, m1); err != nil {
		t.Fatalf("CreateMachine (m1): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_machines", m1.ID)
	if err := repo.CreateMachine(ctxOwn, m2); err != nil {
		t.Fatalf("CreateMachine (m2): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_machines", m2.ID)

	testutil.AssertRowCount(t, pool, ctxOwn, "production_machines", m1.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "production_machines", m1.ID, 0)

	// GetMachine not-found path.
	if _, err := repo.GetMachine(ctxOwn, tenantOwn, uuid.New()); err != ErrMachineNotFound {
		t.Fatalf("GetMachine (missing): expected ErrMachineNotFound, got %v", err)
	}

	// ListMachines with status filter returns only the matching machine.
	maintenance := MachineStatusMaintenance
	filtered, total, err := repo.ListMachines(ctxOwn, tenantOwn, &maintenance, 0, 10)
	if err != nil {
		t.Fatalf("ListMachines (status filter): %v", err)
	}
	if total != 1 || len(filtered) != 1 || filtered[0].ID != m2.ID {
		t.Fatalf("ListMachines (status filter): expected exactly m2, got total=%d len=%d", total, len(filtered))
	}

	// UpdateMachine from a foreign tenant context must not land.
	foreign := *m1
	foreign.Status = MachineStatusInUse
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateMachine(ctxOther, &foreign); err != nil {
		t.Fatalf("UpdateMachine (foreign ctx): %v", err)
	}
	stillAvailable, err := repo.GetMachine(ctxOwn, tenantOwn, m1.ID)
	if err != nil {
		t.Fatalf("GetMachine: %v", err)
	}
	if stillAvailable.Status == MachineStatusInUse {
		t.Fatalf("a foreign-tenant write reached the machine")
	}

	if err := repo.UpdateMachine(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateMachine (own ctx): %v", err)
	}
	updated, err := repo.GetMachine(ctxOwn, tenantOwn, m1.ID)
	if err != nil {
		t.Fatalf("GetMachine after update: %v", err)
	}
	if updated.Status != MachineStatusInUse {
		t.Fatalf("own-tenant write did not land: status=%q", updated.Status)
	}

	// DeleteMachine.
	if err := repo.DeleteMachine(ctxOwn, tenantOwn, m2.ID); err != nil {
		t.Fatalf("DeleteMachine: %v", err)
	}
	if _, err := repo.GetMachine(ctxOwn, tenantOwn, m2.ID); err != ErrMachineNotFound {
		t.Fatalf("GetMachine after delete: expected ErrMachineNotFound, got %v", err)
	}
}

func TestQualityCheckWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Produktion QualityCheck Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Produktion QualityCheck Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	orderA := &ProductionOrder{
		ID: uuid.New(), TenantID: tenantOwn, OrderNumber: "ORD-" + uuid.New().String()[:8],
		ProductName: "Widget Z", Quantity: 2, Status: OrderStatusPlanned,
		PlannedStart: now, PlannedEnd: now.Add(24 * time.Hour), Priority: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	orderB := &ProductionOrder{
		ID: uuid.New(), TenantID: tenantOwn, OrderNumber: "ORD-" + uuid.New().String()[:8],
		ProductName: "Widget W", Quantity: 2, Status: OrderStatusPlanned,
		PlannedStart: now, PlannedEnd: now.Add(24 * time.Hour), Priority: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateOrder(ctxOwn, orderA); err != nil {
		t.Fatalf("CreateOrder (A): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_orders", orderA.ID)
	if err := repo.CreateOrder(ctxOwn, orderB); err != nil {
		t.Fatalf("CreateOrder (B): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_orders", orderB.ID)

	older := now.Add(-2 * time.Hour)
	newer := now
	checkOld := &QualityCheck{
		ID: uuid.New(), TenantID: tenantOwn, OrderID: orderA.ID, Inspector: "Alice",
		CheckedAt: older, Passed: true, CreatedAt: now, UpdatedAt: now,
	}
	checkNew := &QualityCheck{
		ID: uuid.New(), TenantID: tenantOwn, OrderID: orderA.ID, Inspector: "Bob",
		CheckedAt: newer, Passed: false, DefectsFound: 2, CreatedAt: now, UpdatedAt: now,
	}
	checkOtherOrder := &QualityCheck{
		ID: uuid.New(), TenantID: tenantOwn, OrderID: orderB.ID, Inspector: "Carol",
		CheckedAt: now, Passed: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateQualityCheck(ctxOwn, checkOld); err != nil {
		t.Fatalf("CreateQualityCheck (old): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_quality_checks", checkOld.ID)
	if err := repo.CreateQualityCheck(ctxOwn, checkNew); err != nil {
		t.Fatalf("CreateQualityCheck (new): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_quality_checks", checkNew.ID)
	if err := repo.CreateQualityCheck(ctxOwn, checkOtherOrder); err != nil {
		t.Fatalf("CreateQualityCheck (other order): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_quality_checks", checkOtherOrder.ID)

	testutil.AssertRowCount(t, pool, ctxOwn, "production_quality_checks", checkOld.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "production_quality_checks", checkOld.ID, 0)

	// GetQualityCheck not-found path.
	if _, err := repo.GetQualityCheck(ctxOwn, tenantOwn, uuid.New()); err != ErrQualityCheckNotFound {
		t.Fatalf("GetQualityCheck (missing): expected ErrQualityCheckNotFound, got %v", err)
	}

	// GetQualityCheck under a foreign tenant does not find the row.
	if _, err := repo.GetQualityCheck(ctxOther, tenantOther, checkOld.ID); err != ErrQualityCheckNotFound {
		t.Fatalf("GetQualityCheck (foreign tenant): expected ErrQualityCheckNotFound, got %v", err)
	}

	// ListQualityChecks filtered by order_id, ordered checked_at DESC.
	filtered, total, err := repo.ListQualityChecks(ctxOwn, tenantOwn, &orderA.ID, 0, 10)
	if err != nil {
		t.Fatalf("ListQualityChecks (order filter): %v", err)
	}
	if total != 2 || len(filtered) != 2 {
		t.Fatalf("ListQualityChecks (order filter): expected 2 results for orderA, got total=%d len=%d", total, len(filtered))
	}
	if filtered[0].ID != checkNew.ID || filtered[1].ID != checkOld.ID {
		t.Fatalf("ListQualityChecks: expected checked_at DESC order (new then old), got %q then %q", filtered[0].Inspector, filtered[1].Inspector)
	}

	// ListQualityChecks without an order filter still scopes to the tenant.
	all, allTotal, err := repo.ListQualityChecks(ctxOwn, tenantOwn, nil, 0, 10)
	if err != nil {
		t.Fatalf("ListQualityChecks (no filter): %v", err)
	}
	if allTotal < 3 || len(all) < 3 {
		t.Fatalf("ListQualityChecks (no filter): expected at least 3 results, got total=%d len=%d", allTotal, len(all))
	}
}
