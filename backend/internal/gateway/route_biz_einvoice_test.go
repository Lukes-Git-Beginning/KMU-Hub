package gateway

// route_biz_einvoice_test.go covers the e-invoice import route group: the incoming-invoice
// MIME gate (application/pdf, application/xml, text/xml, and the octet-stream fallback
// browsers send for untyped uploads), the 10 MiB size ceiling, and the read/write permission
// keys the group is mounted under (route_biz.go:111 and :300-304 — both stay on the
// legacy-only "finance" guard, not a split finance:invoice catalogue key, per the comment at
// route_biz.go:50-57).
//
// Like route_biz_gobd_archive_test.go, this package has no bufconn/fake-client harness for
// FinanceServiceClient, so a successful RPC round trip — and with it the "fremder Tenant
// liefert 404" case, which is enforced server-side by the biz service's tenant-scoped queries —
// cannot be exercised here. "ReachesRPC" tests assert the handler gets past its own validation
// and hits the (unreachable) dummy connection, which manifests as 503.

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
)

// --- isEInvoiceMIME ---

func TestIsEInvoiceMIME(t *testing.T) {
	cases := []struct {
		mime string
		want bool
	}{
		{"application/pdf", true},
		{"application/xml", true},
		{"text/xml", true},
		{"application/octet-stream", true}, // browsers may send this for untyped .pdf/.xml uploads
		{"image/png", false},
		{"application/json", false},
		{"", false},
		{"APPLICATION/PDF", false}, // exact match only, no case-folding
	}
	for _, tc := range cases {
		if got := isEInvoiceMIME(tc.mime); got != tc.want {
			t.Errorf("isEInvoiceMIME(%q) = %v, want %v", tc.mime, got, tc.want)
		}
	}
}

// multipartBodyWithFileContentType is like multipartBody but lets the caller set an explicit
// Content-Type on the file part. multipart.Writer.CreateFormFile hardcodes
// application/octet-stream on every file part, which can never exercise isEInvoiceMIME's
// rejection path through the HTTP layer.
func multipartBodyWithFileContentType(t *testing.T, fileField, fileName, fileContentType string, fileContent []byte) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="`+fileField+`"; filename="`+fileName+`"`)
	header.Set("Content-Type", fileContentType)
	fw, err := w.CreatePart(header)
	if err != nil {
		t.Fatalf("create form part: %v", err)
	}
	if _, err := fw.Write(fileContent); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return buf, w.FormDataContentType()
}

// --- HandleImportInvoice ---

func TestHandleImportInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleImportInvoice)
}

func TestHandleImportInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, nil, "file", "rechnung.pdf", []byte("%PDF-1.4"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/import", body)
	req.Header.Set("Content-Type", contentType)
	routes.HandleImportInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleImportInvoice_InvalidMultipartForm(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/import", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	req = withTenantID(req, testTenantID)
	routes.HandleImportInvoice(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid multipart form")
}

func TestHandleImportInvoice_MissingFile(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, nil, "", "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleImportInvoice(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "file field is required")
}

// TestHandleImportInvoice_UnsupportedMIMEType locks down the 415 path: a file whose
// part Content-Type isn't PDF/XML/octet-stream must be rejected before it ever reaches
// the biz RPC, not silently accepted and forwarded to a downstream validator.
func TestHandleImportInvoice_UnsupportedMIMEType(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBodyWithFileContentType(t, "file", "rechnung.png", "image/png", []byte("not an invoice"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleImportInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnsupportedMediaType)
	assertErrorContains(t, rec, "unsupported file type")
}

func TestHandleImportInvoice_EmptyFile(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, nil, "file", "leer.pdf", []byte{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleImportInvoice(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "uploaded file is empty")
}

// TestHandleImportInvoice_ExceedsSizeLimit locks down the 10 MiB ceiling
// (maxEInvoiceUploadBytes): a file one byte over it must be rejected with 413,
// not silently truncated and imported.
func TestHandleImportInvoice_ExceedsSizeLimit(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	oversized := bytes.Repeat([]byte{0}, maxEInvoiceUploadBytes+1)
	body, contentType := multipartBody(t, nil, "file", "riesig.pdf", oversized)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleImportInvoice(rec, req)
	assertStatus(t, rec, http.StatusRequestEntityTooLarge)
	assertErrorContains(t, rec, "exceeds 10 MiB limit")
}

func TestHandleImportInvoice_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, nil, "file", "rechnung.pdf", []byte("%PDF-1.4"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleImportInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleImportInvoice_PermissionGuard proves the import route requires finance:write
// (route_biz.go:111) and rejects an unrelated permission.
func TestHandleImportInvoice_PermissionGuard(t *testing.T) {
	r := bizRouter(emptyRegistry())
	body, contentType := multipartBody(t, nil, "file", "rechnung.pdf", []byte("%PDF-1.4"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withPermissions(req, "crm:write")
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)

	body2, contentType2 := multipartBody(t, nil, "file", "rechnung.pdf", []byte("%PDF-1.4"))
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/finance/invoices/import", body2)
	req2.Header.Set("Content-Type", contentType2)
	req2 = withPermissions(req2, "finance:write")
	r.ServeHTTP(rec2, req2)
	assertStatus(t, rec2, http.StatusServiceUnavailable) // guard passed, empty registry stops it
}

// --- HandleListIncomingInvoices ---

func TestHandleListIncomingInvoices_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListIncomingInvoices)
}

func TestHandleListIncomingInvoices_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/incoming-invoices", nil)
	routes.HandleListIncomingInvoices(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListIncomingInvoices_RPCFails(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/incoming-invoices?status=reviewed&date_from=2026-01-01&date_to=2026-12-31&page=2&per_page=25", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListIncomingInvoices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListIncomingInvoices_PermissionGuard(t *testing.T) {
	r := bizRouter(emptyRegistry())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/incoming-invoices", nil)
	req = withPermissions(req, "crm:read")
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/finance/incoming-invoices", nil)
	req2 = withPermissions(req2, "finance:read")
	r.ServeHTTP(rec2, req2)
	assertStatus(t, rec2, http.StatusServiceUnavailable)
}

// --- HandleGetIncomingInvoice ---

func TestHandleGetIncomingInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetIncomingInvoice)
}

func TestHandleGetIncomingInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/incoming-invoices/x", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetIncomingInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetIncomingInvoice_RPCFails(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/incoming-invoices/x", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetIncomingInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateIncomingInvoiceStatus ---

func TestHandleUpdateIncomingInvoiceStatus_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateIncomingInvoiceStatus)
}

func TestHandleUpdateIncomingInvoiceStatus_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/finance/incoming-invoices/x/status", jsonBody(t, map[string]interface{}{
		"status": "reviewed",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateIncomingInvoiceStatus(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateIncomingInvoiceStatus_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/finance/incoming-invoices/x/status", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateIncomingInvoiceStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleUpdateIncomingInvoiceStatus_InvalidStatus locks down the `oneof=reviewed
// booked rejected` constraint: an unrelated status word must not silently reach the RPC.
func TestHandleUpdateIncomingInvoiceStatus_InvalidStatus(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/finance/incoming-invoices/x/status", jsonBody(t, map[string]interface{}{
		"status": "shipped",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateIncomingInvoiceStatus(rec, req)
	assertValidationError(t, rec, "status")
}

func TestHandleUpdateIncomingInvoiceStatus_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/finance/incoming-invoices/x/status", jsonBody(t, map[string]interface{}{
		"status": "booked",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateIncomingInvoiceStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateIncomingInvoiceStatus_PermissionGuard(t *testing.T) {
	r := bizRouter(emptyRegistry())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/finance/incoming-invoices/550e8400-e29b-41d4-a716-446655440000/status", jsonBody(t, map[string]interface{}{
		"status": "reviewed",
	}))
	req = withPermissions(req, "finance:read")
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("PATCH", "/api/v1/finance/incoming-invoices/550e8400-e29b-41d4-a716-446655440000/status", jsonBody(t, map[string]interface{}{
		"status": "reviewed",
	}))
	req2 = withPermissions(req2, "finance:write")
	r.ServeHTTP(rec2, req2)
	assertStatus(t, rec2, http.StatusServiceUnavailable)
}
