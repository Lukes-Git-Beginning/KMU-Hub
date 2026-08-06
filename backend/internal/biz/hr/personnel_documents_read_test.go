package hr

// hr-personnel-documents — read-side proof for GET /api/v1/hr/personnel-documents.
//
// TestHRRoleBased_DocumentAccess_DB already proves the hr_document_access
// policy (mig 000127) itself. What it does not cover is whether the new
// tenant-wide read path actually goes through that policy: ListByTenant has no
// visibility clause of its own by design, so if the query ever ran outside RLS
// it would hand every tier to every caller and nothing else would notice. This
// test drives the repository methods the handler calls, not raw SQL.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func containsDoc(docs []*models.EmployeeDocument, id uuid.UUID) bool {
	for _, d := range docs {
		if d.ID == id {
			return true
		}
	}
	return false
}

func TestPersonnelDocuments_ListByTenant_RespectsVisibility_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// Own tenants rather than the shared TenantA/TenantB, so a parallel test
	// seeding its own documents cannot change the counts asserted here.
	tenantOwn := uuid.New()
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Personalakte Own")
	testutil.EnsureTenant(t, pool, tenantOther, "Personalakte Other")

	seedUser := func(tenantID uuid.UUID, prefix string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantID,
			"email":         prefix + "-" + uuid.New().String()[:8] + "@pd.local",
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}

	emp := seedUser(tenantOwn, "emp")
	stranger := seedUser(tenantOwn, "stranger")
	hrAdmin := seedUser(tenantOwn, "hr")
	outsider := seedUser(tenantOther, "outsider")

	seedCategory := func(visibility string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
			"tenant_id":  tenantOwn,
			"name":       "cat-" + visibility,
			"key":        visibility + "-" + uuid.New().String()[:6],
			"visibility": visibility,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_document_categories", id) })
		return id
	}

	catHROnly := seedCategory("hr_only")
	catManager := seedCategory("manager")
	catEmployee := seedCategory("employee")

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

	docHROnly := seedDoc(catHROnly, "Abmahnung")
	docManager := seedDoc(catManager, "Zeugnis")
	docEmployee := seedDoc(catEmployee, "Bescheinigung")

	repo := employee.NewPostgresEmployeeDocRepo(pool)

	ctxFor := func(tenantID, userID uuid.UUID, roles ...string) context.Context {
		return withRoles(testutil.WithTenantCtx(context.Background(), tenantID), userID, roles...)
	}

	t.Run("hr_admin sees every tier", func(t *testing.T) {
		docs, err := repo.ListByTenant(ctxFor(tenantOwn, hrAdmin, "hr_admin"), tenantOwn)
		if err != nil {
			t.Fatalf("ListByTenant: %v", err)
		}
		for _, id := range []uuid.UUID{docHROnly, docManager, docEmployee} {
			if !containsDoc(docs, id) {
				t.Errorf("hr_admin should see document %s", id)
			}
		}
	})

	t.Run("the employee themself sees only the employee tier", func(t *testing.T) {
		docs, err := repo.ListByTenant(ctxFor(tenantOwn, emp, "member"), tenantOwn)
		if err != nil {
			t.Fatalf("ListByTenant: %v", err)
		}
		if containsDoc(docs, docHROnly) || containsDoc(docs, docManager) {
			t.Error("a member must not see hr_only or manager documents in the tenant-wide list")
		}
		if !containsDoc(docs, docEmployee) {
			t.Error("the employee should see their own employee-tier document")
		}
	})

	// The case the Akte exists to prevent: a colleague without an HR role must
	// not read a stranger's file just because the list is tenant-wide.
	t.Run("a colleague without an HR role sees nothing of a stranger", func(t *testing.T) {
		docs, err := repo.ListByTenant(ctxFor(tenantOwn, stranger, "member"), tenantOwn)
		if err != nil {
			t.Fatalf("ListByTenant: %v", err)
		}
		for _, id := range []uuid.UUID{docHROnly, docManager, docEmployee} {
			if containsDoc(docs, id) {
				t.Errorf("stranger must not see document %s of another employee", id)
			}
		}
	})

	t.Run("another tenant sees nothing, hr_admin role or not", func(t *testing.T) {
		docs, err := repo.ListByTenant(ctxFor(tenantOther, outsider, "hr_admin"), tenantOther)
		if err != nil {
			t.Fatalf("ListByTenant: %v", err)
		}
		for _, id := range []uuid.UUID{docHROnly, docManager, docEmployee} {
			if containsDoc(docs, id) {
				t.Errorf("cross-tenant read leaked document %s", id)
			}
		}
	})

	// Guessing an id must not be a way around the list filter.
	t.Run("GetByID is not a back door around the tier filter", func(t *testing.T) {
		if _, err := repo.GetByID(ctxFor(tenantOwn, stranger, "member"), tenantOwn, docHROnly); err == nil {
			t.Error("stranger resolved an hr_only document by id")
		}
		if _, err := repo.GetByID(ctxFor(tenantOther, outsider, "hr_admin"), tenantOther, docEmployee); err == nil {
			t.Error("cross-tenant GetByID resolved a document")
		}
		if _, err := repo.GetByID(ctxFor(tenantOwn, hrAdmin, "hr_admin"), tenantOwn, docHROnly); err != nil {
			t.Errorf("hr_admin should resolve the document by id: %v", err)
		}
	})
}
