package gateway

// The GoBD Belegarchiv is a §147 AO retention surface (see fix-gobd-belegarchiv-worm-enforcement
// in JOURNAL.md, iteration 12/13-ish of this run's Block A): kmuhub_app has no UPDATE/DELETE on
// the archive tables anymore, so this route group is the only door through which a document
// enters or leaves the archive. These tests pin the HTTP-layer contract: which field is required,
// where the 50 MiB ceiling actually bites, and that the write actions stay behind their own
// gobd-archive:write/:read permission keys rather than falling back to a coarser finance guard.
//
// This package has no bufconn/fake-client harness for FinanceServiceClient (same boundary noted
// in route_bexio_test.go and every other gateway coverage unit this run), so a successful RPC
// round trip — and with it the "fremder Tenant liefert 404" case, which is enforced server-side
// by the biz service's tenant-scoped queries — cannot be exercised here. What IS provable at this
// layer: the tenant ID always comes from the JWT-derived context (getTenantID), never from the
// request body or a URL/query parameter — there is no code path here that would let a caller
// override it.

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// bizRouter registers all Finance routes on a fresh chi router, mirroring how
// cmd/gateway/main.go mounts BizRoutes, so permission-guard tests exercise the real middleware
// chain (authMiddleware -> RequirePermission) instead of calling handlers directly. Shared with
// route_biz_einvoice_test.go — it mounts every finance route, not just gobd-archive.
func bizRouter(registry *ServiceRegistry) *chi.Mux {
	r := chi.NewRouter()
	NewBizRoutes(registry).RegisterRoutes(r, guardTestAuth)
	return r
}

// --- HandleArchiveDocument ---

func TestHandleArchiveDocument_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleArchiveDocument)
}

func TestHandleArchiveDocument_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, map[string]string{"doc_type": "invoice"}, "file", "beleg.pdf", []byte("%PDF-1.4"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive", body)
	req.Header.Set("Content-Type", contentType)
	routes.HandleArchiveDocument(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleArchiveDocument_InvalidMultipartForm(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	req = withTenantID(req, testTenantID)
	routes.HandleArchiveDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "failed to parse multipart form")
}

func TestHandleArchiveDocument_MissingFile(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, map[string]string{"doc_type": "invoice"}, "", "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleArchiveDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "missing field 'file'")
}

func TestHandleArchiveDocument_MissingDocType(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, map[string]string{}, "file", "beleg.pdf", []byte("%PDF-1.4"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleArchiveDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "missing field 'doc_type'")
}

// TestHandleArchiveDocument_ExceedsSizeLimit locks down the 50 MiB ceiling
// (maxGobdDocumentBytes): a file one byte over it must be rejected with 413,
// not silently truncated and archived.
func TestHandleArchiveDocument_ExceedsSizeLimit(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	oversized := bytes.Repeat([]byte{0}, maxGobdDocumentBytes+1)
	body, contentType := multipartBody(t, map[string]string{"doc_type": "invoice"}, "file", "beleg.pdf", oversized)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleArchiveDocument(rec, req)
	assertStatus(t, rec, http.StatusRequestEntityTooLarge)
	assertErrorContains(t, rec, "exceeds 50 MiB limit")
}

func TestHandleArchiveDocument_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, map[string]string{"doc_type": "invoice", "source_invoice_id": "550e8400-e29b-41d4-a716-446655440000"}, "file", "beleg.pdf", []byte("%PDF-1.4"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleArchiveDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleArchiveDocument_PermissionGuard proves the route stays behind its
// own gobd-archive:write key rather than a coarser finance:write — a caller
// with only general finance write access must NOT be able to write into the
// retention-locked archive.
func TestHandleArchiveDocument_PermissionGuard(t *testing.T) {
	r := bizRouter(emptyRegistry())
	body, contentType := multipartBody(t, map[string]string{"doc_type": "invoice"}, "file", "beleg.pdf", []byte("%PDF-1.4"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive", body)
	req.Header.Set("Content-Type", contentType)
	req = withPermissions(req, "finance:write")
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)

	body2, contentType2 := multipartBody(t, map[string]string{"doc_type": "invoice"}, "file", "beleg.pdf", []byte("%PDF-1.4"))
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive", body2)
	req2.Header.Set("Content-Type", contentType2)
	req2 = withPermissions(req2, "gobd-archive:write")
	r.ServeHTTP(rec2, req2)
	assertStatus(t, rec2, http.StatusServiceUnavailable) // guard passed, empty registry stops it
}

// --- HandleArchiveInvoiceDocument ---

func TestHandleArchiveInvoiceDocument_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleArchiveInvoiceDocument)
}

func TestHandleArchiveInvoiceDocument_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive/from-invoice/x", nil)
	req = withChiURLParam(req, "invoiceId", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleArchiveInvoiceDocument(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleArchiveInvoiceDocument_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive/from-invoice/x", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "invoiceId", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleArchiveInvoiceDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListGobdDocuments ---

func TestHandleListGobdDocuments_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListGobdDocuments)
}

func TestHandleListGobdDocuments_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/gobd-archive", nil)
	routes.HandleListGobdDocuments(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListGobdDocuments_RPCFails(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/gobd-archive?doc_type=invoice&page=2&per_page=25", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListGobdDocuments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetGobdDocument ---

func TestHandleGetGobdDocument_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetGobdDocument)
}

func TestHandleGetGobdDocument_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/gobd-archive/x", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetGobdDocument(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetGobdDocument_RPCFails(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/gobd-archive/x", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetGobdDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDownloadGobdDocument ---

func TestHandleDownloadGobdDocument_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDownloadGobdDocument)
}

func TestHandleDownloadGobdDocument_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/gobd-archive/x/download", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDownloadGobdDocument(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDownloadGobdDocument_RPCFails(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/gobd-archive/x/download", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDownloadGobdDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleAddDocumentAnnotation ---

func TestHandleAddDocumentAnnotation_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleAddDocumentAnnotation)
}

func TestHandleAddDocumentAnnotation_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive/x/annotations", jsonBody(t, map[string]interface{}{
		"note": "checked against the original invoice",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddDocumentAnnotation(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleAddDocumentAnnotation_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive/x/annotations", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddDocumentAnnotation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleAddDocumentAnnotation_EmptyNote(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive/x/annotations", jsonBody(t, map[string]interface{}{
		"note": "",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddDocumentAnnotation(rec, req)
	assertValidationError(t, rec, "note")
}

func TestHandleAddDocumentAnnotation_NoteTooLong(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive/x/annotations", jsonBody(t, map[string]interface{}{
		"note": strings.Repeat("a", 2001),
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddDocumentAnnotation(rec, req)
	assertValidationError(t, rec, "note")
}

func TestHandleAddDocumentAnnotation_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive/x/annotations", jsonBody(t, map[string]interface{}{
		"note": "checked against the original invoice",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddDocumentAnnotation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleAddDocumentAnnotation_PermissionGuard proves annotation writes
// require gobd-archive:write specifically, same reasoning as
// TestHandleArchiveDocument_PermissionGuard.
func TestHandleAddDocumentAnnotation_PermissionGuard(t *testing.T) {
	r := bizRouter(emptyRegistry())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive/550e8400-e29b-41d4-a716-446655440000/annotations", jsonBody(t, map[string]interface{}{
		"note": "checked against the original invoice",
	}))
	req = withPermissions(req, "gobd-archive:read")
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/finance/gobd-archive/550e8400-e29b-41d4-a716-446655440000/annotations", jsonBody(t, map[string]interface{}{
		"note": "checked against the original invoice",
	}))
	req2 = withPermissions(req2, "gobd-archive:write")
	r.ServeHTTP(rec2, req2)
	assertStatus(t, rec2, http.StatusServiceUnavailable)
}

// --- parsePageParam ---

func TestParsePageParam(t *testing.T) {
	cases := []struct {
		in   string
		want int32
	}{
		{"", 0},
		{"0", 0},
		{"1", 1},
		{"25", 25},
		{"-1", 0},   // leading '-' is not a digit -> rejected, not parsed as negative
		{"1x", 0},   // any non-digit rune anywhere -> rejected wholesale
		{"abc", 0},
		{"007", 7},
	}
	for _, tc := range cases {
		if got := parsePageParam(tc.in); got != tc.want {
			t.Errorf("parsePageParam(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
