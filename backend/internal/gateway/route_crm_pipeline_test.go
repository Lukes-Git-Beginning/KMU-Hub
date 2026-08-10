package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// pipelineTestID is a stable, valid UUID used wherever a handler needs a
// syntactically valid id/tenant path param but the value itself is not under
// test.
const pipelineTestID = "22222222-2222-2222-2222-222222222222"

// ============================================================================
// ServiceUnavailable — every handler in this file except the two tag-mutation
// stubs (HandleAddDealTags/HandleRemoveDealTags, which never reach a gRPC
// client — see their own tests below) resolves the CRM client first.
// ============================================================================

func TestCRMPipelineRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)

	handlers := map[string]http.HandlerFunc{
		"HandleCreatePipelineStage":   routes.HandleCreatePipelineStage,
		"HandleGetPipelineStage":      routes.HandleGetPipelineStage,
		"HandleListPipelineStages":    routes.HandleListPipelineStages,
		"HandleUpdatePipelineStage":   routes.HandleUpdatePipelineStage,
		"HandleDeletePipelineStage":   routes.HandleDeletePipelineStage,
		"HandleReorderPipelineStages": routes.HandleReorderPipelineStages,
		"HandleCreateDeal":            routes.HandleCreateDeal,
		"HandleGetDeal":               routes.HandleGetDeal,
		"HandleListDeals":             routes.HandleListDeals,
		"HandleUpdateDeal":            routes.HandleUpdateDeal,
		"HandleDeleteDeal":            routes.HandleDeleteDeal,
		"HandleMoveDealToStage":       routes.HandleMoveDealToStage,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// HandleCreatePipelineStage
// ============================================================================

func TestHandleCreatePipelineStage_MissingName(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/stages", jsonBody(t, map[string]interface{}{
		"color": "#ff0000",
	}))
	routes.HandleCreatePipelineStage(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreatePipelineStage_ValidReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/stages", jsonBody(t, map[string]interface{}{
		"name":        "Negotiation",
		"color":       "#00ff00",
		"is_won":      false,
		"is_lost":     false,
		"probability": 0.5,
	}))
	routes.HandleCreatePipelineStage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetPipelineStage / HandleDeletePipelineStage — pure id validation.
// ============================================================================

func TestHandleGetPipelineStage_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline/stages/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetPipelineStage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeletePipelineStage_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pipeline/stages/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeletePipelineStage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// ============================================================================
// HandleUpdatePipelineStage — id before body, omitempty,min=1 on name.
// ============================================================================

func TestHandleUpdatePipelineStage_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/pipeline/stages/not-a-uuid", jsonBody(t, map[string]interface{}{"name": "x"}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdatePipelineStage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdatePipelineStage_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/pipeline/stages/"+pipelineTestID, invalidJSON())
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleUpdatePipelineStage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdatePipelineStage_EmptyName(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/pipeline/stages/"+pipelineTestID, jsonBody(t, map[string]interface{}{"name": ""}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleUpdatePipelineStage(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdatePipelineStage_PartialBodyReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/pipeline/stages/"+pipelineTestID, jsonBody(t, map[string]interface{}{"color": "#123456"}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleUpdatePipelineStage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleReorderPipelineStages — the only handler with real ordering logic
// downstream; at the gateway boundary the guarantee is "non-empty, all valid
// UUIDs" (validate:"required,min=1,dive,uuid").
// ============================================================================

func TestHandleReorderPipelineStages_EmptyIDs(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/stages/reorder", jsonBody(t, map[string]interface{}{
		"stage_ids": []string{},
	}))
	routes.HandleReorderPipelineStages(rec, req)
	assertValidationError(t, rec, "stage_ids")
}

func TestHandleReorderPipelineStages_MissingField(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/stages/reorder", jsonBody(t, map[string]interface{}{}))
	routes.HandleReorderPipelineStages(rec, req)
	assertValidationError(t, rec, "stage_ids")
}

func TestHandleReorderPipelineStages_InvalidElement(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/stages/reorder", jsonBody(t, map[string]interface{}{
		"stage_ids": []string{pipelineTestID, "not-a-uuid"},
	}))
	routes.HandleReorderPipelineStages(rec, req)
	assertValidationError(t, rec, "stage_ids[1]")
}

func TestHandleReorderPipelineStages_ValidReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/stages/reorder", jsonBody(t, map[string]interface{}{
		"stage_ids": []string{pipelineTestID, "33333333-3333-3333-3333-333333333333"},
	}))
	routes.HandleReorderPipelineStages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListPipelineStages — no filters, no tenant check in this handler.
// ============================================================================

func TestHandleListPipelineStages_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline/stages", nil)
	routes.HandleListPipelineStages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateDeal — tenant check runs BEFORE decodeAndValidate here (unlike
// HandleUpdateDeal/HandleMoveDealToStage below, where it runs after).
// ============================================================================

func TestHandleCreateDeal_NoTenant(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", jsonBody(t, map[string]interface{}{
		"name":     "Big Deal",
		"stage_id": pipelineTestID,
	}))
	routes.HandleCreateDeal(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateDeal_MissingFields(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	t.Run("no_name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", jsonBody(t, map[string]interface{}{
			"stage_id": pipelineTestID,
		}))
		req = withAuth(req, "user-1", testTenantID)
		routes.HandleCreateDeal(rec, req)
		assertValidationError(t, rec, "name")
	})

	t.Run("no_stage_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", jsonBody(t, map[string]interface{}{
			"name": "Big Deal",
		}))
		req = withAuth(req, "user-1", testTenantID)
		routes.HandleCreateDeal(rec, req)
		assertValidationError(t, rec, "stage_id")
	})

	t.Run("invalid_stage_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", jsonBody(t, map[string]interface{}{
			"name":     "Big Deal",
			"stage_id": "not-a-uuid",
		}))
		req = withAuth(req, "user-1", testTenantID)
		routes.HandleCreateDeal(rec, req)
		assertValidationError(t, rec, "stage_id")
	})

	t.Run("invalid_contact_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", jsonBody(t, map[string]interface{}{
			"name":       "Big Deal",
			"stage_id":   pipelineTestID,
			"contact_id": "not-a-uuid",
		}))
		req = withAuth(req, "user-1", testTenantID)
		routes.HandleCreateDeal(rec, req)
		assertValidationError(t, rec, "contact_id")
	})

	t.Run("invalid_company_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", jsonBody(t, map[string]interface{}{
			"name":       "Big Deal",
			"stage_id":   pipelineTestID,
			"company_id": "not-a-uuid",
		}))
		req = withAuth(req, "user-1", testTenantID)
		routes.HandleCreateDeal(rec, req)
		assertValidationError(t, rec, "company_id")
	})

	t.Run("invalid_owner_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", jsonBody(t, map[string]interface{}{
			"name":     "Big Deal",
			"stage_id": pipelineTestID,
			"owner_id": "not-a-uuid",
		}))
		req = withAuth(req, "user-1", testTenantID)
		routes.HandleCreateDeal(rec, req)
		assertValidationError(t, rec, "owner_id")
	})

	t.Run("invalid_tag_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", jsonBody(t, map[string]interface{}{
			"name":     "Big Deal",
			"stage_id": pipelineTestID,
			"tag_ids":  []string{"not-a-uuid"},
		}))
		req = withAuth(req, "user-1", testTenantID)
		routes.HandleCreateDeal(rec, req)
		assertValidationError(t, rec, "tag_ids[0]")
	})
}

func TestHandleCreateDeal_ValidWithCustomFieldsReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", jsonBody(t, map[string]interface{}{
		"name":     "Big Deal",
		"stage_id": pipelineTestID,
		"custom_fields": []map[string]interface{}{
			{"field_id": pipelineTestID, "value": "42"},
		},
	}))
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleCreateDeal(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetDeal — tenant check runs BEFORE the id check.
// ============================================================================

func TestHandleGetDeal_NoTenant(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/deals/"+pipelineTestID, nil)
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleGetDeal(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetDeal_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/deals/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetDeal(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetDeal_ValidReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/deals/"+pipelineTestID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleGetDeal(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListDeals — filter combinations, tenant check before parsing filters.
// ============================================================================

func TestHandleListDeals_NoTenant(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/deals", nil)
	routes.HandleListDeals(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListDeals_FilterCombinations(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	cases := map[string]string{
		"empty":         "",
		"stage_id":      "?stage_id=" + pipelineTestID,
		"contact_id":    "?contact_id=" + pipelineTestID,
		"company_id":    "?company_id=" + pipelineTestID,
		"owner_id":      "?owner_id=" + pipelineTestID,
		"search":        "?search=acme",
		"sort":          "?sort_by=value&sort_desc=true",
		"tag_ids":       "?tag_ids=" + pipelineTestID + "&tag_ids=33333333-3333-3333-3333-333333333333",
		"pagination":    "?page=2&page_size=50",
		"all_combined":  "?stage_id=" + pipelineTestID + "&contact_id=" + pipelineTestID + "&company_id=" + pipelineTestID + "&owner_id=" + pipelineTestID + "&search=acme&sort_by=value&sort_desc=true&tag_ids=" + pipelineTestID + "&page=1&page_size=10",
		"bad_pagination": "?page=not-a-number&page_size=not-a-number",
	}

	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/deals"+query, nil)
			req = withTenantID(req, testTenantID)
			routes.HandleListDeals(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// ============================================================================
// HandleUpdateDeal — id check, then body decode, then tenant check LAST
// (opposite order from HandleGetDeal/HandleListDeals/HandleDeleteDeal above).
// ============================================================================

func TestHandleUpdateDeal_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/deals/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateDeal(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateDeal_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/deals/"+pipelineTestID, invalidJSON())
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleUpdateDeal(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateDeal_EmptyName(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/deals/"+pipelineTestID, jsonBody(t, map[string]interface{}{"name": ""}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleUpdateDeal(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateDeal_InvalidOptionalIDs(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	t.Run("invalid_contact_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/deals/"+pipelineTestID, jsonBody(t, map[string]interface{}{"contact_id": "not-a-uuid"}))
		req = withChiURLParam(req, "id", pipelineTestID)
		routes.HandleUpdateDeal(rec, req)
		assertValidationError(t, rec, "contact_id")
	})

	t.Run("invalid_company_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/deals/"+pipelineTestID, jsonBody(t, map[string]interface{}{"company_id": "not-a-uuid"}))
		req = withChiURLParam(req, "id", pipelineTestID)
		routes.HandleUpdateDeal(rec, req)
		assertValidationError(t, rec, "company_id")
	})

	t.Run("invalid_owner_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/deals/"+pipelineTestID, jsonBody(t, map[string]interface{}{"owner_id": "not-a-uuid"}))
		req = withChiURLParam(req, "id", pipelineTestID)
		routes.HandleUpdateDeal(rec, req)
		assertValidationError(t, rec, "owner_id")
	})
}

func TestHandleUpdateDeal_NoTenantAfterValidBody(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/deals/"+pipelineTestID, jsonBody(t, map[string]interface{}{"notes": "called back"}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleUpdateDeal(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateDeal_ValidReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/deals/"+pipelineTestID, jsonBody(t, map[string]interface{}{"notes": "called back"}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleUpdateDeal(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteDeal — tenant check runs BEFORE the id check (same order as
// HandleGetDeal, opposite of HandleUpdateDeal/HandleMoveDealToStage).
// ============================================================================

func TestHandleDeleteDeal_NoTenant(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deals/"+pipelineTestID, nil)
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleDeleteDeal(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteDeal_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deals/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteDeal(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteDeal_ValidReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deals/"+pipelineTestID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleDeleteDeal(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleMoveDealToStage — id check, then body decode, then tenant check LAST
// (same order as HandleUpdateDeal).
// ============================================================================

func TestHandleMoveDealToStage_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals/not-a-uuid/move", jsonBody(t, map[string]interface{}{"stage_id": pipelineTestID}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleMoveDealToStage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleMoveDealToStage_MissingStageID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals/"+pipelineTestID+"/move", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleMoveDealToStage(rec, req)
	assertValidationError(t, rec, "stage_id")
}

func TestHandleMoveDealToStage_InvalidStageID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals/"+pipelineTestID+"/move", jsonBody(t, map[string]interface{}{"stage_id": "not-a-uuid"}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleMoveDealToStage(rec, req)
	assertValidationError(t, rec, "stage_id")
}

func TestHandleMoveDealToStage_NoTenantAfterValidBody(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals/"+pipelineTestID+"/move", jsonBody(t, map[string]interface{}{"stage_id": pipelineTestID}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleMoveDealToStage(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleMoveDealToStage_ValidReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals/"+pipelineTestID+"/move", jsonBody(t, map[string]interface{}{"stage_id": pipelineTestID}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleMoveDealToStage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleAddDealTags / HandleRemoveDealTags — stubs that validate id and body
// but never reach a gRPC client; they always answer 501.
// ============================================================================

func TestHandleAddDealTags_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals/not-a-uuid/tags", jsonBody(t, map[string]interface{}{"tag_ids": []string{}}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAddDealTags(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleAddDealTags_InvalidTagID(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals/"+pipelineTestID+"/tags", jsonBody(t, map[string]interface{}{"tag_ids": []string{"not-a-uuid"}}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleAddDealTags(rec, req)
	assertValidationError(t, rec, "tag_ids[0]")
}

func TestHandleAddDealTags_ValidReturnsNotImplemented(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals/"+pipelineTestID+"/tags", jsonBody(t, map[string]interface{}{"tag_ids": []string{pipelineTestID}}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleAddDealTags(rec, req)
	assertStatus(t, rec, http.StatusNotImplemented)
	assertErrorContains(t, rec, "not implemented via HTTP, use gRPC")
}

func TestHandleRemoveDealTags_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deals/not-a-uuid/tags", jsonBody(t, map[string]interface{}{"tag_ids": []string{}}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleRemoveDealTags(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleRemoveDealTags_InvalidTagID(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deals/"+pipelineTestID+"/tags", jsonBody(t, map[string]interface{}{"tag_ids": []string{"not-a-uuid"}}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleRemoveDealTags(rec, req)
	assertValidationError(t, rec, "tag_ids[0]")
}

func TestHandleRemoveDealTags_ValidReturnsNotImplemented(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deals/"+pipelineTestID+"/tags", jsonBody(t, map[string]interface{}{"tag_ids": []string{pipelineTestID}}))
	req = withChiURLParam(req, "id", pipelineTestID)
	routes.HandleRemoveDealTags(rec, req)
	assertStatus(t, rec, http.StatusNotImplemented)
	assertErrorContains(t, rec, "not implemented via HTTP, use gRPC")
}
