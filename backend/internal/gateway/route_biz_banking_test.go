package gateway

// route_biz_banking.go had no dedicated test file. Its three handlers cover
// the bank statement import (CAMT.053/MT940) and read paths — no IBAN
// formatting or masking happens here (that lives in route_biz_bank_accounts.go,
// already covered by route_biz_bank_accounts_test.go): a statement's
// account_iban/counterparty_iban pass straight through hrMarshalProto/
// hrMarshalSlice, unformatted, matching the plain string the OpenAPI schema
// documents for BankStatement.account_iban. There is nothing to test in
// isolation on this file for that concern.

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- HandleImportBankStatement ---

func TestHandleImportBankStatement_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleImportBankStatement)
}

func TestHandleImportBankStatement_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, nil, "file", "statement.xml", []byte("<Document/>"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-statements/import", body)
	req.Header.Set("Content-Type", contentType)
	routes.HandleImportBankStatement(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleImportBankStatement_InvalidMultipartForm(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-statements/import", bytes.NewReader([]byte("not multipart")))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	req = withTenantID(req, testTenantID)
	routes.HandleImportBankStatement(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid multipart form")
}

func TestHandleImportBankStatement_MissingFile(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, nil, "", "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-statements/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleImportBankStatement(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "file field is required")
}

func TestHandleImportBankStatement_EmptyFile(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, nil, "file", "statement.xml", []byte{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-statements/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleImportBankStatement(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "uploaded file is empty")
}

func TestHandleImportBankStatement_ExceedsSizeLimit(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	oversized := bytes.Repeat([]byte{'A'}, maxBankStatementUploadBytes+1)
	body, contentType := multipartBody(t, nil, "file", "statement.xml", oversized)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-statements/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleImportBankStatement(rec, req)
	assertStatus(t, rec, http.StatusRequestEntityTooLarge)
	assertErrorContains(t, rec, "exceeds 10 MiB limit")
}

func TestHandleImportBankStatement_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	body, contentType := multipartBody(t, nil, "file", "statement.xml", []byte("<Document/>"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-statements/import", body)
	req.Header.Set("Content-Type", contentType)
	req = withTenantID(req, testTenantID)
	routes.HandleImportBankStatement(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListBankStatements ---

func TestHandleListBankStatements_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListBankStatements)
}

func TestHandleListBankStatements_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-statements", nil)
	routes.HandleListBankStatements(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListBankStatements_DefaultPagination_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-statements", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListBankStatements(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListBankStatements_ExplicitPagination_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-statements?page=2&page_size=10", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListBankStatements(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetBankStatement ---

func TestHandleGetBankStatement_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetBankStatement)
}

func TestHandleGetBankStatement_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-statements/11111111-1111-1111-1111-111111111111", nil)
	routes.HandleGetBankStatement(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetBankStatement_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-statements/11111111-1111-1111-1111-111111111111", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleGetBankStatement(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
