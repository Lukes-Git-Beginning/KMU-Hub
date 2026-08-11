package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// ============================================================================
// GET /api/v1/hr/employees/me/documents — self-service Akte access
//
// The route exists because neither existing route covers an employee reading
// their OWN documents: /{id}/documents is guarded by "hr:read", which
// migration 000129 grants to admin alone, and /hr/personnel-documents returns
// every row the caller's RLS tier permits (for a manager that includes their
// colleagues' employee-tier documents).
//
// These are routing-level tests: the registry is empty, so a request that
// reaches the handler dies at the gRPC client with 503. 404 would mean the
// path is unmatched, 403 that the guard locks the very roles out that the
// route exists for.
// ============================================================================

const testOwnDocumentsPath = "/api/v1/hr/employees/me/documents"

// TestHandleListOwnDocuments_MatchesFrontendPath pins the path and proves a
// member — who holds team:documents:view at scope 'own' (migration 000256) but
// not hr:read — actually reaches the handler.
func TestHandleListOwnDocuments_MatchesFrontendPath(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, testOwnDocumentsPath, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withPermissions(req, "team:documents:view")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListOwnDocuments_LegacyPermissionStillWorks pins that the legacy
// "hr:read" key reaches the route too — additive guard, never a replacement,
// because permissions are baked into the JWT at login (see route_hr.go).
func TestHandleListOwnDocuments_LegacyPermissionStillWorks(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, testOwnDocumentsPath, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withPermissions(req, "hr:read")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListOwnDocuments_RequiresPermission verifies a caller with neither
// key is rejected rather than falling through to an unguarded read.
func TestHandleListOwnDocuments_RequiresPermission(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, testOwnDocumentsPath, nil)
	req = withAuth(req, "user-123", testTenantID)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
}

// TestListEmployeeDocuments_StillAdminOnly is the counterpart that matters
// most: adding the /me route must NOT have widened the per-employee route.
// A holder of team:documents:view alone must still be turned away from a
// colleague's Akte — otherwise the new self-service path would have opened
// every employee's documents to every member.
func TestListEmployeeDocuments_StillAdminOnly(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000/documents", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withPermissions(req, "team:documents:view")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
}

// TestHandleListOwnDocuments_MissingUserContext covers the failure path: the
// employee id comes from the JWT, so a request without one must be rejected
// rather than resolved to the zero uuid — which would otherwise reach the RPC
// as a lookup for "the employee with no id".
func TestHandleListOwnDocuments_MissingUserContext(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, testOwnDocumentsPath, nil)
	req = withTenantID(req, testTenantID)

	routes.HandleListOwnDocuments(rec, req)

	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing user context")
}

// ============================================================================
// filterDocumentsByCategory
//
// The handler cannot be exercised end-to-end without a live gRPC server (this
// repo has no bufconn harness), so the one piece of logic it carries is tested
// directly.
// ============================================================================

func docWithCategory(key string) *hrv1.EmployeeDocument {
	return &hrv1.EmployeeDocument{Id: "doc-" + key, CategoryKey: key}
}

func TestFilterDocumentsByCategory(t *testing.T) {
	docs := []*hrv1.EmployeeDocument{
		docWithCategory("gehaltsabrechnung"),
		docWithCategory("vertrag"),
		docWithCategory("gehaltsabrechnung"),
	}

	tests := []struct {
		name     string
		category string
		wantIDs  []string
	}{
		{
			name:     "empty category returns the list untouched",
			category: "",
			wantIDs:  []string{"doc-gehaltsabrechnung", "doc-vertrag", "doc-gehaltsabrechnung"},
		},
		{
			name:     "whitespace-only category is treated as no filter",
			category: "   ",
			wantIDs:  []string{"doc-gehaltsabrechnung", "doc-vertrag", "doc-gehaltsabrechnung"},
		},
		{
			name:     "known key keeps only that category",
			category: "gehaltsabrechnung",
			wantIDs:  []string{"doc-gehaltsabrechnung", "doc-gehaltsabrechnung"},
		},
		{
			name:     "surrounding whitespace is trimmed",
			category: "  gehaltsabrechnung  ",
			wantIDs:  []string{"doc-gehaltsabrechnung", "doc-gehaltsabrechnung"},
		},
		{
			name:     "unknown key yields nothing rather than everything",
			category: "does-not-exist",
			wantIDs:  []string{},
		},
		{
			name:     "partial key does not match",
			category: "gehalt",
			wantIDs:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filterDocumentsByCategory(docs, tc.category)
			if len(got) != len(tc.wantIDs) {
				t.Fatalf("got %d documents, want %d", len(got), len(tc.wantIDs))
			}
			for i, want := range tc.wantIDs {
				if got[i].GetId() != want {
					t.Errorf("document %d: got id %q, want %q", i, got[i].GetId(), want)
				}
			}
		})
	}
}

// TestFilterDocumentsByCategory_EmptyResultIsNotNil pins the wire shape: an
// employee with no salary statements must serialize as [] rather than null,
// or the frontend list renders as an error instead of an empty state.
func TestFilterDocumentsByCategory_EmptyResultIsNotNil(t *testing.T) {
	got := filterDocumentsByCategory(nil, "gehaltsabrechnung")
	if got == nil {
		t.Fatal("filtered result is nil; protojson would emit null instead of []")
	}
	if len(got) != 0 {
		t.Fatalf("got %d documents from a nil input, want 0", len(got))
	}
}
