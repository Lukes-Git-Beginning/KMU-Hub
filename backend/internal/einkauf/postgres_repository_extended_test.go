package einkauf

// postgres_repository_extended.go (CatalogItems, SupplierRatings,
// FrameworkContracts/-Items/-Calls) had no DB-backed test coverage --
// tenant_write_test.go only exercises the base Repository (Supplier/PO).
// Every write below already carries an explicit tenant_id predicate; this
// file proves it against a real database and RLS, following the same
// foreign-ctx-then-own-ctx shape as tenant_write_test.go.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func newEinkaufExtendedTestSupplier(t *testing.T, repo *PostgresRepository, ctx context.Context, tenantID uuid.UUID, name string) *Supplier {
	t.Helper()
	now := time.Now().UTC()
	s := &Supplier{
		ID: uuid.New(), TenantID: tenantID, Name: name,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSupplier(ctx, s); err != nil {
		t.Fatalf("CreateSupplier: %v", err)
	}
	return s
}

func TestCatalogItemCRUD_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf CatalogItem Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf CatalogItem Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctxOwn, tenantOwn, "Catalog Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	item := &CatalogItem{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		Name: "Screwdriver Set", SKU: "SKU-1", Category: "tools",
		Price: "19.9900", Currency: "EUR", Unit: "Stk", Available: true,
		MinOrderQty: "1.0000", CreatedAt: now, UpdatedAt: now,
	}

	// A foreign ctx must not be able to insert a row for another tenant's
	// TenantID -- WITH CHECK on the RLS policy has to reject it.
	if err := repo.CreateCatalogItem(ctxOther, item); err == nil {
		t.Fatalf("CreateCatalogItem (foreign ctx, foreign TenantID field): expected an error, got nil")
	}
	if err := repo.CreateCatalogItem(ctxOwn, item); err != nil {
		t.Fatalf("CreateCatalogItem (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "supplier_catalog_items", item.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "supplier_catalog_items", item.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "supplier_catalog_items", item.ID, 0)

	got, err := repo.GetCatalogItem(ctxOwn, tenantOwn, item.ID)
	if err != nil {
		t.Fatalf("GetCatalogItem (own ctx): %v", err)
	}
	if got.Name != "Screwdriver Set" || got.Price != "19.9900" {
		t.Fatalf("GetCatalogItem returned unexpected data: %+v", got)
	}
	if _, err := repo.GetCatalogItem(ctxOther, tenantOwn, item.ID); err != ErrCatalogItemNotFound {
		t.Fatalf("GetCatalogItem (foreign ctx): expected ErrCatalogItemNotFound, got %v", err)
	}

	// UpdateCatalogItem: foreign ctx targets the real tenantOwn item via its
	// explicit WHERE predicate -- only RLS row-visibility can stop it.
	foreign := *item
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateCatalogItem(ctxOther, &foreign); err != ErrCatalogItemNotFound {
		t.Fatalf("UpdateCatalogItem (foreign ctx): expected ErrCatalogItemNotFound, got %v", err)
	}
	got, _ = repo.GetCatalogItem(sysCtx, tenantOwn, item.ID)
	if got.Name == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the catalog item")
	}
	foreign.Name = "Renamed"
	if err := repo.UpdateCatalogItem(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateCatalogItem (own ctx): %v", err)
	}
	got, _ = repo.GetCatalogItem(ctxOwn, tenantOwn, item.ID)
	if got.Name != "Renamed" {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	// ListCatalogItems: filter by SupplierID/Category/Search/Available, tenant-scoped.
	items, total, err := repo.ListCatalogItems(ctxOwn, tenantOwn, ListCatalogItemsFilter{
		SupplierID: &supplier.ID, Category: "tools", Search: "renamed",
	}, 0, 10)
	if err != nil {
		t.Fatalf("ListCatalogItems: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("ListCatalogItems: expected 1 item, got total=%d len=%d", total, len(items))
	}
	avail := false
	items, total, err = repo.ListCatalogItems(ctxOwn, tenantOwn, ListCatalogItemsFilter{Available: &avail}, 0, 10)
	if err != nil {
		t.Fatalf("ListCatalogItems (available=false): %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("ListCatalogItems (available=false): expected 0 items, got total=%d len=%d", total, len(items))
	}
	items, total, err = repo.ListCatalogItems(ctxOther, tenantOther, ListCatalogItemsFilter{}, 0, 10)
	if err != nil {
		t.Fatalf("ListCatalogItems (other tenant): %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("ListCatalogItems (other tenant): expected 0 items, got total=%d len=%d", total, len(items))
	}

	// DeleteCatalogItem: foreign ctx must not remove the row, own ctx must.
	if err := repo.DeleteCatalogItem(ctxOther, tenantOwn, item.ID); err != ErrCatalogItemNotFound {
		t.Fatalf("DeleteCatalogItem (foreign ctx): expected ErrCatalogItemNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "supplier_catalog_items", item.ID, 1)
	if err := repo.DeleteCatalogItem(ctxOwn, tenantOwn, item.ID); err != nil {
		t.Fatalf("DeleteCatalogItem (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "supplier_catalog_items", item.ID, 0)
}

func TestSupplierRatingCRUD_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf Rating Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf Rating Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctxOwn, tenantOwn, "Rating Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	rating := &SupplierRating{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		Category: RatingCategoryQuality, Rating: 4, Comment: "Good",
		RatedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSupplierRating(ctxOwn, rating); err != nil {
		t.Fatalf("CreateSupplierRating: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "supplier_ratings", rating.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "supplier_ratings", rating.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "supplier_ratings", rating.ID, 0)

	// Duplicate rating for the same tenant/supplier/category must be rejected.
	dup := &SupplierRating{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		Category: RatingCategoryQuality, Rating: 5, Comment: "Also good",
		RatedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSupplierRating(ctxOwn, dup); err != ErrDuplicateRating {
		t.Fatalf("CreateSupplierRating (duplicate): expected ErrDuplicateRating, got %v", err)
	}

	// The same category for a DIFFERENT tenant must not collide -- the
	// duplicate check itself is tenant-scoped.
	otherSupplier := newEinkaufExtendedTestSupplier(t, repo, ctxOther, tenantOther, "Other Rating Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", otherSupplier.ID)
	otherRating := &SupplierRating{
		ID: uuid.New(), TenantID: tenantOther, SupplierID: otherSupplier.ID,
		Category: RatingCategoryQuality, Rating: 3, Comment: "Different tenant",
		RatedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSupplierRating(ctxOther, otherRating); err != nil {
		t.Fatalf("CreateSupplierRating (other tenant, same category): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "supplier_ratings", otherRating.ID)

	got, err := repo.GetSupplierRating(ctxOwn, tenantOwn, rating.ID)
	if err != nil {
		t.Fatalf("GetSupplierRating (own ctx): %v", err)
	}
	if got.Rating != 4 || got.Comment != "Good" {
		t.Fatalf("GetSupplierRating returned unexpected data: %+v", got)
	}
	if _, err := repo.GetSupplierRating(ctxOther, tenantOwn, rating.ID); err != ErrSupplierRatingNotFound {
		t.Fatalf("GetSupplierRating (foreign ctx): expected ErrSupplierRatingNotFound, got %v", err)
	}

	ratings, err := repo.ListSupplierRatings(ctxOwn, tenantOwn, supplier.ID)
	if err != nil {
		t.Fatalf("ListSupplierRatings (own ctx): %v", err)
	}
	if len(ratings) != 1 {
		t.Fatalf("ListSupplierRatings (own ctx): expected 1 rating, got %d", len(ratings))
	}
	ratings, err = repo.ListSupplierRatings(ctxOther, tenantOwn, supplier.ID)
	if err != nil {
		t.Fatalf("ListSupplierRatings (foreign ctx): %v", err)
	}
	if len(ratings) != 0 {
		t.Fatalf("ListSupplierRatings (foreign ctx): expected 0 ratings, got %d", len(ratings))
	}

	// DeleteSupplierRating: foreign ctx must not remove the row, own ctx must.
	if err := repo.DeleteSupplierRating(ctxOther, tenantOwn, rating.ID); err != ErrSupplierRatingNotFound {
		t.Fatalf("DeleteSupplierRating (foreign ctx): expected ErrSupplierRatingNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "supplier_ratings", rating.ID, 1)
	if err := repo.DeleteSupplierRating(ctxOwn, tenantOwn, rating.ID); err != nil {
		t.Fatalf("DeleteSupplierRating (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "supplier_ratings", rating.ID, 0)
}

func TestFrameworkContractCRUD_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf Contract Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf Contract Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctxOwn, tenantOwn, "Contract Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	fc := &FrameworkContract{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		Title: "Yearly Frame Contract", ContractNr: "FC-" + uuid.New().String()[:8],
		TotalValue: "5000.0000", UsedValue: "0.0000", Currency: "EUR",
		Status: ContractStatusDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateFrameworkContract(ctxOwn, fc); err != nil {
		t.Fatalf("CreateFrameworkContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contracts", fc.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "framework_contracts", fc.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "framework_contracts", fc.ID, 0)

	got, err := repo.GetFrameworkContract(ctxOwn, tenantOwn, fc.ID)
	if err != nil {
		t.Fatalf("GetFrameworkContract (own ctx): %v", err)
	}
	if got.Title != "Yearly Frame Contract" {
		t.Fatalf("GetFrameworkContract returned unexpected data: %+v", got)
	}
	if _, err := repo.GetFrameworkContract(ctxOther, tenantOwn, fc.ID); err != ErrContractNotFound {
		t.Fatalf("GetFrameworkContract (foreign ctx): expected ErrContractNotFound, got %v", err)
	}

	withItems, err := repo.GetFrameworkContractWithItems(ctxOwn, tenantOwn, fc.ID)
	if err != nil {
		t.Fatalf("GetFrameworkContractWithItems: %v", err)
	}
	if len(withItems.Items) != 0 {
		t.Fatalf("GetFrameworkContractWithItems: expected no items yet, got %d", len(withItems.Items))
	}

	// UpdateFrameworkContract: foreign ctx targets the real row explicitly.
	foreign := *fc
	foreign.Title = "Hacked"
	foreign.Status = ContractStatusActive
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateFrameworkContract(ctxOther, &foreign); err != ErrContractNotFound {
		t.Fatalf("UpdateFrameworkContract (foreign ctx): expected ErrContractNotFound, got %v", err)
	}
	got, _ = repo.GetFrameworkContract(sysCtx, tenantOwn, fc.ID)
	if got.Title == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the framework contract")
	}
	foreign.Title = "Renewed Frame Contract"
	if err := repo.UpdateFrameworkContract(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateFrameworkContract (own ctx): %v", err)
	}
	got, _ = repo.GetFrameworkContract(ctxOwn, tenantOwn, fc.ID)
	if got.Title != "Renewed Frame Contract" || got.Status != ContractStatusActive {
		t.Fatalf("own-tenant write did not land: %+v", got)
	}

	contracts, total, err := repo.ListFrameworkContracts(ctxOwn, tenantOwn, ListContractsFilter{SupplierID: &supplier.ID}, 0, 10)
	if err != nil {
		t.Fatalf("ListFrameworkContracts (own ctx): %v", err)
	}
	if total != 1 || len(contracts) != 1 {
		t.Fatalf("ListFrameworkContracts (own ctx): expected 1 contract, got total=%d len=%d", total, len(contracts))
	}
	contracts, total, err = repo.ListFrameworkContracts(ctxOther, tenantOther, ListContractsFilter{}, 0, 10)
	if err != nil {
		t.Fatalf("ListFrameworkContracts (other tenant): %v", err)
	}
	if total != 0 || len(contracts) != 0 {
		t.Fatalf("ListFrameworkContracts (other tenant): expected 0 contracts, got total=%d len=%d", total, len(contracts))
	}

	// DeleteFrameworkContract: foreign ctx must not remove the row, own ctx must.
	if err := repo.DeleteFrameworkContract(ctxOther, tenantOwn, fc.ID); err != ErrContractNotFound {
		t.Fatalf("DeleteFrameworkContract (foreign ctx): expected ErrContractNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "framework_contracts", fc.ID, 1)
	if err := repo.DeleteFrameworkContract(ctxOwn, tenantOwn, fc.ID); err != nil {
		t.Fatalf("DeleteFrameworkContract (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "framework_contracts", fc.ID, 0)
}

func TestFrameworkContract_ContractNrExists_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf ContractNr Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf ContractNr Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	supplierOwn := newEinkaufExtendedTestSupplier(t, repo, ctxOwn, tenantOwn, "ContractNr Supplier Own")
	defer testutil.CleanupRow(t, pool, "suppliers", supplierOwn.ID)
	supplierOther := newEinkaufExtendedTestSupplier(t, repo, ctxOther, tenantOther, "ContractNr Supplier Other")
	defer testutil.CleanupRow(t, pool, "suppliers", supplierOther.ID)

	contractNr := "FC-SHARED-" + uuid.New().String()[:8]

	fcOwn := &FrameworkContract{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplierOwn.ID,
		Title: "Own Contract", ContractNr: contractNr,
		TotalValue: "1000.0000", UsedValue: "0.0000", Currency: "EUR",
		Status: ContractStatusDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateFrameworkContract(ctxOwn, fcOwn); err != nil {
		t.Fatalf("CreateFrameworkContract (own): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contracts", fcOwn.ID)

	// Before the other tenant has any contract with this number, ContractNrExists
	// must be false FOR THE OTHER TENANT, even though tenantOwn already has it --
	// the check itself is tenant-scoped, not global.
	exists, err := repo.ContractNrExists(ctxOther, tenantOther, contractNr, nil)
	if err != nil {
		t.Fatalf("ContractNrExists (other tenant, before insert): %v", err)
	}
	if exists {
		t.Fatalf("ContractNrExists (other tenant): a same-numbered contract in a DIFFERENT tenant reported as existing")
	}
	exists, err = repo.ContractNrExists(ctxOwn, tenantOwn, contractNr, nil)
	if err != nil {
		t.Fatalf("ContractNrExists (own tenant): %v", err)
	}
	if !exists {
		t.Fatalf("ContractNrExists (own tenant): expected true, got false")
	}

	// The SAME contract_nr can be reused by the other tenant without collision.
	fcOther := &FrameworkContract{
		ID: uuid.New(), TenantID: tenantOther, SupplierID: supplierOther.ID,
		Title: "Other Tenant Contract", ContractNr: contractNr,
		TotalValue: "1000.0000", UsedValue: "0.0000", Currency: "EUR",
		Status: ContractStatusDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateFrameworkContract(ctxOther, fcOther); err != nil {
		t.Fatalf("CreateFrameworkContract (other tenant, same contract_nr): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contracts", fcOther.ID)

	// excludeID: fcOwn's own contract_nr must not exist "excluding itself".
	exists, err = repo.ContractNrExists(ctxOwn, tenantOwn, contractNr, &fcOwn.ID)
	if err != nil {
		t.Fatalf("ContractNrExists (exclude self): %v", err)
	}
	if exists {
		t.Fatalf("ContractNrExists (exclude self): expected false, got true")
	}
}

func TestFrameworkContract_CreateContractCall_AccumulatesUsedValue(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf UsedValue Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctxOwn, tenantOwn, "UsedValue Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	fc := &FrameworkContract{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		Title: "Accumulation Contract", ContractNr: "FC-ACC-" + uuid.New().String()[:8],
		TotalValue: "5000.0000", UsedValue: "0.0000", Currency: "EUR",
		Status: ContractStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateFrameworkContract(ctxOwn, fc); err != nil {
		t.Fatalf("CreateFrameworkContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contracts", fc.ID)

	call1 := &FrameworkContractCall{
		ID: uuid.New(), TenantID: tenantOwn, ContractID: fc.ID,
		Amount: "1200.5000", Currency: "EUR", CalledAt: now, Notes: "First call",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContractCall(ctxOwn, call1); err != nil {
		t.Fatalf("CreateContractCall (1): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contract_calls", call1.ID)

	// used_value is written by CreateContractCall's own transaction — there is
	// no second statement that could be lost and let the cap drift open.
	got, err := repo.GetFrameworkContract(ctxOwn, tenantOwn, fc.ID)
	if err != nil {
		t.Fatalf("GetFrameworkContract (after call 1): %v", err)
	}
	if got.UsedValue != "1200.5000" {
		t.Fatalf("used_value (after call 1): expected 1200.5000, got %q", got.UsedValue)
	}

	// A second call must ADD to the running total, not overwrite it.
	call2 := &FrameworkContractCall{
		ID: uuid.New(), TenantID: tenantOwn, ContractID: fc.ID,
		Amount: "300.2500", Currency: "EUR", CalledAt: now, Notes: "Second call",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContractCall(ctxOwn, call2); err != nil {
		t.Fatalf("CreateContractCall (2): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contract_calls", call2.ID)

	got, err = repo.GetFrameworkContract(ctxOwn, tenantOwn, fc.ID)
	if err != nil {
		t.Fatalf("GetFrameworkContract (after call 2): %v", err)
	}
	if got.UsedValue != "1500.7500" {
		t.Fatalf("used_value (after call 2): expected 1500.7500 (sum, not overwrite), got %q", got.UsedValue)
	}

	calls, err := repo.ListContractCalls(ctxOwn, tenantOwn, fc.ID)
	if err != nil {
		t.Fatalf("ListContractCalls: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("ListContractCalls: expected 2 calls, got %d", len(calls))
	}
}

// The cap lives in SQL (numeric comparison against a freshly summed
// SUM(amount) under a FOR UPDATE lock), so it has to be proven against a real
// database — the in-memory double in service_extended_test.go can only show
// that the service surfaces the rejection, not that the statement computes it.
func TestFrameworkContract_CreateContractCall_EnforcesRemainingValue(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf Contract Cap Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctxOwn, tenantOwn, "Cap Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	fc := &FrameworkContract{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		Title: "Capped Contract", ContractNr: "FC-CAP-" + uuid.New().String()[:8],
		TotalValue: "1000.0000", UsedValue: "0.0000", Currency: "EUR",
		Status: ContractStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateFrameworkContract(ctxOwn, fc); err != nil {
		t.Fatalf("CreateFrameworkContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contracts", fc.ID)

	newCall := func(amount string) *FrameworkContractCall {
		return &FrameworkContractCall{
			ID: uuid.New(), TenantID: tenantOwn, ContractID: fc.ID,
			Amount: amount, Currency: "EUR", CalledAt: now,
			CreatedAt: now, UpdatedAt: now,
		}
	}

	first := newCall("600.0000")
	if err := repo.CreateContractCall(ctxOwn, first); err != nil {
		t.Fatalf("CreateContractCall (600 of 1000): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contract_calls", first.ID)

	// 500 of a remaining 400 must be refused, with the remainder in the message.
	err := repo.CreateContractCall(ctxOwn, newCall("500.0000"))
	if !errors.Is(err, ErrContractBudgetExceeded) {
		t.Fatalf("CreateContractCall (500 of remaining 400): expected ErrContractBudgetExceeded, got %v", err)
	}
	if !strings.Contains(err.Error(), "400.0000") {
		t.Fatalf("CreateContractCall (over cap): message must name the remaining value, got %q", err.Error())
	}

	// Exactly the remaining 400 still fits.
	exact := newCall("400.0000")
	if err := repo.CreateContractCall(ctxOwn, exact); err != nil {
		t.Fatalf("CreateContractCall (exactly the remaining 400): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contract_calls", exact.ID)

	// A tenth of a cent beyond the exhausted contract does not.
	if err := repo.CreateContractCall(ctxOwn, newCall("0.0001")); !errors.Is(err, ErrContractBudgetExceeded) {
		t.Fatalf("CreateContractCall (0.0001 on an exhausted contract): expected ErrContractBudgetExceeded, got %v", err)
	}

	// Neither rejection may have written a row, and used_value must equal the
	// sum of the two accepted calls.
	calls, err := repo.ListContractCalls(ctxOwn, tenantOwn, fc.ID)
	if err != nil {
		t.Fatalf("ListContractCalls: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("ListContractCalls: expected 2 persisted calls, got %d", len(calls))
	}
	got, err := repo.GetFrameworkContract(ctxOwn, tenantOwn, fc.ID)
	if err != nil {
		t.Fatalf("GetFrameworkContract: %v", err)
	}
	if got.UsedValue != "1000.0000" {
		t.Fatalf("used_value: expected 1000.0000, got %q", got.UsedValue)
	}
}

func TestFrameworkContract_CreateContractCall_RejectsInactiveAndUnknown(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf Contract Status Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctxOwn, tenantOwn, "Status Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	for _, contractStatus := range []ContractStatus{ContractStatusDraft, ContractStatusExpired} {
		fc := &FrameworkContract{
			ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
			Title: "Inactive Contract", ContractNr: "FC-ST-" + uuid.New().String()[:8],
			TotalValue: "1000.0000", UsedValue: "0.0000", Currency: "EUR",
			Status: contractStatus, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateFrameworkContract(ctxOwn, fc); err != nil {
			t.Fatalf("CreateFrameworkContract (%s): %v", contractStatus, err)
		}
		defer testutil.CleanupRow(t, pool, "framework_contracts", fc.ID)

		err := repo.CreateContractCall(ctxOwn, &FrameworkContractCall{
			ID: uuid.New(), TenantID: tenantOwn, ContractID: fc.ID,
			Amount: "10.0000", Currency: "EUR", CalledAt: now,
			CreatedAt: now, UpdatedAt: now,
		})
		if !errors.Is(err, ErrContractNotActive) {
			t.Fatalf("CreateContractCall on a %s contract: expected ErrContractNotActive, got %v", contractStatus, err)
		}
		if !strings.Contains(err.Error(), string(contractStatus)) {
			t.Fatalf("CreateContractCall on a %s contract: message must name the status, got %q", contractStatus, err.Error())
		}
	}

	// An unknown contract id must not fall through to the INSERT — a
	// call-off without a contract would be an orphan the cap can never see.
	err := repo.CreateContractCall(ctxOwn, &FrameworkContractCall{
		ID: uuid.New(), TenantID: tenantOwn, ContractID: uuid.New(),
		Amount: "10.0000", Currency: "EUR", CalledAt: now,
		CreatedAt: now, UpdatedAt: now,
	})
	if !errors.Is(err, ErrContractNotFound) {
		t.Fatalf("CreateContractCall on an unknown contract: expected ErrContractNotFound, got %v", err)
	}
}

func TestFrameworkContractItems_CRUD_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Einkauf ContractItem Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Einkauf ContractItem Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	supplier := newEinkaufExtendedTestSupplier(t, repo, ctxOwn, tenantOwn, "ContractItem Supplier")
	defer testutil.CleanupRow(t, pool, "suppliers", supplier.ID)

	fc := &FrameworkContract{
		ID: uuid.New(), TenantID: tenantOwn, SupplierID: supplier.ID,
		Title: "Item Contract", ContractNr: "FC-ITM-" + uuid.New().String()[:8],
		TotalValue: "2000.0000", UsedValue: "0.0000", Currency: "EUR",
		Status: ContractStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateFrameworkContract(ctxOwn, fc); err != nil {
		t.Fatalf("CreateFrameworkContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contracts", fc.ID)

	item := &FrameworkContractItem{
		ID: uuid.New(), TenantID: tenantOwn, ContractID: fc.ID,
		Name: "Bolt M8", UnitPrice: "0.2500", Unit: "Stk",
		AgreedQty: "1000.0000", CalledQty: "0.0000",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContractItem(ctxOwn, item); err != nil {
		t.Fatalf("CreateContractItem: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "framework_contract_items", item.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "framework_contract_items", item.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "framework_contract_items", item.ID, 0)

	got, err := repo.QueryRowContractItem(ctxOwn, tenantOwn, item.ID)
	if err != nil {
		t.Fatalf("QueryRowContractItem (own ctx): %v", err)
	}
	if got.Name != "Bolt M8" {
		t.Fatalf("QueryRowContractItem returned unexpected data: %+v", got)
	}
	if _, err := repo.QueryRowContractItem(ctxOther, tenantOwn, item.ID); err != ErrContractItemNotFound {
		t.Fatalf("QueryRowContractItem (foreign ctx): expected ErrContractItemNotFound, got %v", err)
	}

	// UpdateContractItem: foreign ctx targets the real row explicitly.
	foreign := *item
	foreign.Name = "Hacked Bolt"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateContractItem(ctxOther, &foreign); err != ErrContractItemNotFound {
		t.Fatalf("UpdateContractItem (foreign ctx): expected ErrContractItemNotFound, got %v", err)
	}
	got, _ = repo.QueryRowContractItem(sysCtx, tenantOwn, item.ID)
	if got.Name == "Hacked Bolt" {
		t.Fatalf("a foreign-tenant write reached the contract item")
	}
	foreign.Name = "Bolt M8 Renamed"
	if err := repo.UpdateContractItem(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateContractItem (own ctx): %v", err)
	}
	got, _ = repo.QueryRowContractItem(ctxOwn, tenantOwn, item.ID)
	if got.Name != "Bolt M8 Renamed" {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	items, err := repo.ListContractItems(ctxOwn, tenantOwn, fc.ID)
	if err != nil {
		t.Fatalf("ListContractItems (own ctx): %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ListContractItems (own ctx): expected 1 item, got %d", len(items))
	}
	items, err = repo.ListContractItems(ctxOther, tenantOwn, fc.ID)
	if err != nil {
		t.Fatalf("ListContractItems (foreign ctx): %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("ListContractItems (foreign ctx): expected 0 items, got %d", len(items))
	}

	// DeleteContractItem: foreign ctx must not remove the row, own ctx must.
	if err := repo.DeleteContractItem(ctxOther, tenantOwn, item.ID); err != ErrContractItemNotFound {
		t.Fatalf("DeleteContractItem (foreign ctx): expected ErrContractItemNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "framework_contract_items", item.ID, 1)
	if err := repo.DeleteContractItem(ctxOwn, tenantOwn, item.ID); err != nil {
		t.Fatalf("DeleteContractItem (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "framework_contract_items", item.ID, 0)
}
