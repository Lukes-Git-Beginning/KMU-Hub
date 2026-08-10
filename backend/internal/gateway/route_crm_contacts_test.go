package gateway

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Contacts: Create ---
// ServiceUnavailable, NoUserID, InvalidJSON, MissingFields, InvalidEmail and
// InvalidCompanyID already covered in route_crm_test.go — not duplicated here.

func TestHandleCreateContact_InvalidPhone(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts", jsonBody(t, map[string]interface{}{
		"first_name": "Max",
		"last_name":  "Muster",
		"phone":      "not-a-phone-number",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "phone")
}

func TestHandleCreateContact_InvalidTagID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts", jsonBody(t, map[string]interface{}{
		"first_name": "Max",
		"last_name":  "Muster",
		"tag_ids":    []string{"not-a-uuid"},
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "tag_ids[0]")
}

func TestHandleCreateContact_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts", jsonBody(t, map[string]interface{}{
		"first_name": "Max",
		"last_name":  "Muster",
		"email":      "max@example.com",
		"company_id": "550e8400-e29b-41d4-a716-446655440000",
		"tag_ids":    []string{"550e8400-e29b-41d4-a716-446655440000"},
		"custom_fields": []map[string]string{
			{"field_id": "550e8400-e29b-41d4-a716-446655440000", "value": "x"},
		},
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateContact(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Contacts: Get ---
// ServiceUnavailable and InvalidUUID already covered in route_crm_test.go.

func TestHandleGetContact_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetContact(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Contacts: List ---
// ServiceUnavailable already covered in route_crm_test.go.

// TestHandleListContacts_FilterCombinations exercises every query-string
// filter branch (company_id, search, sort_by, sort_desc, visibility,
// tag_ids, pagination) plus the empty combination. None of these can be
// observed reaching the proto request without a fake CRMServiceClient (no
// bufconn stub exists for this package, same boundary noted in every prior
// gateway coverage unit this run) — each case is proven to parse past every
// filter branch by reaching the RPC layer (503), not panicking.
func TestHandleListContacts_FilterCombinations(t *testing.T) {
	cases := []struct {
		name  string
		query string
	}{
		{"empty", ""},
		{"company_id", "?company_id=550e8400-e29b-41d4-a716-446655440000"},
		{"search", "?search=max"},
		{"sort", "?sort_by=name&sort_desc=true"},
		{"visibility", "?visibility=personal"},
		{"tag_ids", "?tag_ids=550e8400-e29b-41d4-a716-446655440000&tag_ids=650e8400-e29b-41d4-a716-446655440001"},
		{"pagination", "?page=2&page_size=50"},
		{"all_combined", "?company_id=550e8400-e29b-41d4-a716-446655440000&search=max&sort_by=name&sort_desc=true&visibility=shared&tag_ids=550e8400-e29b-41d4-a716-446655440000&page=2&page_size=50"},
	}
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/contacts"+tc.query, nil)
			req = withUserID(req, "user-123")
			routes.HandleListContacts(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// --- Contacts: Update ---

func TestHandleUpdateContact_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContact(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateContact_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateContact_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateContact_InvalidEmail(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"email": "not-an-email",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "email")
}

func TestHandleUpdateContact_InvalidPhone(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"phone": "not-a-phone-number",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "phone")
}

func TestHandleUpdateContact_InvalidCompanyID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"company_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "company_id")
}

func TestHandleUpdateContact_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"first_name": "NewName",
		"notes":      "updated",
		"custom_fields": []map[string]string{
			{"field_id": "550e8400-e29b-41d4-a716-446655440000", "value": "x"},
		},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContact(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Contacts: Delete ---
// InvalidUUID already covered in route_crm_test.go.

func TestHandleDeleteContact_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteContact(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteContact_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteContact(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Contacts: Tags ---

func TestHandleAddContactTags_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/tags", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddContactTags(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAddContactTags_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/bad/tags", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAddContactTags(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleAddContactTags_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/tags", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddContactTags(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleAddContactTags_InvalidTagID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/tags", jsonBody(t, map[string]interface{}{
		"tag_ids": []string{"not-a-uuid"},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddContactTags(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "tag_ids[0]")
}

func TestHandleAddContactTags_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/tags", jsonBody(t, map[string]interface{}{
		"tag_ids": []string{"650e8400-e29b-41d4-a716-446655440001"},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddContactTags(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRemoveContactTags_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/tags", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRemoveContactTags(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRemoveContactTags_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/contacts/bad/tags", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleRemoveContactTags(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleRemoveContactTags_InvalidTagID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/tags", jsonBody(t, map[string]interface{}{
		"tag_ids": []string{"not-a-uuid"},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRemoveContactTags(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "tag_ids[0]")
}

func TestHandleRemoveContactTags_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/tags", jsonBody(t, map[string]interface{}{
		"tag_ids": []string{"650e8400-e29b-41d4-a716-446655440001"},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRemoveContactTags(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Contacts: Import ---

// multipartBody builds a multipart/form-data request body from the given
// form fields and an optional file part, returning the body plus the
// Content-Type header (including the boundary) the caller must set.
func multipartBody(t *testing.T, fields map[string]string, fileField, fileName string, fileContent []byte) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("write field %q: %v", k, err)
		}
	}
	if fileField != "" {
		fw, err := w.CreateFormFile(fileField, fileName)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := fw.Write(fileContent); err != nil {
			t.Fatalf("write file content: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return buf, w.FormDataContentType()
}

func TestHandleImportContactsCSV_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/csv", jsonBody(t, map[string]interface{}{}))
	routes.HandleImportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleImportContactsCSV_InvalidMultipartForm(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/csv", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleImportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid multipart form")
}

// TestHandleImportContactsCSV_NoMappingNoFile locks in that a well-formed
// multipart request carrying neither a "file" part nor any "map_"-prefixed
// mapping field is rejected for the missing file, not silently accepted
// with an empty mapping.
func TestHandleImportContactsCSV_NoMappingNoFile(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	body, contentType := multipartBody(t, map[string]string{"visibility": "shared"}, "", "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/csv", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleImportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "file is required")
}

// TestHandleImportContactsCSV_FieldMappingContract locks in the "map_<column>"
// form-field contract: each field prefixed "map_" is meant to become one
// entry of the field mapping sent to the CRM service, keyed by the suffix
// after the prefix, while unrelated fields (visibility, merge_by_email) stay
// out of it. The outgoing proto request itself cannot be observed without a
// fake CRMServiceClient (no bufconn stub exists for this package, same
// boundary noted in every prior gateway coverage unit this run) — this proves
// the parsing loop over r.MultipartForm.Value runs to completion without a
// panic for a realistic multi-field payload and still reaches the RPC layer
// (503).
func TestHandleImportContactsCSV_FieldMappingContract(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	body, contentType := multipartBody(t, map[string]string{
		"map_first_name": "Vorname",
		"map_last_name":  "Nachname",
		"map_email":      "E-Mail",
		"visibility":     "personal",
		"merge_by_email": "true",
	}, "file", "contacts.csv", []byte("Vorname,Nachname,E-Mail\nMax,Muster,max@example.com\n"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/csv", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleImportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleImportContactsVCard_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/vcard", jsonBody(t, map[string]interface{}{}))
	routes.HandleImportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleImportContactsVCard_InvalidMultipartForm(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/vcard", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleImportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid multipart form")
}

func TestHandleImportContactsVCard_MissingFile(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	body, contentType := multipartBody(t, map[string]string{}, "", "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/vcard", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleImportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "file is required")
}

func TestHandleImportContactsVCard_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	body, contentType := multipartBody(t, map[string]string{"merge_by_email": "true"}, "file", "contacts.vcf", []byte("BEGIN:VCARD\nEND:VCARD\n"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/vcard", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleImportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleImportContactsXLSX_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/xlsx", jsonBody(t, map[string]interface{}{}))
	routes.HandleImportContactsXLSX(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleImportContactsXLSX_InvalidMultipartForm(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/xlsx", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleImportContactsXLSX(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid multipart form")
}

func TestHandleImportContactsXLSX_MissingFile(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	body, contentType := multipartBody(t, map[string]string{}, "", "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/xlsx", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleImportContactsXLSX(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "file is required")
}

// TestHandleImportContactsXLSX_FieldMappingContract mirrors
// TestHandleImportContactsCSV_FieldMappingContract — HandleImportContactsXLSX
// duplicates the identical map_-prefix parsing loop, so the same contract is
// pinned here too.
func TestHandleImportContactsXLSX_FieldMappingContract(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	body, contentType := multipartBody(t, map[string]string{
		"map_first_name": "Vorname",
		"map_email":      "E-Mail",
	}, "file", "contacts.xlsx", []byte("fake-xlsx-bytes"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/xlsx", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleImportContactsXLSX(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandlePreviewImportCSV_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/preview", jsonBody(t, map[string]interface{}{}))
	routes.HandlePreviewImportCSV(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandlePreviewImportCSV_InvalidMultipartForm(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/preview", jsonBody(t, map[string]interface{}{}))
	routes.HandlePreviewImportCSV(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid multipart form")
}

func TestHandlePreviewImportCSV_MissingFile(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	body, contentType := multipartBody(t, map[string]string{}, "", "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/preview", body)
	req.Header.Set("Content-Type", contentType)
	routes.HandlePreviewImportCSV(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "file is required")
}

func TestHandlePreviewImportCSV_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	body, contentType := multipartBody(t, map[string]string{}, "file", "contacts.csv", []byte("first_name\nMax\n"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/import/preview", body)
	req.Header.Set("Content-Type", contentType)
	routes.HandlePreviewImportCSV(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Contacts: Export ---

func TestHandleExportContactsCSV_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/export/csv", jsonBody(t, map[string]interface{}{}))
	routes.HandleExportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportContactsCSV_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/export/csv", invalidJSON())
	routes.HandleExportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleExportContactsCSV_InvalidContactID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/export/csv", jsonBody(t, map[string]interface{}{
		"contact_ids": []string{"not-a-uuid"},
	}))
	routes.HandleExportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "contact_ids[0]")
}

func TestHandleExportContactsCSV_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/export/csv", jsonBody(t, map[string]interface{}{
		"contact_ids": []string{"550e8400-e29b-41d4-a716-446655440000"},
		"fields":      []string{"first_name", "email"},
	}))
	req = withUserID(req, "user-123")
	routes.HandleExportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportContactsVCard_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/export/vcard", jsonBody(t, map[string]interface{}{}))
	routes.HandleExportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportContactsVCard_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/export/vcard", invalidJSON())
	routes.HandleExportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleExportContactsVCard_InvalidContactID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/export/vcard", jsonBody(t, map[string]interface{}{
		"contact_ids": []string{"not-a-uuid"},
	}))
	routes.HandleExportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "contact_ids[0]")
}

func TestHandleExportContactsVCard_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts/export/vcard", jsonBody(t, map[string]interface{}{
		"contact_ids": []string{"550e8400-e29b-41d4-a716-446655440000"},
	}))
	req = withUserID(req, "user-123")
	routes.HandleExportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Contacts: Visibility ---

func TestHandleUpdateContactVisibility_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/visibility", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContactVisibility(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateContactVisibility_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/bad/visibility", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateContactVisibility(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateContactVisibility_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/visibility", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContactVisibility(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateContactVisibility_MissingVisibility(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/visibility", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContactVisibility(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "visibility")
}

func TestHandleUpdateContactVisibility_InvalidVisibilityValue(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/visibility", jsonBody(t, map[string]interface{}{
		"visibility": "public",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateContactVisibility(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "visibility")
}

func TestHandleUpdateContactVisibility_ValidRequestReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/contacts/550e8400-e29b-41d4-a716-446655440000/visibility", jsonBody(t, map[string]interface{}{
		"visibility": "personal",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withUserID(req, "user-123")
	routes.HandleUpdateContactVisibility(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
