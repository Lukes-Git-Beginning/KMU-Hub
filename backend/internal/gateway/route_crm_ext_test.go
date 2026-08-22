package gateway

// route_crm_ext_test.go covers route_crm_ext.go (unit
// cov-gateway-crm-ext-duplicates-merge-gdpr), which had no test file. It is
// the largest untested CRM route file: duplicate detection, merge, contact
// timeline, the deletion-impact preview, consent management, and the two GDPR
// deletion-request routes (request + admin-only process). Every handler is a
// thin pass-through to the CRM gRPC client (no tenant check of its own -- the
// tenant is threaded through context and enforced server-side), so most of
// this file proves the pre-RPC branches: invalid JSON, validation, and
// service-unavailable. Two done_when items need more than that:
//
//   - The four-eyes separation between REQUESTING and PROCESSING a deletion
//     is proven at the router-registration level (TestGDPRDeletionRoutes_
//     RequestAndProcessRequireDifferentGuards below), because the split lives
//     in route_crm.go's/registerGDPRRoutes' `r.With(...)` wiring, not inside
//     either handler body: HandleRequestDeletion sits under
//     `/api/v1/contacts/{id}/gdpr/deletion-request` guarded by
//     RequirePermissionAny(contacts:write, crm:contact:create) (route_crm.go
//     registers /contacts with contactCreate... actually the ext route itself
//     specifies "contacts","write" directly, see registerContactExtRoutes),
//     while HandleProcessDeletion sits under the separate top-level
//     `/api/v1/gdpr/deletion-requests/{id}/process` guarded by
//     middleware.RequireRole("admin") -- a different mechanism entirely, the
//     same pattern already proven for automation stats in
//     route_capability_guard_test.go. No fix-unit is needed: the separation
//     already exists, this only pins it down with a test.
//   - Merge with colliding unique links (a tag or custom-field value the
//     duplicate and primary both carry) is proven against the real repository
//     in internal/crm/contact/postgres_repository_db_test.go
//     TestRepository_MergeInto_ReassignsRelationsMergesTagsAndCustomFieldsThenSoftDeletes
//     (contact_tags uses ON CONFLICT DO NOTHING, contact_custom_field_values
//     uses ON CONFLICT (contact_id, field_id) DO NOTHING) and the mirror test
//     in internal/crm/company/postgres_repository_db_test.go. Foreign-tenant
//     merges, already-merged contacts/companies and self-merge are proven at
//     the service layer (service_test.go TestService_MergeContacts_* /
//     TestService_MergeCompanies_* and TestService_CrossTenant_MergeContacts_
//     NilTenantRejected / ...MergeCompanies...). None of that is re-pinned
//     here as a gateway-level fake.
//
// Cross-tenant reads (deletion preview, duplicates) are likewise proven where
// the tenant scoping actually happens: internal/crm/contact/service_test.go
// TestService_PreviewDeletion_NotFoundForUnknownOrForeignContact and
// FindDuplicates' own GetByID(tenantID) check. Consent management is proven
// end to end (including tenant isolation) in internal/crm/consent/rls_test.go
// and tenant_write_test.go.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

const (
	extContactTestID = "55555555-5555-5555-5555-555555555555"
	extCompanyTestID = "66666666-6666-6666-6666-666666666666"
	extRequestTestID = "77777777-7777-7777-7777-777777777777"
	extConsentType   = "marketing"
)

// ============================================================================
// ServiceUnavailable — every handler resolves the CRM client first.
// ============================================================================

func TestCRMExtRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewCRMExtRoutes(emptyRegistry())

	idReq := func(method, path, param, value string) *http.Request {
		r := httptest.NewRequest(method, path, nil)
		return withChiURLParam(r, param, value)
	}

	handlers := map[string]http.HandlerFunc{
		"HandleFindContactDuplicates": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleFindContactDuplicates(w, idReq(http.MethodGet, "/", "id", extContactTestID))
		},
		"HandleMergeContacts": routes.HandleMergeContacts,
		"HandleFindCompanyDuplicates": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleFindCompanyDuplicates(w, idReq(http.MethodGet, "/", "id", extCompanyTestID))
		},
		"HandleMergeCompanies": routes.HandleMergeCompanies,
		"HandleContactTimeline": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleContactTimeline(w, idReq(http.MethodGet, "/", "id", extContactTestID))
		},
		"HandleContactDeletionPreview": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleContactDeletionPreview(w, idReq(http.MethodGet, "/", "id", extContactTestID))
		},
		"HandleGetConsents": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleGetConsents(w, idReq(http.MethodGet, "/", "id", extContactTestID))
		},
		"HandleGrantConsent": routes.HandleGrantConsent,
		"HandleRevokeConsent": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleRevokeConsent(w, idReq(http.MethodDelete, "/", "id", extContactTestID))
		},
		"HandleGetConsentHistory": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleGetConsentHistory(w, idReq(http.MethodGet, "/", "id", extContactTestID))
		},
		"HandleRequestDeletion": routes.HandleRequestDeletion,
		"HandleProcessDeletion": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleProcessDeletion(w, idReq(http.MethodPost, "/", "id", extRequestTestID))
		},
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// HandleFindContactDuplicates / HandleFindCompanyDuplicates
// ============================================================================

func TestHandleFindContactDuplicates_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+extContactTestID+"/duplicates", nil)
	req = withChiURLParam(req, "id", extContactTestID)
	routes.HandleFindContactDuplicates(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleFindCompanyDuplicates_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/"+extCompanyTestID+"/duplicates", nil)
	req = withChiURLParam(req, "id", extCompanyTestID)
	routes.HandleFindCompanyDuplicates(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleMergeContacts
// ============================================================================

func TestHandleMergeContacts_InvalidJSON(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/merge", invalidJSON())
	routes.HandleMergeContacts(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleMergeContacts_MissingIDs(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/merge", jsonBody(t, map[string]interface{}{}))
	routes.HandleMergeContacts(rec, req)
	assertValidationError(t, rec, "primary_id")
}

func TestHandleMergeContacts_InvalidUUID(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/merge", jsonBody(t, map[string]interface{}{
		"primary_id":   "not-a-uuid",
		"duplicate_id": extContactTestID,
	}))
	routes.HandleMergeContacts(rec, req)
	assertValidationError(t, rec, "primary_id")
}

func TestHandleMergeContacts_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/merge", jsonBody(t, map[string]interface{}{
		"primary_id":   extContactTestID,
		"duplicate_id": "88888888-8888-8888-8888-888888888888",
	}))
	routes.HandleMergeContacts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleMergeCompanies
// ============================================================================

func TestHandleMergeCompanies_InvalidJSON(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/merge", invalidJSON())
	routes.HandleMergeCompanies(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleMergeCompanies_InvalidUUID(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/merge", jsonBody(t, map[string]interface{}{
		"primary_id":   extCompanyTestID,
		"duplicate_id": "not-a-uuid",
	}))
	routes.HandleMergeCompanies(rec, req)
	assertValidationError(t, rec, "duplicate_id")
}

func TestHandleMergeCompanies_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/merge", jsonBody(t, map[string]interface{}{
		"primary_id":   extCompanyTestID,
		"duplicate_id": "99999999-9999-9999-9999-999999999999",
	}))
	routes.HandleMergeCompanies(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleContactTimeline — offset/limit are optional and clamp silently
// instead of rejecting: values are read into page/pageSize, and out-of-range
// or unparsable ones just fall back to the defaults (page 1, size 20).
// ============================================================================

func TestHandleContactTimeline_DefaultsWithoutQuery(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+extContactTestID+"/timeline", nil)
	req = withChiURLParam(req, "id", extContactTestID)
	routes.HandleContactTimeline(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleContactTimeline_WithValidOffsetAndLimit(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+extContactTestID+"/timeline?offset=2&limit=50", nil)
	req = withChiURLParam(req, "id", extContactTestID)
	routes.HandleContactTimeline(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleContactTimeline_OutOfRangeLimitFallsBackToDefault(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+extContactTestID+"/timeline?offset=-1&limit=500", nil)
	req = withChiURLParam(req, "id", extContactTestID)
	routes.HandleContactTimeline(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleContactDeletionPreview — pure read, proves nothing else here (see
// file header for where tenant scoping and 404-vs-200 are proven).
// ============================================================================

func TestHandleContactDeletionPreview_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+extContactTestID+"/deletion-preview", nil)
	req = withChiURLParam(req, "id", extContactTestID)
	routes.HandleContactDeletionPreview(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetConsents / HandleGetConsentHistory
// ============================================================================

func TestHandleGetConsents_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+extContactTestID+"/consents", nil)
	req = withChiURLParam(req, "id", extContactTestID)
	routes.HandleGetConsents(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetConsentHistory_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+extContactTestID+"/consents/"+extConsentType+"/history", nil)
	req = withChiURLParam(req, "id", extContactTestID)
	req = withChiURLParam(req, "type", extConsentType)
	routes.HandleGetConsentHistory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGrantConsent — consent_type, source and legal_basis are required.
// ============================================================================

func TestHandleGrantConsent_InvalidJSON(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+extContactTestID+"/consents", invalidJSON())
	req = withChiURLParam(req, "id", extContactTestID)
	routes.HandleGrantConsent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleGrantConsent_MissingRequiredFields(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+extContactTestID+"/consents", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", extContactTestID)
	routes.HandleGrantConsent(rec, req)
	assertValidationError(t, rec, "consent_type")
}

func TestHandleGrantConsent_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+extContactTestID+"/consents", jsonBody(t, map[string]interface{}{
		"consent_type": extConsentType,
		"source":       "signup_form",
		"legal_basis":  "consent",
	}))
	req = withChiURLParam(req, "id", extContactTestID)
	req = withUserID(req, "user-123")
	routes.HandleGrantConsent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleRevokeConsent — "notes" carries no validate tag, any JSON body passes.
// ============================================================================

func TestHandleRevokeConsent_InvalidJSON(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/"+extContactTestID+"/consents/"+extConsentType, invalidJSON())
	req = withChiURLParam(req, "id", extContactTestID)
	req = withChiURLParam(req, "type", extConsentType)
	routes.HandleRevokeConsent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleRevokeConsent_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/"+extContactTestID+"/consents/"+extConsentType, jsonBody(t, map[string]interface{}{
		"notes": "withdrawn by contact",
	}))
	req = withChiURLParam(req, "id", extContactTestID)
	req = withChiURLParam(req, "type", extConsentType)
	req = withUserID(req, "user-123")
	routes.HandleRevokeConsent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleRequestDeletion — "reason" carries no validate tag either.
// ============================================================================

func TestHandleRequestDeletion_InvalidJSON(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+extContactTestID+"/gdpr/deletion-request", invalidJSON())
	req = withChiURLParam(req, "id", extContactTestID)
	routes.HandleRequestDeletion(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleRequestDeletion_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+extContactTestID+"/gdpr/deletion-request", jsonBody(t, map[string]interface{}{
		"reason": "customer request",
	}))
	req = withChiURLParam(req, "id", extContactTestID)
	req = withUserID(req, "user-123")
	routes.HandleRequestDeletion(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleProcessDeletion
// ============================================================================

func TestHandleProcessDeletion_ReachesRPC(t *testing.T) {
	routes := NewCRMExtRoutes(registryWithService("crm"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gdpr/deletion-requests/"+extRequestTestID+"/process", nil)
	req = withChiURLParam(req, "id", extRequestTestID)
	routes.HandleProcessDeletion(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Four-eyes separation between requesting and processing a GDPR deletion.
//
// This is the done_when item that matters most for this unit: a normal
// contacts:write user may FILE a deletion request, but only an admin-role
// token may EXECUTE it. The split is wired through two different guard
// mechanisms (RequirePermission vs. RequireRole) at route registration, not
// inside either handler -- so it has to be proven by mounting the real
// router, exactly like the automation-stats role gate in
// route_capability_guard_test.go.
// ============================================================================

func TestGDPRDeletionRoutes_RequestAndProcessRequireDifferentGuards(t *testing.T) {
	registry := emptyRegistry()
	router := chi.NewRouter()
	NewCRMRoutes(registry, NewCRMExtRoutes(registry)).RegisterRoutes(router, guardTestAuth)

	const (
		allowed = http.StatusServiceUnavailable // guard passed, empty registry stops it further down
		denied  = http.StatusForbidden
	)

	tests := []struct {
		name       string
		method     string
		path       string
		perms      []string
		roles      []string
		wantStatus int
	}{
		{
			name:       "request deletion with contacts:write is allowed",
			method:     http.MethodPost,
			path:       "/api/v1/contacts/" + extContactTestID + "/gdpr/deletion-request",
			perms:      []string{"contacts:write"},
			wantStatus: allowed,
		},
		{
			name:       "request deletion without contacts:write is denied",
			method:     http.MethodPost,
			path:       "/api/v1/contacts/" + extContactTestID + "/gdpr/deletion-request",
			perms:      []string{"contacts:read"},
			wantStatus: denied,
		},
		{
			name:       "process deletion with contacts:write alone is denied -- permission does not grant the admin-role gate",
			method:     http.MethodPost,
			path:       "/api/v1/gdpr/deletion-requests/" + extRequestTestID + "/process",
			perms:      []string{"contacts:write"},
			wantStatus: denied,
		},
		{
			name:       "process deletion without any role is denied",
			method:     http.MethodPost,
			path:       "/api/v1/gdpr/deletion-requests/" + extRequestTestID + "/process",
			wantStatus: denied,
		},
		{
			name:       "process deletion with admin role is allowed",
			method:     http.MethodPost,
			path:       "/api/v1/gdpr/deletion-requests/" + extRequestTestID + "/process",
			roles:      []string{"admin"},
			wantStatus: allowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req = withAuth(req, "user-123", testTenantID)
			req = withPermissions(req, tt.perms...)
			req = withRoles(req, tt.roles...)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
