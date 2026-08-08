package employee

// postgres_repository_db_test.go covers the read paths of PostgresEmployeeRepo,
// PostgresDocCategoryRepo and PostgresEmployeeDocRepo against the real
// migration schema (testutil.SkipIfNoDB — the pattern the rest of this
// package's offboard_db_test.go already uses): the CONCAT_WS name resolution
// with its email fallback, and tenant scoping on every read path.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// employeeDocCtx returns a context carrying tenantID plus the hr_admin role.
// hr_employee_documents' hr_document_access policy (migration 000127) gates
// both reads and writes on current_user_has_hr_role — a bare tenant context
// is not enough, unlike the other tables in this package. Every production
// caller of UploadEmployeeDocument already carries admin/hr_admin on their
// JWT (RequirePermission gates the RPC before it runs), so this mirrors a
// real caller rather than reaching for the system-context bypass SeedRow uses.
func employeeDocCtx(tenantID uuid.UUID) context.Context {
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	return context.WithValue(ctx, middleware.UserRolesKey, []string{"hr_admin"})
}

// seedNamedUser inserts a users row with an explicit first/last name — empty
// strings allowed, which is what the email-fallback tests need. users.first_name
// and .last_name are NOT NULL DEFAULT ” (migration 000001), never NULL, so
// this is what every real row looks like, not a synthetic edge case.
func seedNamedUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, firstName, lastName, email string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "email": email,
		"password_hash": "hash", "first_name": firstName, "last_name": lastName,
		"is_active": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}

// ============================================================================
// Name resolution — CONCAT_WS + email fallback (the display_name incident)
// ============================================================================

func TestRepository_GetByID_NamedUserResolvesFullName(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	repo := NewPostgresEmployeeRepo(pool)

	userID := seedNamedUser(t, pool, tenantID, "Anna", "Müller", "anna-"+uuid.NewString()+"@example.test")
	profileID := seedOffboardProfile(t, pool, tenantID, userID, nil)

	got, err := repo.GetByID(ctx, profileID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.UserName != "Anna Müller" {
		t.Errorf("UserName: got %q, want %q", got.UserName, "Anna Müller")
	}
}

// TestRepository_GetByID_BlankNameFallsBackToEmail is the regression this unit
// exists for. CONCAT_WS(' ', a, b) only skips NULL arguments, not empty ones —
// with two empty strings it still applies the separator and returns a single
// space, which NULLIF(x, ”) does not recognize as blank. Every real user with
// no name set (first_name/last_name default to ”, never NULL) therefore fell
// through to a one-space display name instead of the email fallback. This
// fails against the pre-fix query (COALESCE(NULLIF(CONCAT_WS(...), ”), ...))
// and passes only with a TRIM() around the CONCAT_WS before the NULLIF.
func TestRepository_GetByID_BlankNameFallsBackToEmail(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	repo := NewPostgresEmployeeRepo(pool)

	email := "blank-" + uuid.NewString() + "@example.test"
	userID := seedNamedUser(t, pool, tenantID, "", "", email)
	profileID := seedOffboardProfile(t, pool, tenantID, userID, nil)

	got, err := repo.GetByID(ctx, profileID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.UserName != email {
		t.Errorf("UserName: got %q, want the email fallback %q", got.UserName, email)
	}
}

// TestRepository_GetByUserID_BlankNameFallsBackToEmail is the same regression
// through the second read path: GetByUserID carries its own copy of the SQL
// fragment, so a fix to GetByID alone would not fix this one.
func TestRepository_GetByUserID_BlankNameFallsBackToEmail(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	repo := NewPostgresEmployeeRepo(pool)

	email := "blank-" + uuid.NewString() + "@example.test"
	userID := seedNamedUser(t, pool, tenantID, "", "", email)
	seedOffboardProfile(t, pool, tenantID, userID, nil)

	got, err := repo.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if got.UserName != email {
		t.Errorf("UserName: got %q, want the email fallback %q", got.UserName, email)
	}
}

// TestRepository_List_BlankNameFallsBackToEmail is the third copy of the
// fragment, in the paginated list query.
func TestRepository_List_BlankNameFallsBackToEmail(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	repo := NewPostgresEmployeeRepo(pool)

	email := "blank-" + uuid.NewString() + "@example.test"
	userID := seedNamedUser(t, pool, tenantID, "", "", email)
	seedOffboardProfile(t, pool, tenantID, userID, nil)

	employees, _, err := repo.List(ctx, EmployeeFilter{TenantID: tenantID})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, e := range employees {
		if e.UserID == userID {
			found = true
			if e.UserName != email {
				t.Errorf("UserName: got %q, want the email fallback %q", e.UserName, email)
			}
		}
	}
	if !found {
		t.Fatalf("seeded employee %s not present in List result", userID)
	}
}

// TestRepository_GetByID_ManagerNameResolvesWithSameFallback covers the
// manager join, which carries its own copy of the same CONCAT_WS/email
// expression as the employee's own name.
func TestRepository_GetByID_ManagerNameResolvesWithSameFallback(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	repo := NewPostgresEmployeeRepo(pool)

	mgrEmail := "mgr-" + uuid.NewString() + "@example.test"
	mgrID := seedNamedUser(t, pool, tenantID, "", "", mgrEmail)
	seedOffboardProfile(t, pool, tenantID, mgrID, nil)

	empID := seedNamedUser(t, pool, tenantID, "Report", "Ee", "report-"+uuid.NewString()+"@example.test")
	profileID := seedOffboardProfile(t, pool, tenantID, empID, &mgrID)

	got, err := repo.GetByID(ctx, profileID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ManagerName != mgrEmail {
		t.Errorf("ManagerName: got %q, want the email fallback %q", got.ManagerName, mgrEmail)
	}
}

// ============================================================================
// Cross-tenant on every read path
// ============================================================================

func TestRepository_GetByID_CrossTenantIsInvisible(t *testing.T) {
	pool, tenantA, _ := newOffboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Employee DB Test Tenant B")
	repo := NewPostgresEmployeeRepo(pool)

	userID := seedNamedUser(t, pool, tenantA, "Secret", "Owner", "secret-"+uuid.NewString()+"@example.test")
	profileID := seedOffboardProfile(t, pool, tenantA, userID, nil)

	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	if _, err := repo.GetByID(ctxB, profileID); !errors.Is(err, ErrEmployeeNotFound) {
		t.Fatalf("GetByID cross-tenant: got %v, want ErrEmployeeNotFound", err)
	}
}

func TestRepository_GetByUserID_CrossTenantIsInvisible(t *testing.T) {
	pool, tenantA, _ := newOffboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Employee DB Test Tenant B")
	repo := NewPostgresEmployeeRepo(pool)

	userID := seedNamedUser(t, pool, tenantA, "Secret", "Owner", "secret2-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantA, userID, nil)

	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	if _, err := repo.GetByUserID(ctxB, userID); !errors.Is(err, ErrEmployeeNotFound) {
		t.Fatalf("GetByUserID cross-tenant: got %v, want ErrEmployeeNotFound", err)
	}
}

func TestRepository_List_CrossTenantExcludesForeignRows(t *testing.T) {
	pool, tenantA, _ := newOffboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Employee DB Test Tenant B")
	repo := NewPostgresEmployeeRepo(pool)

	userA := seedNamedUser(t, pool, tenantA, "Alpha", "A", "alpha-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantA, userA, nil)

	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	employeesB, totalB, err := repo.List(ctxB, EmployeeFilter{TenantID: tenantB})
	if err != nil {
		t.Fatalf("List B: %v", err)
	}
	if totalB != 0 {
		t.Errorf("List B total: got %d, want 0", totalB)
	}
	for _, e := range employeesB {
		if e.UserID == userA {
			t.Fatalf("cross-tenant leak: tenant B sees tenant A's employee %s", userA)
		}
	}
}

// ============================================================================
// Update
// ============================================================================

func TestRepository_Update_PersistsFields(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	repo := NewPostgresEmployeeRepo(pool)

	userID := seedNamedUser(t, pool, tenantID, "Update", "Ee", "update-"+uuid.NewString()+"@example.test")
	profileID := seedOffboardProfile(t, pool, tenantID, userID, nil)

	got, err := repo.GetByID(ctx, profileID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	got.Department = "Finance"
	got.PositionTitle = "Controller"
	got.IsMinor = true

	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	reread, err := repo.GetByID(ctx, profileID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if reread.Department != "Finance" || reread.PositionTitle != "Controller" || !reread.IsMinor {
		t.Errorf("update did not persist: department=%q position=%q is_minor=%v",
			reread.Department, reread.PositionTitle, reread.IsMinor)
	}
}

// ============================================================================
// Document categories
// ============================================================================

// TestRepository_DocCategory_TenantOwnRowWinsOverSystemSeedOnGetByKey pins
// GetByKey's ORDER BY (tenant_id = $2) DESC tie-break: a tenant that defines
// its own category under a key already used by a system seed (migration
// 000046 ships "arbeitsvertrag") must see its own row, not the seed's.
func TestRepository_DocCategory_TenantOwnRowWinsOverSystemSeedOnGetByKey(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	repo := NewPostgresDocCategoryRepo(pool)

	tenantCatID := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "key": "arbeitsvertrag",
		"name": "Eigener Arbeitsvertrag", "visibility": "manager", "is_system": false, "sort_order": 1,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_document_categories", tenantCatID) })

	got, err := repo.GetByKey(ctx, tenantID, "arbeitsvertrag")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got.ID != tenantCatID {
		t.Errorf("GetByKey: got id %s (system seed), want the tenant's own row %s", got.ID, tenantCatID)
	}
}

func TestRepository_DocCategory_CrossTenantNotFoundOrListed(t *testing.T) {
	pool, tenantA, ctxA := newOffboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Employee DB Test Tenant B")
	repo := NewPostgresDocCategoryRepo(pool)

	catID := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"id": uuid.New(), "tenant_id": tenantA, "key": "geheim-" + uuid.NewString()[:8],
		"name": "Geheim", "visibility": "hr_only", "is_system": false, "sort_order": 90,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_document_categories", catID) })

	// Sanity: the owning tenant sees it.
	if _, err := repo.GetByID(ctxA, tenantA, catID); err != nil {
		t.Fatalf("GetByID same tenant: %v", err)
	}

	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	if _, err := repo.GetByID(ctxB, tenantB, catID); !errors.Is(err, ErrDocumentCategoryNotFound) {
		t.Fatalf("GetByID cross-tenant: got %v, want ErrDocumentCategoryNotFound", err)
	}

	cats, err := repo.ListByTenant(ctxB, tenantB)
	if err != nil {
		t.Fatalf("ListByTenant B: %v", err)
	}
	for _, c := range cats {
		if c.ID == catID {
			t.Fatalf("cross-tenant leak: tenant B lists tenant A's category %s", catID)
		}
	}
}

// ============================================================================
// Employee documents
// ============================================================================

func TestRepository_EmployeeDoc_CreateGetByID_RoundTripsMetadata(t *testing.T) {
	pool, tenantID, _ := newOffboardFixture(t)
	ctx := employeeDocCtx(tenantID)
	repo := NewPostgresEmployeeDocRepo(pool)
	catRepo := NewPostgresDocCategoryRepo(pool)

	empID := seedNamedUser(t, pool, tenantID, "Doc", "Owner", "docowner-"+uuid.NewString()+"@example.test")
	// Blank name on purpose: exercises the uploaded_by_name email fallback in
	// the same query that TestRepository_GetByID_BlankNameFallsBackToEmail
	// covers for the employee side.
	uploaderID := seedNamedUser(t, pool, tenantID, "", "", "uploader-"+uuid.NewString()+"@example.test")

	cats, err := catRepo.ListByTenant(ctx, tenantID)
	if err != nil || len(cats) == 0 {
		t.Fatalf("ListByTenant (system seeds): err=%v len=%d", err, len(cats))
	}
	catID := cats[0].ID

	docID := uuid.New()
	doc := &models.EmployeeDocument{
		ID: docID, TenantID: tenantID, EmployeeID: empID, CategoryID: catID,
		UploadedBy: uploaderID, Notes: "Testnotiz", Title: "Testdokument",
		FileName: "test.pdf", FileSize: "10 KB", CreatedAt: time.Now(),
	}
	if err := repo.Create(ctx, doc); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_documents", docID) })

	got, err := repo.GetByID(ctx, tenantID, docID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "Testdokument" || got.FileName != "test.pdf" || got.Notes != "Testnotiz" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.UploadedByName == "" {
		t.Errorf("UploadedByName empty — the email fallback for a blank-named uploader did not resolve")
	}
}

func TestRepository_EmployeeDoc_CrossTenantNotFoundOrListed(t *testing.T) {
	pool, tenantA, _ := newOffboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Employee DB Test Tenant B")
	ctxA := employeeDocCtx(tenantA)
	repo := NewPostgresEmployeeDocRepo(pool)
	catRepo := NewPostgresDocCategoryRepo(pool)

	empID := seedNamedUser(t, pool, tenantA, "Doc", "OwnerB", "docownerb-"+uuid.NewString()+"@example.test")
	uploaderID := seedNamedUser(t, pool, tenantA, "Up", "Loader", "uploaderb-"+uuid.NewString()+"@example.test")
	cats, err := catRepo.ListByTenant(ctxA, tenantA)
	if err != nil || len(cats) == 0 {
		t.Fatalf("ListByTenant: err=%v len=%d", err, len(cats))
	}

	docID := uuid.New()
	doc := &models.EmployeeDocument{
		ID: docID, TenantID: tenantA, EmployeeID: empID, CategoryID: cats[0].ID,
		UploadedBy: uploaderID, Title: "Fremdakte", CreatedAt: time.Now(),
	}
	if err := repo.Create(ctxA, doc); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_documents", docID) })

	// Same hr_admin role on both sides — tenant is the only variable, isolating
	// the tenant boundary from the role check the policy also enforces.
	ctxB := employeeDocCtx(tenantB)
	if _, err := repo.GetByID(ctxB, tenantB, docID); !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("GetByID cross-tenant: got %v, want ErrDocumentNotFound", err)
	}

	docsB, err := repo.ListByTenant(ctxB, tenantB)
	if err != nil {
		t.Fatalf("ListByTenant B: %v", err)
	}
	for _, d := range docsB {
		if d.ID == docID {
			t.Fatalf("cross-tenant leak: tenant B lists tenant A's document %s", docID)
		}
	}
}

func TestRepository_EmployeeDoc_ListByEmployee_ScopedToEmployee(t *testing.T) {
	pool, tenantID, _ := newOffboardFixture(t)
	ctx := employeeDocCtx(tenantID)
	repo := NewPostgresEmployeeDocRepo(pool)
	catRepo := NewPostgresDocCategoryRepo(pool)

	empA := seedNamedUser(t, pool, tenantID, "Emp", "A", "empa-"+uuid.NewString()+"@example.test")
	empB := seedNamedUser(t, pool, tenantID, "Emp", "B", "empb-"+uuid.NewString()+"@example.test")
	uploaderID := seedNamedUser(t, pool, tenantID, "Up", "Loader", "uploaderc-"+uuid.NewString()+"@example.test")
	cats, err := catRepo.ListByTenant(ctx, tenantID)
	if err != nil || len(cats) == 0 {
		t.Fatalf("ListByTenant: err=%v len=%d", err, len(cats))
	}

	docA := uuid.New()
	if err := repo.Create(ctx, &models.EmployeeDocument{
		ID: docA, TenantID: tenantID, EmployeeID: empA, CategoryID: cats[0].ID,
		UploadedBy: uploaderID, Title: "Akte A", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Create A: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_documents", docA) })

	docB := uuid.New()
	if err := repo.Create(ctx, &models.EmployeeDocument{
		ID: docB, TenantID: tenantID, EmployeeID: empB, CategoryID: cats[0].ID,
		UploadedBy: uploaderID, Title: "Akte B", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Create B: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_documents", docB) })

	docsA, err := repo.ListByEmployee(ctx, empA)
	if err != nil {
		t.Fatalf("ListByEmployee: %v", err)
	}
	found := false
	for _, d := range docsA {
		if d.EmployeeID != empA {
			t.Fatalf("ListByEmployee leaked document %s of employee %s into employee %s's list", d.ID, d.EmployeeID, empA)
		}
		if d.ID == docA {
			found = true
		}
	}
	if !found {
		t.Fatalf("ListByEmployee did not return the seeded document")
	}
}

func TestRepository_EmployeeDoc_Delete_RemovesRow(t *testing.T) {
	pool, tenantID, _ := newOffboardFixture(t)
	ctx := employeeDocCtx(tenantID)
	repo := NewPostgresEmployeeDocRepo(pool)
	catRepo := NewPostgresDocCategoryRepo(pool)

	empID := seedNamedUser(t, pool, tenantID, "Del", "Ee", "del-"+uuid.NewString()+"@example.test")
	uploaderID := seedNamedUser(t, pool, tenantID, "Up", "Loader", "uploaderd-"+uuid.NewString()+"@example.test")
	cats, err := catRepo.ListByTenant(ctx, tenantID)
	if err != nil || len(cats) == 0 {
		t.Fatalf("ListByTenant: err=%v len=%d", err, len(cats))
	}

	docID := uuid.New()
	if err := repo.Create(ctx, &models.EmployeeDocument{
		ID: docID, TenantID: tenantID, EmployeeID: empID, CategoryID: cats[0].ID,
		UploadedBy: uploaderID, Title: "Zu Loeschen", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, tenantID, docID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := repo.GetByID(ctx, tenantID, docID); !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("GetByID after delete: got %v, want ErrDocumentNotFound", err)
	}
}
