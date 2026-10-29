package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// RLS policy hr_document_access decides the Personalakte tiers from app.user_roles. The roles
// did not cross the gateway->biz hop, so the GUC was empty in the HR service: admin and hr_admin
// read an empty file and every upload failed WITH CHECK. The chain is the production one: the
// real seed through ResolveTokenPermissions, the real Auth middleware and route registration, a
// real HR gRPC server behind TenantInboundUnaryInterceptor and the Postgres employee repositories
// as kmuhub_app.
//
// manager crosses the hop too and reads a direct report's file (manager and employee tier, no
// salary statement without team:salary:view), but nothing beyond the reporting line: the
// Personalakte list and a stranger's file stay as closed as before.
func TestHRDocuments_RolesReachTheDocumentPolicy(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "HR Documents Roles")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	employeeSvc := employee.NewService(employee.NewPostgresEmployeeRepo(pool),
		employee.NewPostgresDocCategoryRepo(pool), employee.NewPostgresEmployeeDocRepo(pool))
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	hrv1.RegisterHRServiceServer(srv, server.NewHRGRPCServer(nil, nil, employeeSvc, nil, nil, nil))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	NewHRRoutes(reg, nil).RegisterRoutes(router, middleware.Auth(authSvc))

	userFor := func(role string) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "email": "hr-doc-roles-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Akte", "last_name": role,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
		var roleID uuid.UUID
		if err := pool.QueryRow(sysCtx,
			`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, role).Scan(&roleID); err != nil {
			t.Fatalf("preset role %q: %v", role, err)
		}
		if _, err := pool.Exec(sysCtx,
			`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
			userID, roleID, tenantID); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, tenantID, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, tenantID.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return userID, token
	}

	adminID, adminToken := userFor("admin")
	_, hrAdminToken := userFor("hr_admin")
	managerID, managerToken := userFor("manager")
	memberID, memberToken := userFor("member")

	category := map[string]uuid.UUID{}
	for _, tier := range []string{"hr_only", "manager", "employee"} {
		category[tier] = testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "name": "Stufe " + tier, "key": "roles-" + tier, "visibility": tier,
		})
		id := category[tier]
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_document_categories", id) })
	}
	seedDoc := func(employeeID uuid.UUID, tier string) string {
		return testutil.SeedRow(t, pool, "hr_employee_documents", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "employee_id": employeeID, "category_id": category[tier],
			"uploaded_by": adminID, "title": tier, "file_name": tier + ".pdf",
		}).String()
	}
	memberHROnly := seedDoc(memberID, "hr_only")
	memberManagerTier := seedDoc(memberID, "manager")
	memberEmployeeTier := seedDoc(memberID, "employee")
	managerOwn := seedDoc(managerID, "employee")

	// The reporting line: member reports to manager, stranger to admin. The salary statement
	// lives in the employee tier (000310), so only the key keeps it from the manager.
	strangerID, _ := userFor("member")
	strangerManagerTier := seedDoc(strangerID, "manager")
	// The nullable text columns are written as "" like the application does: the profile read
	// scans them into plain strings.
	for report, boss := range map[uuid.UUID]uuid.UUID{memberID: managerID, strangerID: adminID} {
		profileID := testutil.SeedRow(t, pool, "hr_employee_profiles", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "user_id": report, "manager_user_id": boss, "start_date": "2026-01-01",
			"department": "", "position_title": "", "address_city": "", "address_street": "", "address_postal_code": "",
			"emergency_contact_name": "", "emergency_contact_phone": "",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", profileID) })
	}
	category["salary"] = testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "Gehaltsabrechnung", "key": employee.SalaryDocumentCategoryKey,
		"visibility": "employee",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_document_categories", category["salary"]) })
	memberSalary := seedDoc(memberID, "salary")
	// Registered last, so it runs before the category and user cleanups; it also removes the
	// documents the upload subtests create.
	t.Cleanup(func() {
		if _, err := pool.Exec(sysCtx, `DELETE FROM hr_employee_documents WHERE tenant_id = $1`, tenantID); err != nil {
			t.Errorf("cleanup documents: %v", err)
		}
	})

	do := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&buf).Encode(body); err != nil {
				t.Fatalf("encode: %v", err)
			}
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	list := func(t *testing.T, who, path, token string) []string {
		t.Helper()
		rec := do(http.MethodGet, path, token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s GET %s = %d (body %s)", who, path, rec.Code, rec.Body.String())
		}
		var resp struct {
			Documents []struct {
				ID string `json:"id"`
			} `json:"documents"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v (body %s)", err, rec.Body.String())
		}
		ids := make([]string, 0, len(resp.Documents))
		for _, d := range resp.Documents {
			ids = append(ids, d.ID)
		}
		slices.Sort(ids)
		return ids
	}
	expectIDs := func(t *testing.T, who string, got []string, want ...string) {
		t.Helper()
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("%s sees %v, want %v", who, got, want)
		}
	}
	created := func(t *testing.T, who string, rec *httptest.ResponseRecorder) string {
		t.Helper()
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s upload = %d, want 201 (body %s)", who, rec.Code, rec.Body.String())
		}
		// /{id}/documents wraps the document, /personnel-documents does not.
		var doc struct {
			ID       string `json:"id"`
			Document struct {
				ID string `json:"id"`
			} `json:"document"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil || doc.ID+doc.Document.ID == "" {
			t.Fatalf("%s upload body %s: %v", who, rec.Body.String(), err)
		}
		return doc.ID + doc.Document.ID
	}

	t.Run("admin reads every tier of an employee's file", func(t *testing.T) {
		expectIDs(t, "admin", list(t, "admin", "/api/v1/hr/employees/"+memberID.String()+"/documents", adminToken),
			memberHROnly, memberManagerTier, memberEmployeeTier, memberSalary)
	})

	t.Run("admin and hr_admin read the whole Personalakte", func(t *testing.T) {
		all := []string{memberHROnly, memberManagerTier, memberEmployeeTier, memberSalary, managerOwn, strangerManagerTier}
		expectIDs(t, "admin", list(t, "admin", "/api/v1/hr/personnel-documents", adminToken), all...)
		expectIDs(t, "hr_admin", list(t, "hr_admin", "/api/v1/hr/personnel-documents", hrAdminToken), all...)
	})

	t.Run("manager and member see only their own employee-tier documents", func(t *testing.T) {
		expectIDs(t, "manager", list(t, "manager", "/api/v1/hr/personnel-documents", managerToken), managerOwn)
		expectIDs(t, "member", list(t, "member", "/api/v1/hr/personnel-documents", memberToken), memberEmployeeTier, memberSalary)
	})

	reportFile := "/api/v1/hr/employees/" + memberID.String() + "/documents"

	t.Run("manager reads a report's file without hr_only and salary statements", func(t *testing.T) {
		expectIDs(t, "manager", list(t, "manager", reportFile, managerToken), memberManagerTier, memberEmployeeTier)
	})

	t.Run("hr_admin reads the report's salary statement", func(t *testing.T) {
		expectIDs(t, "hr_admin", list(t, "hr_admin", reportFile, hrAdminToken),
			memberHROnly, memberManagerTier, memberEmployeeTier, memberSalary)
	})

	t.Run("manager reads their own file through the same route", func(t *testing.T) {
		expectIDs(t, "manager", list(t, "manager", "/api/v1/hr/employees/"+managerID.String()+"/documents", managerToken), managerOwn)
	})

	t.Run("manager is refused a file outside the reporting line", func(t *testing.T) {
		// strangerID reports to admin; adminID has no profile at all. Both answer alike.
		for _, id := range []uuid.UUID{strangerID, adminID} {
			if rec := do(http.MethodGet, "/api/v1/hr/employees/"+id.String()+"/documents", managerToken, nil); rec.Code != http.StatusForbidden {
				t.Errorf("manager GET file of %s = %d, want 403 (body %s)", id, rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("member reads only their own file", func(t *testing.T) {
		expectIDs(t, "member", list(t, "member", reportFile, memberToken), memberEmployeeTier, memberSalary)
		if rec := do(http.MethodGet, "/api/v1/hr/employees/"+managerID.String()+"/documents", memberToken, nil); rec.Code != http.StatusForbidden {
			t.Errorf("member GET colleague's file = %d, want 403", rec.Code)
		}
	})

	t.Run("admin and hr_admin upload into an employee's file", func(t *testing.T) {
		byAdmin := created(t, "admin", do(http.MethodPost, "/api/v1/hr/employees/"+memberID.String()+"/documents", adminToken,
			map[string]any{"category_id": category["employee"].String(), "fileName": "vertrag.pdf"}))
		byHRAdmin := created(t, "hr_admin", do(http.MethodPost, "/api/v1/hr/personnel-documents", hrAdminToken,
			map[string]any{"employee_id": memberID.String(), "title": "Zeugnis", "category": "roles-employee", "file_name": "zeugnis.pdf"}))
		// The employee finds both in their own file; the manager still does not.
		expectIDs(t, "member", list(t, "member", "/api/v1/hr/personnel-documents", memberToken),
			memberEmployeeTier, memberSalary, byAdmin, byHRAdmin)
		expectIDs(t, "manager", list(t, "manager", "/api/v1/hr/personnel-documents", managerToken), managerOwn)
	})

	t.Run("manager and member cannot upload", func(t *testing.T) {
		body := map[string]any{"employee_id": memberID.String(), "title": "x", "category": "roles-employee", "file_name": "x.pdf"}
		if rec := do(http.MethodPost, "/api/v1/hr/personnel-documents", managerToken, body); rec.Code != http.StatusForbidden {
			t.Errorf("manager upload = %d, want 403", rec.Code)
		}
		if rec := do(http.MethodPost, "/api/v1/hr/personnel-documents", memberToken, body); rec.Code != http.StatusForbidden {
			t.Errorf("member upload = %d, want 403", rec.Code)
		}
	})
}
