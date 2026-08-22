package gateway

// route_email_contact_links_import_export_test.go covers the third and last
// 0%-covered slice of route_email.go (unit
// cov-gateway-email-contact-linking-import-export, after
// cov-gateway-email-messages-folders-sync and
// cov-gateway-email-signatures-templates): message-contact linking
// (HandleGetEmailContactLinks, HandleLinkEmailToContact,
// HandleUnlinkEmailFromContact, HandleGetContactEmails) and contact
// import/export (HandleImportContactsCSV, HandleImportContactsVCard,
// HandleExportContactsCSV, HandleExportContactsVCard) -- verified against a
// fresh `go tool cover -func` on route_email.go before writing this file.
//
// Tenant isolation for the link repository (email_contact_links) is already
// proven at the repository layer by
// internal/email/contactlink/tenant_isolation_test.go
// (TestTenantIsolation_EmailContactLinks_DB); these gateway handlers are thin
// pass-throughs to the gRPC client and add nothing of their own to isolate.
//
// CSV formula injection: internal/email/contact/export_service.go's
// ExportCSV writes first_name/last_name/notes/position/company fields
// straight into CSV cells with no neutralization of a leading =, +, -, or @
// (verified by reading export_service.go: the field switch just appends the
// raw string to the row). A contact whose name or notes starts with one of
// those characters would open as a live formula in Excel/LibreOffice. This
// is a real finding, not a test fixture to pin -- writing a test that
// asserts the unescaped output would enshrine the vulnerability as expected
// behavior. Filed as fix-email-contacts-csv-export-formula-injection at the
// end of BACKLOG.yml instead; see JOURNAL.md for this iteration.
//
// The cross-tenant export proof (done_when: "Ein Export mit zwei geseedeten
// Tenants...") is a DB-level guarantee that belongs where ExportContactsCSV
// actually touches the database -- internal/server/email_grpc_export_test.go
// (TestEmailExportContactsCSV_ExcludesOtherTenantContact) -- not here; the
// gateway handler below never sees a tenant ID, it only decodes JSON and
// forwards to the RPC client.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// Contact Link Handlers
// ============================================================================

func TestEmailHandleGetEmailContactLinks_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleGetEmailContactLinks)
}

func TestEmailHandleGetEmailContactLinks_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/messages/msg-1/contact-links", nil)
	req = withChiURLParam(req, "messageId", "msg-1")
	routes.HandleGetEmailContactLinks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleLinkEmailToContact_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleLinkEmailToContact)
}

func TestEmailHandleLinkEmailToContact_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contact-links", invalidJSON())
	routes.HandleLinkEmailToContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleLinkEmailToContact_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contact-links", jsonBody(t, map[string]interface{}{
		"message_id": "msg-1",
		"contact_id": "contact-1",
	}))
	routes.HandleLinkEmailToContact(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleUnlinkEmailFromContact_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleUnlinkEmailFromContact)
}

func TestEmailHandleUnlinkEmailFromContact_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contact-links/unlink", invalidJSON())
	routes.HandleUnlinkEmailFromContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleUnlinkEmailFromContact_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contact-links/unlink", jsonBody(t, map[string]interface{}{
		"message_id": "msg-1",
		"contact_id": "contact-1",
	}))
	routes.HandleUnlinkEmailFromContact(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleGetContactEmails_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleGetContactEmails)
}

func TestEmailHandleGetContactEmails_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/contacts/contact-1/emails?page=2&per_page=10", nil)
	req = withChiURLParam(req, "contactId", "contact-1")
	routes.HandleGetContactEmails(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Contact Import/Export Handlers
// ============================================================================

func TestEmailHandleImportContactsCSV_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleImportContactsCSV)
}

func TestEmailHandleImportContactsCSV_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contacts/import/csv", invalidJSON())
	routes.HandleImportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleImportContactsCSV_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contacts/import/csv", jsonBody(t, map[string]interface{}{
		"file_content": "RS1NYWlsCg==",
		"visibility":   "shared",
	}))
	routes.HandleImportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleImportContactsVCard_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleImportContactsVCard)
}

func TestEmailHandleImportContactsVCard_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contacts/import/vcard", invalidJSON())
	routes.HandleImportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleImportContactsVCard_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contacts/import/vcard", jsonBody(t, map[string]interface{}{
		"file_content": "QkVHSU46VkNBUkQK",
		"visibility":   "personal",
	}))
	routes.HandleImportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleExportContactsCSV_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleExportContactsCSV)
}

func TestEmailHandleExportContactsCSV_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contacts/export/csv", invalidJSON())
	routes.HandleExportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleExportContactsCSV_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contacts/export/csv", jsonBody(t, map[string]interface{}{
		"contact_ids": []string{"contact-1", "contact-2"},
	}))
	routes.HandleExportContactsCSV(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleExportContactsVCard_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleExportContactsVCard)
}

func TestEmailHandleExportContactsVCard_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contacts/export/vcard", invalidJSON())
	routes.HandleExportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleExportContactsVCard_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/contacts/export/vcard", jsonBody(t, map[string]interface{}{
		"contact_ids": []string{"contact-1"},
	}))
	routes.HandleExportContactsVCard(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
