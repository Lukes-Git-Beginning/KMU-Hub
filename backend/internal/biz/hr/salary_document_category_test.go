package hr

// hr-salary-document-category (migration 000310) — proves the seeded
// "gehaltsabrechnung" system category actually reaches an employee through
// the visibility tier it was given, and stays out of reach of an hr_only
// tier the same way any other employee-visible category would.

import (
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSalaryDocumentCategory_SeedRow_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	var visibility string
	var isSystem bool
	err := pool.QueryRow(t.Context(),
		`SELECT visibility, is_system FROM hr_document_categories
		 WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND key = 'gehaltsabrechnung'`,
	).Scan(&visibility, &isSystem)
	if err != nil {
		t.Fatalf("seeded gehaltsabrechnung category not found: %v", err)
	}
	if visibility != "employee" {
		t.Errorf("visibility = %q, want employee", visibility)
	}
	if !isSystem {
		t.Error("is_system = false, want true")
	}
}

func TestSalaryDocumentCategory_VisibleToEmployee_NotHROnly_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Salary Doc Category Own")

	seedUser := func(prefix string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantOwn,
			"email":         prefix + "-" + uuid.New().String()[:8] + "@salarydoc.local",
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}

	emp := seedUser("emp")
	stranger := seedUser("stranger")
	hrAdmin := seedUser("hr")

	var catSalary uuid.UUID
	err := pool.QueryRow(t.Context(),
		`SELECT id FROM hr_document_categories
		 WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND key = 'gehaltsabrechnung'`,
	).Scan(&catSalary)
	if err != nil {
		t.Fatalf("resolve gehaltsabrechnung category id: %v", err)
	}

	catHROnly := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "cat-hr_only",
		"key":        "hr_only-" + uuid.New().String()[:6],
		"visibility": "hr_only",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_document_categories", catHROnly) })

	seedDoc := func(categoryID uuid.UUID, title string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "hr_employee_documents", map[string]any{
			"tenant_id":   tenantOwn,
			"employee_id": emp,
			"category_id": categoryID,
			"uploaded_by": hrAdmin,
			"title":       title,
			"file_name":   title + ".pdf",
			"file_size":   "120 KB",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_documents", id) })
		return id
	}

	docSalary := seedDoc(catSalary, "Lohnabrechnung 2026-07")
	docHROnly := seedDoc(catHROnly, "Abmahnung")

	repo := employee.NewPostgresEmployeeDocRepo(pool)

	t.Run("the employee sees their own salary statement", func(t *testing.T) {
		docs, err := repo.ListByTenant(withRoles(testutil.WithTenantCtx(t.Context(), tenantOwn), emp, "member"), tenantOwn)
		if err != nil {
			t.Fatalf("ListByTenant: %v", err)
		}
		if !containsDoc(docs, docSalary) {
			t.Error("employee should see their own gehaltsabrechnung document")
		}
		if containsDoc(docs, docHROnly) {
			t.Error("employee must not see the hr_only document")
		}
	})

	t.Run("a colleague without an HR role sees neither", func(t *testing.T) {
		docs, err := repo.ListByTenant(withRoles(testutil.WithTenantCtx(t.Context(), tenantOwn), stranger, "member"), tenantOwn)
		if err != nil {
			t.Fatalf("ListByTenant: %v", err)
		}
		if containsDoc(docs, docSalary) || containsDoc(docs, docHROnly) {
			t.Error("a colleague must not see another employee's salary statement or hr_only document")
		}
	})

	t.Run("hr_admin sees both tiers", func(t *testing.T) {
		docs, err := repo.ListByTenant(withRoles(testutil.WithTenantCtx(t.Context(), tenantOwn), hrAdmin, "hr_admin"), tenantOwn)
		if err != nil {
			t.Fatalf("ListByTenant: %v", err)
		}
		if !containsDoc(docs, docSalary) || !containsDoc(docs, docHROnly) {
			t.Error("hr_admin should see both the salary statement and the hr_only document")
		}
	})
}
