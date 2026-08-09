package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// activityTestID is a stable, valid UUID used wherever a handler needs a
// syntactically valid id/tenant path param but the value itself is not under
// test.
const activityTestID = "11111111-1111-1111-1111-111111111111"

// ============================================================================
// ServiceUnavailable — every handler in this file except the two tag-mutation
// stubs (HandleAddActivityTags/HandleRemoveActivityTags, which never reach a
// gRPC client — see their own tests below) resolves the CRM client first.
// ============================================================================

func TestCRMActivitiesRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)

	handlers := map[string]http.HandlerFunc{
		"HandleCreateActivity":      routes.HandleCreateActivity,
		"HandleGetActivity":         routes.HandleGetActivity,
		"HandleListActivities":      routes.HandleListActivities,
		"HandleUpdateActivity":      routes.HandleUpdateActivity,
		"HandleDeleteActivity":      routes.HandleDeleteActivity,
		"HandleCompleteActivity":    routes.HandleCompleteActivity,
		"HandleSearch":              routes.HandleSearch,
		"HandleCreateSavedFilter":   routes.HandleCreateSavedFilter,
		"HandleGetSavedFilter":      routes.HandleGetSavedFilter,
		"HandleListSavedFilters":    routes.HandleListSavedFilters,
		"HandleUpdateSavedFilter":   routes.HandleUpdateSavedFilter,
		"HandleDeleteSavedFilter":   routes.HandleDeleteSavedFilter,
		"HandleGetPipelineReport":   routes.HandleGetPipelineReport,
		"HandleGetConversionReport": routes.HandleGetConversionReport,
		"HandleGetActivityReport":   routes.HandleGetActivityReport,
		"HandleListCustomFields":    routes.HandleListCustomFields,
		"HandleUpdateCustomField":   routes.HandleUpdateCustomField,
		"HandleDeleteCustomField":   routes.HandleDeleteCustomField,
		"HandleListTags":            routes.HandleListTags,
		"HandleUpdateTag":           routes.HandleUpdateTag,
		"HandleDeleteTag":           routes.HandleDeleteTag,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// HandleCreateActivity — polymorphic assignment (contact/company/deal) each
// need their own validation case, plus the required fields and the dive,uuid
// tag_ids rule.
// ============================================================================

func TestHandleCreateActivity_MissingFields(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	t.Run("no_activity_type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", jsonBody(t, map[string]interface{}{
			"subject": "Call customer",
		}))
		req = withAuth(req, "user-123", testTenantID)
		routes.HandleCreateActivity(rec, req)
		assertValidationError(t, rec, "activity_type")
	})

	t.Run("no_subject", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", jsonBody(t, map[string]interface{}{
			"activity_type": "call",
		}))
		req = withAuth(req, "user-123", testTenantID)
		routes.HandleCreateActivity(rec, req)
		assertValidationError(t, rec, "subject")
	})
}

func TestHandleCreateActivity_InvalidContactID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", jsonBody(t, map[string]interface{}{
		"activity_type": "call",
		"subject":       "Call customer",
		"contact_id":    "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateActivity(rec, req)
	assertValidationError(t, rec, "contact_id")
}

func TestHandleCreateActivity_InvalidCompanyID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", jsonBody(t, map[string]interface{}{
		"activity_type": "call",
		"subject":       "Call customer",
		"company_id":    "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateActivity(rec, req)
	assertValidationError(t, rec, "company_id")
}

func TestHandleCreateActivity_InvalidDealID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", jsonBody(t, map[string]interface{}{
		"activity_type": "call",
		"subject":       "Call customer",
		"deal_id":       "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateActivity(rec, req)
	assertValidationError(t, rec, "deal_id")
}

func TestHandleCreateActivity_InvalidTagIDs(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", jsonBody(t, map[string]interface{}{
		"activity_type": "call",
		"subject":       "Call customer",
		"tag_ids":       []string{"not-a-uuid"},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateActivity(rec, req)
	assertValidationError(t, rec, "tag_ids[0]")
}

func TestHandleCreateActivity_NoTenant(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", jsonBody(t, map[string]interface{}{
		"activity_type": "call",
		"subject":       "Call customer",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateActivity(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing or invalid tenant")
}

// ============================================================================
// HandleGetActivity — the tenant check runs BEFORE the UUID param check, so
// exercising the invalid-UUID path needs a tenant in context too, else the
// request dies earlier with 401 instead of 400.
// ============================================================================

func TestHandleGetActivity_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/bad", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetActivity(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetActivity_NoTenant(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/"+activityTestID, nil)
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleGetActivity(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// ============================================================================
// HandleListActivities — filter combinations. Each combination is built into
// the gRPC request before the (unreachable, dummy-backend) RPC call, so this
// exercises every query-param branch without panicking; the RPC itself always
// ends in 503 here, matching the repo convention that gateway unit tests only
// verify what happens before the RPC call.
// ============================================================================

func TestHandleListActivities_FilterCombinations(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	cases := map[string]string{
		"none":               "",
		"activity_type":      "?activity_type=call",
		"contact_id":         "?contact_id=" + activityTestID,
		"company_id":         "?company_id=" + activityTestID,
		"deal_id":            "?deal_id=" + activityTestID,
		"assigned_to":        "?assigned_to=user-1",
		"is_completed_true":  "?is_completed=true",
		"is_completed_false": "?is_completed=false",
		"sort":               "?sort_by=due_date&sort_desc=true",
		"pagination":         "?page=2&page_size=50",
		"all_combined":       "?activity_type=call&contact_id=" + activityTestID + "&is_completed=true&sort_by=due_date&sort_desc=true&page=3&page_size=10",
	}

	for name, qs := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/activities"+qs, nil)
			req = withAuth(req, "user-123", testTenantID)
			routes.HandleListActivities(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// ============================================================================
// HandleUpdateActivity — the UUID check runs BEFORE decode/validate, and the
// tenant check runs AFTER decode/validate (the reverse order of Get/Delete).
// ============================================================================

func TestHandleUpdateActivity_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateActivity(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateActivity_EmptySubject(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/"+activityTestID, jsonBody(t, map[string]interface{}{
		"subject": "",
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleUpdateActivity(rec, req)
	assertValidationError(t, rec, "subject")
}

func TestHandleUpdateActivity_InvalidContactID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/"+activityTestID, jsonBody(t, map[string]interface{}{
		"contact_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleUpdateActivity(rec, req)
	assertValidationError(t, rec, "contact_id")
}

func TestHandleUpdateActivity_InvalidCompanyID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/"+activityTestID, jsonBody(t, map[string]interface{}{
		"company_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleUpdateActivity(rec, req)
	assertValidationError(t, rec, "company_id")
}

func TestHandleUpdateActivity_InvalidDealID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/"+activityTestID, jsonBody(t, map[string]interface{}{
		"deal_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleUpdateActivity(rec, req)
	assertValidationError(t, rec, "deal_id")
}

func TestHandleUpdateActivity_NoTenant(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/"+activityTestID, jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleUpdateActivity(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing or invalid tenant")
}

// ============================================================================
// HandleDeleteActivity / HandleCompleteActivity — tenant check runs BEFORE
// the UUID check (same order as Get).
// ============================================================================

func TestHandleDeleteActivity_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/activities/bad", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteActivity(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteActivity_NoTenant(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/activities/"+activityTestID, nil)
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleDeleteActivity(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCompleteActivity_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities/bad/complete", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCompleteActivity(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCompleteActivity_NoTenant(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities/"+activityTestID+"/complete", nil)
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleCompleteActivity(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// ============================================================================
// HandleAddActivityTags / HandleRemoveActivityTags — these never reach a
// gRPC client at all; they validate the URL param and body, then always
// answer 501 (documented as gRPC-only). Testing that explicitly here so the
// stub-looking 501 is a proven, intentional contract rather than an
// unverified guess.
// ============================================================================

func TestHandleAddActivityTags_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities/bad/tags", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAddActivityTags(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleAddActivityTags_InvalidTagIDs(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities/"+activityTestID+"/tags", jsonBody(t, map[string]interface{}{
		"tag_ids": []string{"not-a-uuid"},
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleAddActivityTags(rec, req)
	assertValidationError(t, rec, "tag_ids[0]")
}

func TestHandleAddActivityTags_NotImplemented(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities/"+activityTestID+"/tags", jsonBody(t, map[string]interface{}{
		"tag_ids": []string{activityTestID},
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleAddActivityTags(rec, req)
	assertStatus(t, rec, http.StatusNotImplemented)
}

func TestHandleRemoveActivityTags_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/activities/bad/tags", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleRemoveActivityTags(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleRemoveActivityTags_NotImplemented(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/activities/"+activityTestID+"/tags", jsonBody(t, map[string]interface{}{
		"tag_ids": []string{activityTestID},
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleRemoveActivityTags(rec, req)
	assertStatus(t, rec, http.StatusNotImplemented)
}

// ============================================================================
// HandleSearch
// ============================================================================

func TestHandleSearch_MissingQuery(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	routes.HandleSearch(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "query parameter 'q' is required")
}

func TestHandleSearch_LimitClamp(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	// Out-of-range and non-numeric limits fall back to the default (20)
	// instead of being forwarded as-is or panicking.
	cases := []string{"?q=max&limit=0", "?q=max&limit=-5", "?q=max&limit=1000", "?q=max&limit=abc", "?q=max&limit=50", "?q=max&types=contact&types=deal"}

	for _, qs := range cases {
		t.Run(qs, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/search"+qs, nil)
			routes.HandleSearch(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// ============================================================================
// Saved Filters
// ============================================================================

func TestHandleCreateSavedFilter_MissingFields(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	t.Run("no_name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-filters", jsonBody(t, map[string]interface{}{
			"entity_type": "contact",
			"filter_json": "{}",
		}))
		routes.HandleCreateSavedFilter(rec, req)
		assertValidationError(t, rec, "name")
	})

	t.Run("no_entity_type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-filters", jsonBody(t, map[string]interface{}{
			"name":        "My filter",
			"filter_json": "{}",
		}))
		routes.HandleCreateSavedFilter(rec, req)
		assertValidationError(t, rec, "entity_type")
	})

	t.Run("no_filter_json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-filters", jsonBody(t, map[string]interface{}{
			"name":        "My filter",
			"entity_type": "contact",
		}))
		routes.HandleCreateSavedFilter(rec, req)
		assertValidationError(t, rec, "filter_json")
	})
}

func TestHandleCreateSavedFilter_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-filters", invalidJSON())
	routes.HandleCreateSavedFilter(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleGetSavedFilter_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/saved-filters/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetSavedFilter(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateSavedFilter_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/saved-filters/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateSavedFilter(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateSavedFilter_EmptyName(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/saved-filters/"+activityTestID, jsonBody(t, map[string]interface{}{
		"name": "",
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleUpdateSavedFilter(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateSavedFilter_EmptyFilterJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/saved-filters/"+activityTestID, jsonBody(t, map[string]interface{}{
		"filter_json": "",
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleUpdateSavedFilter(rec, req)
	assertValidationError(t, rec, "filter_json")
}

func TestHandleDeleteSavedFilter_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/saved-filters/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteSavedFilter(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// ============================================================================
// Reports — HandleGetPipelineReport / HandleGetConversionReport /
// HandleGetActivityReport. All three only check that start_date/end_date are
// non-empty; there is no local date-format parsing (see the finding recorded
// in JOURNAL.md), so a syntactically invalid date is not rejected here —
// it is forwarded to the gRPC call like any other string. The tests below
// prove the required-fields check and that a garbage date does not panic.
// ============================================================================

func TestHandleGetPipelineReport_MissingDates(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	cases := map[string]string{
		"neither":    "",
		"only_start": "?start_date=2026-01-01",
		"only_end":   "?end_date=2026-01-31",
	}
	for name, qs := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/pipeline"+qs, nil)
			routes.HandleGetPipelineReport(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertErrorContains(t, rec, "start_date and end_date are required")
		})
	}
}

func TestHandleGetPipelineReport_MalformedDateNoPanic(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/pipeline?start_date=not-a-date&end_date=also-not-a-date&owner_id="+activityTestID, nil)
	routes.HandleGetPipelineReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetConversionReport_MissingDates(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/conversion", nil)
	routes.HandleGetConversionReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "start_date and end_date are required")
}

func TestHandleGetConversionReport_MalformedDateNoPanic(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/conversion?start_date=banana&end_date=banana", nil)
	routes.HandleGetConversionReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetActivityReport_MissingDates(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/activities", nil)
	routes.HandleGetActivityReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "start_date and end_date are required")
}

func TestHandleGetActivityReport_MalformedDateNoPanic(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/activities?start_date=banana&end_date=banana&user_id="+activityTestID, nil)
	routes.HandleGetActivityReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Custom Fields — the remaining handlers not already covered in
// route_crm_test.go (Create/Get are covered there).
// ============================================================================

func TestHandleListCustomFields_FilterCombinations(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	cases := []string{"", "?entity_type=contact", "?entity_type=deal&page=2&page_size=10"}
	for _, qs := range cases {
		t.Run(qs, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/custom-fields"+qs, nil)
			routes.HandleListCustomFields(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

func TestHandleUpdateCustomField_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/custom-fields/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateCustomField(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateCustomField_EmptyFieldLabel(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/custom-fields/"+activityTestID, jsonBody(t, map[string]interface{}{
		"field_label": "",
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleUpdateCustomField(rec, req)
	assertValidationError(t, rec, "field_label")
}

func TestHandleDeleteCustomField_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/custom-fields/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteCustomField(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// ============================================================================
// Tags — the remaining handlers not already covered in route_crm_test.go
// (Create/Get are covered there).
// ============================================================================

func TestHandleListTags_FilterCombinations(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	cases := []string{"", "?entity_type=contact", "?entity_type=deal&page=2&page_size=10"}
	for _, qs := range cases {
		t.Run(qs, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tags"+qs, nil)
			routes.HandleListTags(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

func TestHandleUpdateTag_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tags/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateTag(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateTag_EmptyName(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tags/"+activityTestID, jsonBody(t, map[string]interface{}{
		"name": "",
	}))
	req = withChiURLParam(req, "id", activityTestID)
	routes.HandleUpdateTag(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleDeleteTag_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteTag(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}
