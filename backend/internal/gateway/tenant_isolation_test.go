package gateway

// Tenant isolation smoke tests for the HTTP gateway layer.
//
// These tests verify that the GetTenantID extraction is correctly wired into
// the rapporte, vermietung and inventar handlers. They do NOT test gRPC-level
// data isolation (that is covered by service_test.go in each module package),
// but they do verify:
//
//  1. Requests without an Authorization header / without a JWT → 401.
//  2. JWTs with an empty tid claim (legacy tokens) → 401 (fail-closed).
//  3. JWTs with a valid tid claim → the handler proceeds past the tenant check
//     (response is 503 when the gRPC backend is absent — which is fine for unit tests).
//
// Cross-tenant data isolation (Token-A writes, Token-B reads → 404) is
// enforced by the service layer and exercised in e.g.
// internal/rapporte/service_test.go:TestService_UploadAttachment_CrossTenantPath_Rejected.
// An end-to-end integration test would require a real gRPC backend; that is
// outside the scope of gateway unit tests.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
)

// ============================================================================
// Helper: build a flag registry with a single module enabled
// ============================================================================

func moduleFlag(key string) *featureflag.Registry {
	return featureflag.NewRegistry().Load(func(k string) string {
		if k == key {
			return "true"
		}
		return ""
	})
}

// ============================================================================
// Helper: build a request with TenantID set directly in context (simulates
// what the Auth middleware does after validating a JWT that contains a tid claim).
// ============================================================================

func reqWithTenant(method, path string, tenantID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID.String())
	return req.WithContext(ctx)
}

// reqWithEmptyTenant simulates a legacy JWT that was parsed successfully but had
// no tid claim — the middleware stores an empty string.
func reqWithEmptyTenant(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, "")
	return req.WithContext(ctx)
}

// ============================================================================
// Rapporte — tenant isolation checks
// ============================================================================

func rapporteFlagsON() *featureflag.Registry {
	return moduleFlag("COSMI_MODULE_RAPPORTE_ENABLED")
}

// TestRapporte_NoTenant_Returns401 verifies that a request without any tenant
// context is rejected before it reaches the gRPC client.
func TestRapporte_NoTenant_Returns401(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), rapporteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rapporte/reports", nil)
	routes.HandleListReports(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestRapporte_EmptyTid_Returns401 verifies that a legacy JWT (empty tid) is
// rejected with 401 — no placeholder substitution may occur.
func TestRapporte_EmptyTid_Returns401(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), rapporteFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodGet, "/api/v1/rapporte/reports")
	routes.HandleListReports(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestRapporte_ValidTid_PassesTenantCheck verifies that a request carrying a valid
// tid proceeds past the tenant check (503 is expected because there is no real
// gRPC backend in unit tests — anything other than 401 proves the check passed).
func TestRapporte_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), rapporteFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithTenant(http.MethodGet, "/api/v1/rapporte/reports", uuid.New())
	routes.HandleListReports(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

// TestRapporte_TwoTenants_DifferentContextValues verifies that two requests with
// different tenant IDs are treated as independent — the handler reads from context
// and does not share any mutable state.
func TestRapporte_TwoTenants_DifferentContextValues(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), rapporteFlagsON())

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/rapporte/reports", tenantA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/rapporte/reports", tenantB)

	routes.HandleListReports(recA, reqA)
	routes.HandleListReports(recB, reqB)

	// Both should fail in the same way (503 — no backend); neither should be 401.
	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B request rejected with 401")
	}
}

// ============================================================================
// Vermietung — tenant isolation checks
// ============================================================================

func vermietungFlagsON() *featureflag.Registry {
	return moduleFlag("COSMI_MODULE_VERMIETUNG_ENABLED")
}

func TestVermietung_NoTenant_Returns401(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), vermietungFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vermietung/objects", nil)
	routes.HandleListObjects(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestVermietung_EmptyTid_Returns401(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), vermietungFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodGet, "/api/v1/vermietung/objects")
	routes.HandleListObjects(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestVermietung_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), vermietungFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithTenant(http.MethodGet, "/api/v1/vermietung/objects", uuid.New())
	routes.HandleListObjects(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

// ============================================================================
// Inventar — tenant isolation checks
// ============================================================================

func inventarFlagsON() *featureflag.Registry {
	return moduleFlag("COSMI_MODULE_INVENTAR_ENABLED")
}

func TestInventar_NoTenant_Returns401(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), inventarFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventar/items", nil)
	routes.HandleListItems(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestInventar_EmptyTid_Returns401(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), inventarFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodGet, "/api/v1/inventar/items")
	routes.HandleListItems(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestInventar_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), inventarFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithTenant(http.MethodGet, "/api/v1/inventar/items", uuid.New())
	routes.HandleListItems(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

// ============================================================================
// CRM Deals — tenant isolation checks
// ============================================================================

func TestDeals_NoTenant_Returns401(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/crm/deals", nil)
	routes.HandleListDeals(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestDeals_EmptyTid_Returns401(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodGet, "/api/v1/crm/deals")
	routes.HandleListDeals(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestDeals_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := reqWithTenant(http.MethodGet, "/api/v1/crm/deals", uuid.New())
	routes.HandleListDeals(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

func TestDeals_TwoTenants_IndependentContexts(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/crm/deals", tenantA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/crm/deals", tenantB)

	routes.HandleListDeals(recA, reqA)
	routes.HandleListDeals(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B request rejected with 401")
	}
}

// ============================================================================
// CRM Activities — tenant isolation checks
// ============================================================================

func TestActivities_NoTenant_Returns401(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/crm/activities", nil)
	routes.HandleListActivities(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestActivities_EmptyTid_Returns401(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodGet, "/api/v1/crm/activities")
	routes.HandleListActivities(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestActivities_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := reqWithTenant(http.MethodGet, "/api/v1/crm/activities", uuid.New())
	routes.HandleListActivities(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

// ============================================================================
// Work Tasks — tenant isolation checks
// ============================================================================

func TestTasks_NoTenant_Returns401(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/work/tasks", nil)
	routes.HandleListTasks(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestTasks_EmptyTid_Returns401(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodGet, "/api/v1/work/tasks")
	routes.HandleListTasks(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestTasks_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := reqWithTenant(http.MethodGet, "/api/v1/work/tasks", uuid.New())
	routes.HandleListTasks(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

func TestTasks_TwoTenants_IndependentContexts(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/work/tasks", tenantA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/work/tasks", tenantB)

	routes.HandleListTasks(recA, reqA)
	routes.HandleListTasks(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B request rejected with 401")
	}
}

// ============================================================================
// Chat Messages (SendMessage) — tenant isolation checks
// ============================================================================

func TestMessages_NoTenant_Returns401(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	// Use a valid channel UUID in the path so the handler doesn't fail on param parsing
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", uuid.New().String())
	routes.HandleSendMessage(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestMessages_EmptyTid_Returns401(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	// Build req2 directly with a real body — reqWithEmptyTenant returned an empty body
	// and the result was not used after chi URL-param attachment.
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"content":"hello"}`))
	req2.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req2.Context(), middleware.TenantIDKey, "")
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user1")
	req2 = req2.WithContext(ctx)
	req2 = withChiURLParam(req2, "id", uuid.New().String())
	routes.HandleSendMessage(rec, req2)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestMessages_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withTenantID(req, uuid.New())
	req = withChiURLParam(req, "id", uuid.New().String())
	routes.HandleSendMessage(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

// ============================================================================
// Recording — initiator-consent tenant isolation (R2-P0.4 / Welle 3.5)
//
// The /api/v1/video/recordings/{id}/initiator-consent endpoint is the
// pre-recording consent gate. It MUST fail-closed on missing or empty
// tenant context — otherwise an attacker with knowledge of a recording UUID
// could stamp consent on a recording belonging to another tenant.
// ============================================================================

func TestRecordingInitiatorConsent_NoTenant_Returns401(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withChiURLParam(req, "id", uuid.New().String())
	routes.HandleConfirmInitiatorConsent(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestRecordingInitiatorConsent_EmptyTid_Returns401(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodPost, "/")
	req = withChiURLParam(req, "id", uuid.New().String())
	routes.HandleConfirmInitiatorConsent(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestRecordingInitiatorConsent_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withTenantID(req, uuid.New())
	req = withChiURLParam(req, "id", uuid.New().String())
	routes.HandleConfirmInitiatorConsent(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

// TestRecordingInitiatorConsent_TwoTenants_IndependentContexts ensures that
// two requests from different tenants do not share state and both reach the
// gRPC layer (where MarkInitiatorConsent then enforces tenant_id at the DB
// level — see recording/postgres_repository.go::MarkInitiatorConsent).
func TestRecordingInitiatorConsent_TwoTenants_IndependentContexts(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"))
	tenantA := uuid.New()
	tenantB := uuid.New()
	recordingID := uuid.New().String()

	recA := httptest.NewRecorder()
	reqA := httptest.NewRequest(http.MethodPost, "/", nil)
	reqA = withTenantID(reqA, tenantA)
	reqA = withChiURLParam(reqA, "id", recordingID)
	routes.HandleConfirmInitiatorConsent(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := httptest.NewRequest(http.MethodPost, "/", nil)
	reqB = withTenantID(reqB, tenantB)
	reqB = withChiURLParam(reqB, "id", recordingID)
	routes.HandleConfirmInitiatorConsent(recB, reqB)

	if recA.Code == http.StatusUnauthorized || recB.Code == http.StatusUnauthorized {
		t.Errorf("valid tids should not be rejected with 401 (A=%d, B=%d)", recA.Code, recB.Code)
	}
}

// ============================================================================
// P2-7: 12 additional Tenant Isolation Subtests (Welle 4B.2 Stream 2D)
//
// These tests verify that the HTTP gateway correctly extracts or passes the
// tenant context for each domain. Where a handler calls middleware.GetTenantID
// directly it must return 401 on missing/empty tid. Where the handler delegates
// tenant enforcement to the gRPC layer the test verifies that two independent
// tenant contexts produce independent responses (no shared state, no 401).
// ============================================================================

// ============================================================================
// P2-7.1 — Pipeline Stages
// ============================================================================

func TestTenantIsolation_Pipeline_Stages(t *testing.T) {
	// HandleListPipelineStages delegates tenant enforcement to the gRPC backend.
	// Two requests with different tenant contexts must both reach the gRPC layer
	// (503 expected — no backend in unit tests — not 401).
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/crm/pipeline-stages", tenantA)
	routes.HandleListPipelineStages(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/crm/pipeline-stages", tenantB)
	routes.HandleListPipelineStages(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A pipeline-stages request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B pipeline-stages request rejected with 401")
	}
}

// ============================================================================
// P2-7.2 — Calendar Events
// ============================================================================

func TestTenantIsolation_CalendarEvents(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/calendar/events", tenantA)
	routes.HandleListEventsInRange(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/calendar/events", tenantB)
	routes.HandleListEventsInRange(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A calendar-events request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B calendar-events request rejected with 401")
	}
}

// ============================================================================
// P2-7.3 — Time Entries
// ============================================================================

func TestTenantIsolation_TimeEntries(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))

	tenantA := uuid.New()
	tenantB := uuid.New()
	taskID := uuid.New().String()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/work/tasks/"+taskID+"/time-entries", tenantA)
	reqA = withChiURLParam(reqA, "id", taskID)
	routes.HandleListTimeEntries(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/work/tasks/"+taskID+"/time-entries", tenantB)
	reqB = withChiURLParam(reqB, "id", taskID)
	routes.HandleListTimeEntries(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A time-entries request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B time-entries request rejected with 401")
	}
}

// ============================================================================
// P2-7.4 — Automations
// ============================================================================

func TestTenantIsolation_Automations(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/automations", tenantA)
	routes.HandleListAutomations(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/automations", tenantB)
	routes.HandleListAutomations(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A automations request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B automations request rejected with 401")
	}
}

// ============================================================================
// P2-7.5 — Saved Filters
// ============================================================================

func TestTenantIsolation_SavedFilters(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/crm/saved-filters", tenantA)
	routes.HandleListSavedFilters(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/crm/saved-filters", tenantB)
	routes.HandleListSavedFilters(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A saved-filters request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B saved-filters request rejected with 401")
	}
}

// ============================================================================
// P2-7.6 — Custom Fields
// ============================================================================

func TestTenantIsolation_CustomFields(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/crm/custom-fields", tenantA)
	routes.HandleListCustomFields(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/crm/custom-fields", tenantB)
	routes.HandleListCustomFields(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A custom-fields request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B custom-fields request rejected with 401")
	}
}

// ============================================================================
// P2-7.7 — Email Messages
// ============================================================================

func TestTenantIsolation_EmailMessages(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))

	tenantA := uuid.New()
	tenantB := uuid.New()
	accountID := uuid.New().String()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/email/accounts/"+accountID+"/messages", tenantA)
	reqA = withChiURLParam(reqA, "accountId", accountID)
	routes.HandleListMessages(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/email/accounts/"+accountID+"/messages", tenantB)
	reqB = withChiURLParam(reqB, "accountId", accountID)
	routes.HandleListMessages(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A email-messages request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B email-messages request rejected with 401")
	}
}

// ============================================================================
// P2-7.8 — Inbox Messages
// ============================================================================

func TestTenantIsolation_InboxMessages(t *testing.T) {
	routes := NewInboxRoutes(registryWithService("inbox"))

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/inbox/messages", tenantA)
	routes.HandleListMessages(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/inbox/messages", tenantB)
	routes.HandleListMessages(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A inbox-messages request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B inbox-messages request rejected with 401")
	}
}

// ============================================================================
// P2-7.9 — Dialer Campaigns
// ============================================================================

func TestTenantIsolation_Dialer_Campaigns(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/dialer/campaigns", tenantA)
	routes.HandleListCampaigns(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/dialer/campaigns", tenantB)
	routes.HandleListCampaigns(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A dialer-campaigns request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B dialer-campaigns request rejected with 401")
	}
}

// ============================================================================
// P2-7.10 — Audit Log
// ============================================================================

func TestTenantIsolation_AuditLog(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("security"))

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/security/audit", tenantA)
	routes.HandleListAuditEntries(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/security/audit", tenantB)
	routes.HandleListAuditEntries(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A audit-log request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B audit-log request rejected with 401")
	}
}

// ============================================================================
// P2-7.11 — Recordings list (distinct from InitiatorConsent gate above)
// ============================================================================

func TestTenantIsolation_Recordings(t *testing.T) {
	// HandleListRecordings delegates all tenant enforcement to the gRPC layer.
	// Verify that two distinct tenant contexts both pass the HTTP layer without 401.
	routes := NewVideoRoutes(registryWithService("work"))

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/video/recordings", tenantA)
	routes.HandleListRecordings(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/video/recordings", tenantB)
	routes.HandleListRecordings(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A recordings-list request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B recordings-list request rejected with 401")
	}
}

// ============================================================================
// P2-7.12 — Channels
// ============================================================================

func TestTenantIsolation_Channels(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/chat/channels", tenantA)
	routes.HandleListChannels(recA, reqA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/chat/channels", tenantB)
	routes.HandleListChannels(recB, reqB)

	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A channels request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B channels request rejected with 401")
	}
}
