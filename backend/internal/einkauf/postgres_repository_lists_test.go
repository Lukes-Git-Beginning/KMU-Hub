package einkauf

// tenant_write_test.go already exercises CreateSupplier/UpdatePO/... etc.
// against a real repository. This file closes the remaining gap: the
// read/list paths PONumberExists, ListPOs, ListSuppliers and
// GetPOWithLines never ran against a real repository call before -- only
// through the mocked Repository interface in service tests.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPONumberExists_TenantScopedAndExcludeID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf PONumberExists Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf PONumberExists Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	supplier := &Supplier{ID: uuid.New(), TenantID: tenantOwn, Name: "PONumberExists Supplier", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSupplier(ctxOwn, supplier); err != nil {
		t.Fatalf("CreateSupplier: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	poNumber := "PO-EXISTS-" + uuid.New().String()[:8]
	po := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		PONumber: poNumber, Status: POStatusDraft, OrderDate: now,
		TotalAmount: "0", Currency: "EUR", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePO(ctxOwn, po); err != nil {
		t.Fatalf("CreatePO: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "purchase_orders", po.ID)

	// A different PO in the same tenant, so excludeID has something real to
	// distinguish from.
	otherPO := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		PONumber: "PO-EXISTS-OTHER-" + uuid.New().String()[:8], Status: POStatusDraft,
		OrderDate: now, TotalAmount: "0", Currency: "EUR", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePO(ctxOwn, otherPO); err != nil {
		t.Fatalf("CreatePO (other): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "purchase_orders", otherPO.ID)

	exists, err := repo.PONumberExists(ctxOwn, tenantOwn, poNumber, nil)
	if err != nil {
		t.Fatalf("PONumberExists: %v", err)
	}
	if !exists {
		t.Fatalf("PONumberExists: expected true for a number that was just created")
	}

	// excludeID pointing at the PO itself: its own number must not block its
	// own update.
	exists, err = repo.PONumberExists(ctxOwn, tenantOwn, poNumber, &po.ID)
	if err != nil {
		t.Fatalf("PONumberExists (exclude self): %v", err)
	}
	if exists {
		t.Fatalf("PONumberExists: excludeID=own ID must exclude the row itself, got exists=true")
	}

	// excludeID pointing at an unrelated PO must not suppress the match.
	exists, err = repo.PONumberExists(ctxOwn, tenantOwn, poNumber, &otherPO.ID)
	if err != nil {
		t.Fatalf("PONumberExists (exclude unrelated): %v", err)
	}
	if !exists {
		t.Fatalf("PONumberExists: excludeID of an unrelated PO must not hide the real match")
	}

	// Same tenantID passed explicitly, but from a foreign-tenant ctx: RLS
	// must block visibility of the row regardless of the explicit parameter.
	exists, err = repo.PONumberExists(ctxOther, tenantOwn, poNumber, nil)
	if err != nil {
		t.Fatalf("PONumberExists (foreign ctx): %v", err)
	}
	if exists {
		t.Fatalf("PONumberExists: a foreign-tenant ctx must not see another tenant's po_number, RLS should have blocked it")
	}

	// A number that was never created must not exist.
	exists, err = repo.PONumberExists(ctxOwn, tenantOwn, "PO-NEVER-CREATED-"+uuid.New().String(), nil)
	if err != nil {
		t.Fatalf("PONumberExists (unknown number): %v", err)
	}
	if exists {
		t.Fatalf("PONumberExists: expected false for a number that was never created")
	}
}

func TestListSuppliers_FilterAndTenantScopedTotal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf ListSuppliers Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf ListSuppliers Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	names := []string{"Alpha Corp", "Beta GmbH", "Gamma AG"}
	ids := make([]uuid.UUID, 0, len(names))
	for _, name := range names {
		s := &Supplier{ID: uuid.New(), TenantID: tenantOwn, Name: name, CreatedAt: now, UpdatedAt: now}
		if err := repo.CreateSupplier(ctxOwn, s); err != nil {
			t.Fatalf("CreateSupplier(%q): %v", name, err)
		}
		ids = append(ids, s.ID)
		defer testutil.CleanupRow(t, pool, "suppliers", s.ID)
	}

	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign := &Supplier{ID: uuid.New(), TenantID: tenantOther, Name: "Alpha Foreign", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSupplier(ctxOther, foreign); err != nil {
		t.Fatalf("CreateSupplier (foreign): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "suppliers", foreign.ID)

	// No filter: total must count only this tenant's three suppliers, never
	// the foreign tenant's "Alpha Foreign".
	list, total, err := repo.ListSuppliers(ctxOwn, tenantOwn, ListSuppliersFilter{}, 0, 50)
	if err != nil {
		t.Fatalf("ListSuppliers: %v", err)
	}
	if total != 3 {
		t.Fatalf("ListSuppliers total: expected 3, got %d", total)
	}
	if len(list) != 3 {
		t.Fatalf("ListSuppliers page: expected 3 rows, got %d", len(list))
	}

	// Search filter: only "Alpha Corp" matches, never the foreign tenant's
	// "Alpha Foreign" even though its name also matches the search term.
	list, total, err = repo.ListSuppliers(ctxOwn, tenantOwn, ListSuppliersFilter{Search: "alpha"}, 0, 50)
	if err != nil {
		t.Fatalf("ListSuppliers (search): %v", err)
	}
	if total != 1 {
		t.Fatalf("ListSuppliers (search) total: expected 1, got %d", total)
	}
	if len(list) != 1 || list[0].Name != "Alpha Corp" {
		t.Fatalf("ListSuppliers (search): expected only 'Alpha Corp', got %+v", list)
	}

	// Soft-deleted suppliers must drop out of both the page and the total.
	if err := repo.DeleteSupplier(ctxOwn, tenantOwn, ids[0]); err != nil {
		t.Fatalf("DeleteSupplier: %v", err)
	}
	list, total, err = repo.ListSuppliers(ctxOwn, tenantOwn, ListSuppliersFilter{}, 0, 50)
	if err != nil {
		t.Fatalf("ListSuppliers (after delete): %v", err)
	}
	if total != 2 {
		t.Fatalf("ListSuppliers (after delete) total: expected 2, got %d", total)
	}
	for _, s := range list {
		if s.ID == ids[0] {
			t.Fatalf("ListSuppliers (after delete): soft-deleted supplier %s still in the page", ids[0])
		}
	}
}

func TestListPOs_FilterCombinationsAndTenantScopedTotal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf ListPOs Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf ListPOs Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	supplierA := &Supplier{ID: uuid.New(), TenantID: tenantOwn, Name: "ListPOs Supplier A", CreatedAt: now, UpdatedAt: now}
	supplierB := &Supplier{ID: uuid.New(), TenantID: tenantOwn, Name: "ListPOs Supplier B", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSupplier(ctxOwn, supplierA); err != nil {
		t.Fatalf("CreateSupplier A: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "suppliers", supplierA.ID)
	if err := repo.CreateSupplier(ctxOwn, supplierB); err != nil {
		t.Fatalf("CreateSupplier B: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "suppliers", supplierB.ID)

	early := now.AddDate(0, 0, -10)
	late := now.AddDate(0, 0, -1)

	poDraftA := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplierA.ID,
		PONumber: "PO-LIST-A1-" + uuid.New().String()[:8], Status: POStatusDraft,
		OrderDate: early, TotalAmount: "0", Currency: "EUR", CreatedAt: now, UpdatedAt: now,
	}
	poSubmittedA := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplierA.ID,
		PONumber: "PO-LIST-A2-" + uuid.New().String()[:8], Status: POStatusSubmitted,
		OrderDate: late, TotalAmount: "0", Currency: "EUR", CreatedAt: now, UpdatedAt: now,
	}
	poDraftB := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplierB.ID,
		PONumber: "PO-LIST-B1-" + uuid.New().String()[:8], Status: POStatusDraft,
		OrderDate: now, TotalAmount: "0", Currency: "EUR", CreatedAt: now, UpdatedAt: now,
	}
	for _, po := range []*PurchaseOrder{poDraftA, poSubmittedA, poDraftB} {
		if err := repo.CreatePO(ctxOwn, po); err != nil {
			t.Fatalf("CreatePO(%s): %v", po.PONumber, err)
		}
		defer testutil.CleanupRow(t, pool, "purchase_orders", po.ID)
	}

	foreignSupplier := &Supplier{ID: uuid.New(), TenantID: tenantOther, Name: "Foreign Supplier", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSupplier(ctxOther, foreignSupplier); err != nil {
		t.Fatalf("CreateSupplier (foreign): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "suppliers", foreignSupplier.ID)
	foreignPO := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantOther, SupplierID: foreignSupplier.ID,
		PONumber: "PO-LIST-FOREIGN-" + uuid.New().String()[:8], Status: POStatusDraft,
		OrderDate: now, TotalAmount: "0", Currency: "EUR", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePO(ctxOther, foreignPO); err != nil {
		t.Fatalf("CreatePO (foreign): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "purchase_orders", foreignPO.ID)

	// No filter: total must be exactly this tenant's three POs, never the
	// foreign tenant's fourth.
	_, total, err := repo.ListPOs(ctxOwn, tenantOwn, ListPOsFilter{}, 0, 50)
	if err != nil {
		t.Fatalf("ListPOs: %v", err)
	}
	if total != 3 {
		t.Fatalf("ListPOs total: expected 3, got %d", total)
	}

	// SupplierID filter.
	list, total, err := repo.ListPOs(ctxOwn, tenantOwn, ListPOsFilter{SupplierID: &supplierA.ID}, 0, 50)
	if err != nil {
		t.Fatalf("ListPOs (supplier filter): %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("ListPOs (supplier filter): expected 2/2, got %d/%d", len(list), total)
	}
	for _, po := range list {
		if po.SupplierID != supplierA.ID {
			t.Fatalf("ListPOs (supplier filter): got a PO for supplier %s, expected only %s", po.SupplierID, supplierA.ID)
		}
	}

	// Status filter.
	submitted := POStatusSubmitted
	list, total, err = repo.ListPOs(ctxOwn, tenantOwn, ListPOsFilter{Status: &submitted}, 0, 50)
	if err != nil {
		t.Fatalf("ListPOs (status filter): %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != poSubmittedA.ID {
		t.Fatalf("ListPOs (status filter): expected exactly poSubmittedA, got %+v (total=%d)", list, total)
	}

	// Date range filter: only the early draft PO falls within [-11d, -9d].
	dateFrom := now.AddDate(0, 0, -11)
	dateTo := now.AddDate(0, 0, -9)
	list, total, err = repo.ListPOs(ctxOwn, tenantOwn, ListPOsFilter{DateFrom: &dateFrom, DateTo: &dateTo}, 0, 50)
	if err != nil {
		t.Fatalf("ListPOs (date filter): %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != poDraftA.ID {
		t.Fatalf("ListPOs (date filter): expected exactly poDraftA, got %+v (total=%d)", list, total)
	}

	// Combined SupplierID + Status filter narrows to a single row.
	list, total, err = repo.ListPOs(ctxOwn, tenantOwn, ListPOsFilter{SupplierID: &supplierA.ID, Status: &submitted}, 0, 50)
	if err != nil {
		t.Fatalf("ListPOs (combined filter): %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != poSubmittedA.ID {
		t.Fatalf("ListPOs (combined filter): expected exactly poSubmittedA, got %+v (total=%d)", list, total)
	}

	// The count query must carry the same tenant condition as the page
	// query: an explicit tenantOwn passed from a foreign-tenant ctx must be
	// blocked by RLS on both, not just on the page.
	list, total, err = repo.ListPOs(ctxOther, tenantOwn, ListPOsFilter{}, 0, 50)
	if err != nil {
		t.Fatalf("ListPOs (foreign ctx): %v", err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("ListPOs (foreign ctx): expected 0/0, got %d/%d -- count and page query disagree on tenant scoping", len(list), total)
	}
}

func TestGetPOWithLines_HeaderAndLinesConsistent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf GetPOWithLines Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	supplier := &Supplier{ID: uuid.New(), TenantID: tenantOwn, Name: "GetPOWithLines Supplier", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSupplier(ctxOwn, supplier); err != nil {
		t.Fatalf("CreateSupplier: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	po := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		PONumber: "PO-WITHLINES-" + uuid.New().String()[:8], Status: POStatusDraft,
		OrderDate: now, TotalAmount: "0", Currency: "EUR", Notes: "with lines",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePO(ctxOwn, po); err != nil {
		t.Fatalf("CreatePO: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "purchase_orders", po.ID)

	// Lines are created out of position order; GetPOWithLines must still
	// return them ordered by line_position ascending.
	lineSecond := &POLine{
		ID: uuid.New(), TenantID: tenantOwn, POID: po.ID, ProductName: "Second Line",
		Quantity: "1", UnitPrice: "1", TaxRate: "0", ReceivedQuantity: "0",
		LinePosition: 2, CreatedAt: now, UpdatedAt: now,
	}
	lineFirst := &POLine{
		ID: uuid.New(), TenantID: tenantOwn, POID: po.ID, ProductName: "First Line",
		Quantity: "1", UnitPrice: "1", TaxRate: "0", ReceivedQuantity: "0",
		LinePosition: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePOLine(ctxOwn, lineSecond); err != nil {
		t.Fatalf("CreatePOLine (second): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "po_lines", lineSecond.ID)
	if err := repo.CreatePOLine(ctxOwn, lineFirst); err != nil {
		t.Fatalf("CreatePOLine (first): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "po_lines", lineFirst.ID)

	got, err := repo.GetPOWithLines(ctxOwn, tenantOwn, po.ID)
	if err != nil {
		t.Fatalf("GetPOWithLines: %v", err)
	}
	if got.PONumber != po.PONumber || got.Notes != "with lines" {
		t.Fatalf("GetPOWithLines: header mismatch, got %+v", got)
	}
	if len(got.Lines) != 2 {
		t.Fatalf("GetPOWithLines: expected 2 lines, got %d", len(got.Lines))
	}
	if got.Lines[0].ProductName != "First Line" || got.Lines[1].ProductName != "Second Line" {
		t.Fatalf("GetPOWithLines: lines not ordered by line_position, got [%s, %s]",
			got.Lines[0].ProductName, got.Lines[1].ProductName)
	}

	// A PO with no lines must come back with an empty (len 0), usable Lines
	// slice, not fail or panic on the caller ranging over it.
	emptyPO := &PurchaseOrder{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		PONumber: "PO-NOLINES-" + uuid.New().String()[:8], Status: POStatusDraft,
		OrderDate: now, TotalAmount: "0", Currency: "EUR", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePO(ctxOwn, emptyPO); err != nil {
		t.Fatalf("CreatePO (empty): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "purchase_orders", emptyPO.ID)

	gotEmpty, err := repo.GetPOWithLines(ctxOwn, tenantOwn, emptyPO.ID)
	if err != nil {
		t.Fatalf("GetPOWithLines (no lines): %v", err)
	}
	if len(gotEmpty.Lines) != 0 {
		t.Fatalf("GetPOWithLines (no lines): expected 0 lines, got %d", len(gotEmpty.Lines))
	}

	// An unknown PO ID must surface ErrPONotFound, not a generic error.
	if _, err := repo.GetPOWithLines(ctxOwn, tenantOwn, uuid.New()); err != ErrPONotFound {
		t.Fatalf("GetPOWithLines (unknown ID): expected ErrPONotFound, got %v", err)
	}
}
